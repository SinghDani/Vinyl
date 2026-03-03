package main

import (
	"database/sql"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type Fingerprint struct {
	Hash       uint32 // anchor freq | target freq | dt
	AnchorTime uint32
	SongId     uint32
}

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

func (db *DBConnection) CloseDBConnection() error {
	return db.db.Close()
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

func (db *DBConnection) StoreHashes(hashes []GeneratedHash, songname string) error {
	tx, err := db.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var id int
	row := tx.QueryRow("INSERT INTO songs (name) VALUES($1) RETURNING id", songname)
	if err := row.Scan(&id); err != nil {
		return err
	}

	stmt, err := tx.Prepare("INSERT INTO fingerprints (hash, song_id, anchor_time) VALUES($1, $2, $3)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, hash := range hashes {
		if _, err := stmt.Exec(hash.Hash, id, hash.AnchorTime); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (db *DBConnection) ExtractHashes() ([]Fingerprint, error) {
	return nil, nil
}
