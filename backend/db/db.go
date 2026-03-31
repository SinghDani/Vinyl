package db

import (
	"database/sql"
	"errors"
	"fmt"
	"os"

	"github.com/SinghDani/audioRecognition/internal"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type DBConnection struct {
	db *sql.DB
}

func NewDBConnection() (*DBConnection, error) {
	connection := fmt.Sprintf("user=%s password=%s host=%s port=%s database=%s sslmode=%s",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_SSLMODE"),
	)

	db, err := sql.Open("pgx", connection)
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

func (db *DBConnection) GetSong(songId uint32) (string, error) {
	row := db.db.QueryRow("SELECT name FROM songs WHERE id = $1", int32(songId))
	var song string
	err := row.Scan(&song)
	return song, err
}

func (db *DBConnection) StoreHashes(hashes []internal.GeneratedHash, songname string) error {
	if len(hashes) == 0 {
		return errors.New("no hashes to store")
	}
	tx, err := db.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var id int32
	//since name is unique will return err if song already in db
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
		if _, err := stmt.Exec(int32(hash.Hash), id, int32(hash.AnchorTime)); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (db *DBConnection) ExtractHashes(hashes []uint32) ([]internal.Fingerprint, error) {
	if len(hashes) == 0 {
		return nil, nil
	}
	intHashes := make([]int32, len(hashes))
	for i, h := range hashes {
		intHashes[i] = int32(h)
	}

	var matches []internal.Fingerprint
	rows, err := db.db.Query("SELECT hash, song_id, anchor_time from fingerprints WHERE hash = ANY($1)", intHashes)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var hash int32
		var songId int32
		var anchorTime int32
		if err := rows.Scan(&hash, &songId, &anchorTime); err != nil {
			return nil, err
		}
		matches = append(matches, internal.Fingerprint{Hash: uint32(hash), SongId: uint32(songId), AnchorTime: uint32(anchorTime)})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return matches, nil
}
