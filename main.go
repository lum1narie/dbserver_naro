package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/srinathgs/mysqlstore"
	"golang.org/x/crypto/bcrypt"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

var (
	db *sqlx.DB
)

type (
	LoginRequestBody struct {
		Username string `json:"username,omitempty" form:"username"`
		Password string `json:"password,omitempty" form:"password"`
	}

	User struct {
		Username   string `json:"username,omitempty"  db:"Username"`
		HashedPass string `json:"-"  db:"HashedPass"`
	}

	City struct {
		ID          int    `json:"id" db:"ID"`
		Name        string `json:"name" db:"Name"`
		CountryCode string `json:"countryCode" db:"CountryCode"`
		District    string `json:"district,omitempty" db:"District"`
		Population  int    `json:"population,omitempty" db:"Population"`
	}

	Country struct {
		Code           string          `db:"Code"`
		Name           string          `db:"Name"`
		Continent      string          `db:"Continent"`
		Region         string          `db:"Region"`
		SurfaceArea    float64         `db:"SurfaceArea"`
		IndepYear      sql.NullInt32   `db:"IndepYear"`
		Population     int             `db:"Population"`
		LifeExpectancy sql.NullFloat64 `db:"LifeExpectancy"`
		GNP            sql.NullFloat64 `db:"GNP"`
		GNPOld         sql.NullFloat64 `db:"GNPOld"`
		LocalName      string          `db:"LocalName"`
		GovernmentForm string          `db:"GovernmentForm"`
		HeadOfState    sql.NullString  `db:"HeadOfState"`
		Capital        sql.NullInt32   `db:"Capital"`
		Code2          string          `db:"Code2"`
	}

	CountryNamePop struct {
		Code       string `json:"countryCode,omitempty" db:"Code"`
		Name       string `json:"name,omitempty" db:"Name"`
		Population int    `json:"population,omitempty" db:"Population"`
	}

	CityPopulationResponse struct {
		Name       string
		Population int
		Ratio      float64
	}

	WhoAmIResponse struct {
		Username string `json:"username,omitempty"  db:"username"`
	}

	CountryNameResponse struct {
		Name string
	}

	CityNameResponse struct {
		Name string
	}
)

func initDB() *sqlx.DB {
	_db, err := sqlx.Connect("mysql", fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?charset=utf8&parseTime=True&loc=Local",
		os.Getenv("DB_USERNAME"), os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOSTNAME"), os.Getenv("DB_PORT"),
		os.Getenv("DB_DATABASE")))
	if err != nil {
		log.Fatalf("Cannot Connect to Database: %s", err)
	}

	// fmt.Println("Connected!")
	return _db
}

func initSession() sessions.Store {
	store, err := mysqlstore.NewMySQLStoreFromConnection(
		db.DB, "sessions", "/", 60*60*24*14, []byte("secret-token"))
	if err != nil {
		panic(err)
	}
	return store
}

func postSignUpHandler(c echo.Context) error {
	req := &LoginRequestBody{}
	if err := c.Bind(req); err != nil {
		return c.String(http.StatusBadRequest, "Bad request")
	}

	// TODO: ?????????????????????????????????????????????????????????
	if req.Password == "" || req.Username == "" {
		return c.String(http.StatusBadRequest, "??????????????????")
	}

	hashedPass, hashErr := bcrypt.GenerateFromPassword(
		[]byte(req.Password), bcrypt.DefaultCost)
	if hashErr != nil {
		return c.String(http.StatusInternalServerError,
			fmt.Sprintf("bcrypt genetrate error: %v", hashErr))
	}

	// ?????????????????????????????????
	var count int

	countQuery := "select count(*) from users where Username=?"
	if queryErr := db.Get(&count, countQuery, req.Username); queryErr != nil {
		return c.String(http.StatusInternalServerError,
			fmt.Sprintf("db error: %v", queryErr))
	}
	if count > 0 {
		return c.String(http.StatusConflict, "??????????????????????????????????????????")
	}

	addUserQuery := "insert into users (Username, HashedPass) values(?, ?)"
	if _, queryErr := db.Exec(
		addUserQuery, req.Username, hashedPass); queryErr != nil {
		return c.String(http.StatusInternalServerError,
			fmt.Sprintf("db error: %v", queryErr))
	}

	return c.NoContent(http.StatusCreated)
}

func postLoginHandler(c echo.Context) error {
	req := &LoginRequestBody{}
	if err := c.Bind(req); err != nil {
		return c.String(http.StatusBadRequest, "Bad request")
	}

	user := &User{}
	userQuery := "select * from users where username=?"
	if queryErr := db.Get(user, userQuery, req.Username); queryErr != nil {
		return c.String(http.StatusInternalServerError,
			fmt.Sprintf("db error: %v", queryErr))
	}

	if hashErr := bcrypt.CompareHashAndPassword(
		[]byte(user.HashedPass), []byte(req.Password)); hashErr != nil {
		if errors.Is(hashErr, bcrypt.ErrMismatchedHashAndPassword) {
			return c.NoContent(http.StatusForbidden)
		} else {
			return c.NoContent(http.StatusInternalServerError)
		}
	}

	sess, sessErr := session.Get("sessions", c)
	if sessErr != nil {
		fmt.Println(sessErr)
		return c.String(http.StatusInternalServerError,
			"something wrong in getting session")
	}
	sess.Values["userName"] = req.Username
	sess.Save(c.Request(), c.Response())

	return c.NoContent(http.StatusOK)
}

