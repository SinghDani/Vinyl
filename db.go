package main

import (
	"database/sql"
	"errors"
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

func (db *DBConnection) ContainsSong(song string) (bool, error) {
	var exists int
	row := db.db.QueryRow("SELECT 1 FROM songs WHERE songname = $1", song)
	if err := row.Scan(&exists); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (db *DBConnection) StoreSong(songname string) error {
	contains, err := db.ContainsSong(songname)
	if contains {
		return errors.New("song already in db")
	} else if err != nil {
		return err
	}

	_, err = db.db.Exec("INSERT INTO songs (songname) VALUES($1)", songname)
	return err
}
