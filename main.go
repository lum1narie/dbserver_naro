package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"

	"github.com/labstack/echo/v4"
)

var (
	db *sqlx.DB
)

type (
	City struct {
		ID          int    `json:"id,omitempty" db:"ID"`
		Name        string `json:"name,omitempty" db:"Name"`
		CountryCode string `json:"countryCode,omitempty" db:"CountryCode"`
		District    string `json:"district,omitempty" db:"District"`
		Population  int    `json:"population,omitempty" db:"Population"`
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
)

func initDB() {
	var err error
	db, err = sqlx.Connect("mysql", fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?charset=utf8&parseTime=True&loc=Local",
		os.Getenv("DB_USERNAME"), os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOSTNAME"), os.Getenv("DB_PORT"),
		os.Getenv("DB_DATABASE")))

	if err != nil {
		log.Fatalf("Cannot Connect to Database: %s", err)
	}

	fmt.Println("Connected!")
}

func getCity(name string) City {
	var city City
	if err := db.Get(&city, "select * from city where name = ?", name); errors.Is(err, sql.ErrNoRows) {
		log.Printf("no such city Name %s\n", name)
	} else if err != nil {
		log.Fatalf("DB Error: %s", err)
	}

	return city
}

func getCountryNamePop(code string) CountryNamePop {
	var countryNamePop CountryNamePop
	if err := db.Get(&countryNamePop, "select Code, Name, Population from country where code = ?", code); errors.Is(err, sql.ErrNoRows) {
		log.Printf("no such country Code %s\n", code)
	} else if err != nil {
		log.Fatalf("DB Error: %s", err)
	}

	return countryNamePop
}

func getCityPopulationHandler(c echo.Context) error {
	target_city := c.Param("cityName")

	city := getCity(target_city)
	fmt.Printf("%sの人口は%d人です\n", target_city, city.Population)

	countryNamePop := getCountryNamePop(city.CountryCode)

	ratio := float64(city.Population) / float64(countryNamePop.Population)
	fmt.Printf("%sの人口は%sの人口の%f%%です\n", target_city, countryNamePop.Name, ratio*100.0)
	return c.JSON(http.StatusOK, CityPopulationResponse{
		Name:       target_city,
		Population: city.Population,
		Ratio:      ratio,
	})
}

func getCityInfoHandler(c echo.Context) error {
	target_city := c.Param("cityName")

	city := getCity(target_city)
	return c.JSON(http.StatusOK, city)
}

func main() {
	initDB()

	e := echo.New()

	e.GET("/cities/:cityName", getCityInfoHandler)
	e.GET("/cities/:cityName/population", getCityPopulationHandler)

	e.Start(":10101")

}
