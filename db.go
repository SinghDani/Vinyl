package main

import (
	"database/sql"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type DBConnection struct {
	db *sql.DB
}

func NewDBConnection() (*DBConnection, error) {
	db, err := sql.Open("pgx", os.Getenv("CONNECTIONSTRING"))
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return &DBConnection{db}, nil
}

func (db *DBConnection) CloseDBConnection() {
	db.db.Close()
}
