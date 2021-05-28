package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

type City struct {
	ID          int    `json:"id, omitempty" db:"ID"`
	Name        string `json:"name,omitempty"  db:"Name"`
	CountryCode string `json:"countryCode,omitempty"  db:"CountryCode"`
	District    string `json:"district,omitempty"  db:"District"`
	Population  int    `json:"population,omitempty"  db:"Population"`
}

func main() {
	target_city := "Tokyo"

	if len(os.Args) >= 2 {
		target_city = os.Args[1]
	}

	db, err := sqlx.Connect("mysql", fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?charset=utf8&parseTime=True&loc=Local",
		os.Getenv("DB_USERNAME"), os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOSTNAME"), os.Getenv("DB_PORT"),
		os.Getenv("DB_DATABASE")))

	if err != nil {
		log.Fatalf("Cannot Connect to Database: %s", err)
	}

	fmt.Println("Connected!")
	var city City
	if err := db.Get(&city, "select * from city where name = ?", target_city); errors.Is(err, sql.ErrNoRows) {
		log.Printf("no such city Name %s\n", target_city)
	} else if err != nil {
		log.Fatalf("DB Error: %s", err)
	}

	fmt.Printf("%sの人口は%d人です\n", target_city, city.Population)
}
