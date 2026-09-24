package db

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"

	"github.com/SinghDani/audioRecognition/internal"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DBConnection struct {
	db *pgxpool.Pool
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

	ctx := context.Background()
	db, err := pgxpool.New(ctx, connection)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(ctx); err != nil {
		return nil, err
	}
	return &DBConnection{db}, nil
}

func (db *DBConnection) CloseDBConnection() {
	db.db.Close()
}

func (db *DBConnection) ContainsSong(ctx context.Context, songname string) (bool, error) {
	var exists bool
	row := db.db.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM songs WHERE name = $1)", songname)
	if err := row.Scan(&exists); err != nil {
		return false, err
	}
	return exists, nil
}

func (db *DBConnection) RemoveSong(ctx context.Context, songname string) error {
	_, err := db.db.Exec(ctx, "DELETE FROM songs WHERE name = $1", songname)
	return err
}

func (db *DBConnection) GetSong(ctx context.Context, songId uint32) (internal.Song, error) {
	row := db.db.QueryRow(ctx, "SELECT name, artist FROM songs WHERE id = $1", int32(songId))
	var songName string
	var artist string
	if err := row.Scan(&songName, &artist); err != nil {
		return internal.Song{}, err
	}
	return internal.Song{Name: songName, Artist: artist}, nil
}

func (db *DBConnection) GetAllSongs(ctx context.Context) ([]internal.Song, error) {
	rows, err := db.db.Query(ctx, "SELECT name, artist FROM songs")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	songs := make([]internal.Song, 0)
	var songName string
	var artist string
	for rows.Next() {
		if err := rows.Scan(&songName, &artist); err != nil {
			return nil, err
		}
		songs = append(songs, internal.Song{Name: songName, Artist: artist})
	}
	return songs, nil
}

func (db *DBConnection) StoreHashes(ctx context.Context, hashes []internal.GeneratedHash, song string) error {
	if len(hashes) == 0 {
		return errors.New("no hashes to store")
	}

	//Todo: Document the required "Song Name - Artist.wav" filename format in the README
	r := regexp.MustCompile(`^(.+) - (.+)\.wav$`)
	songInfo := r.FindStringSubmatch(song)
	if songInfo == nil {
		return errors.New("wrong file format")
	}

	tx, err := db.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var id int32
	//since name is unique will return err if song already in db
	row := tx.QueryRow(ctx, "INSERT INTO songs (name, artist) VALUES($1, $2) RETURNING id", songInfo[1], songInfo[2])
	if err := row.Scan(&id); err != nil {
		return err
	}

	_, err = tx.CopyFrom(ctx,
		pgx.Identifier{"fingerprints"},
		[]string{"hash", "song_id", "anchor_time"},
		pgx.CopyFromSlice(len(hashes), func(i int) ([]any, error) {
			return []any{int32(hashes[i].Hash), id, int32(hashes[i].AnchorTime)}, nil
		}),
	)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (db *DBConnection) ExtractHashes(ctx context.Context, hashes []internal.GeneratedHash) ([]internal.Fingerprint, error) {
	if len(hashes) == 0 {
		return nil, nil
	}
	intHashes := make([]int32, len(hashes))
	for i, h := range hashes {
		intHashes[i] = int32(h.Hash)
	}

	var matches []internal.Fingerprint
	rows, err := db.db.Query(ctx, "SELECT hash, song_id, anchor_time from fingerprints WHERE hash = ANY($1)", intHashes)
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
