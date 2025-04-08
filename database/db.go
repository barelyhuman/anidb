package database

import (
	"database/sql"
	"log"

	"github.com/barelyhuman/go/env"
	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

func init() {
	var err error
	db, err = sql.Open("sqlite3", env.Get("DATABASE_URL", "data.sqlite3"))
	if err != nil {
		log.Fatalf("Failed to open database with error: %v", err)
	}
	if err = db.Ping(); err != nil {
		log.Fatalf("Failed to connect to database")
	}
}

func GetDB() *sql.DB {
	return db
}
