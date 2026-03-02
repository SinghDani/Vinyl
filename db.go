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

func (db *DBConnection) ContainsSong(songname string) (bool, error) {
	var exists bool
	row := db.db.QueryRow("SELECT EXISTS (SELECT 1 FROM songs WHERE name = $1)", songname)
	if err := row.Scan(&exists); err != nil {
		return false, err
	}
	return exists, nil
}

func (db *DBConnection) RemoveSong(songname string) error {
	_, err := db.db.Exec("DELETE FROM songs WHERE name = $1", songname)
	return err
}

func (db *DBConnection) StoreSong(songname string) (int, error) {
	//since name is unique will return err if song already in db
	var id int
	row := db.db.QueryRow("INSERT INTO songs (name) VALUES($1) RETURNING id", songname)
	if err := row.Scan(&id); err != nil {
		return 0, err
	}
	return id, nil
}

func (db *DBConnection) StoreHashes(hashes []hashEntry, songname string) error {
	id, err := db.StoreSong(songname)
	if err != nil {
		return err
	}
	for _, hash := range hashes {
		_, err := db.db.Exec("INSERT INTO hashes (hash, song_id, anchor_time) VALUES ($1, $2, $3)", hash.hash, id, hash.anchorTime)
		if err != nil {
			db.RemoveSong(songname)
			return err
		}
	}
	return nil
}