func checkLogin(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		sess, err := session.Get("sessions", c)
		if err != nil {
			fmt.Println(err)
			return c.String(http.StatusInternalServerError,
				"something wrong in getting session")
		}

		if sess.Values["userName"] == nil {
			return c.String(http.StatusForbidden, "please login")
		}
		c.Set("userName", sess.Values["userName"].(string))

		return next(c)
	}
}

func getCity(name string) (*City, error) {
	city := &City{}

	query := "select * from city where name = ?"
	if err := db.Get(city, query, name); errors.Is(err, sql.ErrNoRows) {
		log.Printf("no such city Name %s\n", name)
		return nil, err
	} else if err != nil {
		log.Fatalf("DB Error: %s", err)
		return nil, err
	}

	return city, nil
}

func getCountries() (*[]Country, error) {
	countries := &[]Country{}
	query := "select * from country"

	if err := db.Select(countries, query); err != nil {
		log.Fatalf("DB Error: %s", err)
		return nil, err
	}

	return countries, nil
}

func getCountryByName(name string) (*Country, error) {
	country := &Country{}
	query := "select * from country where Name = ?"

	if err := db.Get(country, query, name); errors.Is(err, sql.ErrNoRows) {
		log.Printf("no country %s found \n", country.Name)
		return nil, err
	} else if err != nil {
		log.Fatalf("DB Error: %s", err)
		return nil, err
	}

	return country, nil
}

func getCountryNamePop(code string) CountryNamePop {
	countryNamePop := CountryNamePop{}

	query := "select Code, Name, Population from country where code = ?"
	if err := db.Get(
		&countryNamePop, query, code); errors.Is(err, sql.ErrNoRows) {
		log.Printf("no such country Code %s\n", code)
	} else if err != nil {
		log.Fatalf("DB Error: %s", err)
	}

	return countryNamePop
}

func getCitiesByCountry(country *Country) (*[]City, error) {
	cities := &[]City{}

	query := "select * from city where CountryCode = ?"
	if err := db.Select(cities, query, country.Code); errors.Is(err, sql.ErrNoRows) {
		log.Printf("no city found in country %s\n", country.Name)
		return nil, err
	} else if err != nil {
		log.Fatalf("DB Error: %s", err)
		return nil, err
	}

	return cities, nil
}

func getCityPopulationHandler(c echo.Context) error {
	target_city := c.Param("cityName")

	city, _ := getCity(target_city)
	fmt.Printf("%s????????????%d?????????\n", target_city, city.Population)

	countryNamePop := getCountryNamePop(city.CountryCode)

	ratio := float64(city.Population) / float64(countryNamePop.Population)
	fmt.Printf("%s????????????%s????????????%f%%??????\n",
		target_city, countryNamePop.Name, ratio*100.0)
	return c.JSON(http.StatusOK, CityPopulationResponse{
		Name:       target_city,
		Population: city.Population,
		Ratio:      ratio,
	})
}

func getCityInfoHandler(c echo.Context) error {
	target_city := c.Param("cityName")

	city, err := getCity(target_city)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.String(http.StatusBadRequest, "no city found")
		} else {
			return c.String(http.StatusInternalServerError,
				"something went wrong")
		}
	}

	return c.JSON(http.StatusOK, *city)
}

func postCityHandler(c echo.Context) error {
	new_city := &City{}
	if err := c.Bind(new_city); err != nil {
		return c.String(http.StatusBadRequest, "Bad request")
	}

	query := `
insert into
    city (Name, CountryCode, District, Population)
values
    (:Name, :CountryCode, :District, :Population)
	`
	if _, err := db.NamedExec(query, new_city); err != nil {
		log.Fatalf("DB Error: %s", err)
	}

	return c.String(http.StatusOK, "")
}

func getWhoAmIHandler(c echo.Context) error {
	username := c.Get("userName").(string)
	return c.JSON(http.StatusOK, WhoAmIResponse{
		Username: username,
	})
}

func getCountriesHandler(c echo.Context) error {
	countries, err := getCountries()
	if err != nil {
		return c.String(http.StatusInternalServerError,
			"something went wrong")
	}

	response := []CountryNameResponse{}
	for _, country := range *countries {
		response = append(response, CountryNameResponse{
			Name: country.Name,
		})
	}

	sort.SliceStable(response,
		func(i, j int) bool { return response[i].Name < response[j].Name })

	return c.JSON(http.StatusOK, response)
}

func getCountryCitiesHandler(c echo.Context) error {
	countryName := c.Param("countryName")

	country, err := getCountryByName(countryName)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.String(http.StatusBadRequest, "no cities found")
		} else {
			return c.String(http.StatusInternalServerError,
				"something went wrong")
		}
	}

	cities, err := getCitiesByCountry(country)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.String(http.StatusBadRequest, "no cities found")
		} else {
			return c.String(http.StatusInternalServerError,
				"something went wrong")
		}
	}

	response := []CityNameResponse{}
	for _, city := range *cities {
		response = append(response, CityNameResponse{
			Name: city.Name,
		})
	}

	sort.SliceStable(response,
		func(i, j int) bool { return response[i].Name < response[j].Name })

	return c.JSON(http.StatusOK, response)
}

func main() {
	db = initDB()
	store := initSession()

	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(session.Middleware(store))

	e.GET("/ping", func(c echo.Context) error {
		return c.String(http.StatusOK, "pong")
	})
	e.POST("/login", postLoginHandler)
	e.POST("/signup", postSignUpHandler)

	withLogin := e.Group("")
	withLogin.Use(checkLogin)

	withLogin.GET("/cities/:cityName", getCityInfoHandler)
	withLogin.GET("/whoami", getWhoAmIHandler)
	withLogin.GET("/countries", getCountriesHandler)
	withLogin.GET("/country/:countryName/cities", getCountryCitiesHandler)

	e.Start(":10101")
}
