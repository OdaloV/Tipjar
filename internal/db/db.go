package db

import (
	"database/sql"
	"log"
	"os"

	_ "github.com/lib/pq"
)

var DB *sql.DB

func Connect() {
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		log.Fatal("DB_URL is not set")
	}

	var err error
	DB, err = sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("Error opening database: ", err)
	}

	err = DB.Ping()
	if err != nil {
		log.Fatal("Error connecting to database: ", err)
	}

	log.Println("Database connected successfully")
}
