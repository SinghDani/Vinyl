package main

import (
	"context"
	"log"
	"os"
	"path/filepath"

	"github.com/SinghDani/audioRecognition/api"
	"github.com/SinghDani/audioRecognition/db"
	"github.com/SinghDani/audioRecognition/fingerprint"
	"github.com/joho/godotenv"
)

func main() {
	action := "record"
	if len(os.Args) > 1 {
		action = os.Args[1]
	}

	if err := godotenv.Load("./.env"); err != nil {
		_ = godotenv.Load("../.env")
	}

	db, err := db.NewDBConnection()
	if err != nil {
		log.Fatal(err)
	}
	defer db.CloseDBConnection()

	if action == "store" {
		if len(os.Args) < 3 {
			log.Fatal("not enough arguments")
		}
		file := os.Args[2]
		genHashes, err := fingerprint.ExtractHashesFromFile(file, 0)
		if err != nil {
			log.Fatal(err)
		}

		if err := db.StoreHashes(context.Background(), genHashes, filepath.Base(file)); err != nil {
			log.Fatal(err)
		}
	} else if action == "record" {
		port := os.Getenv("PORT")
		if port == "" {
			port = "3000"
		}
		server := api.NewServer(":"+port, db)
		server.Run()
	}
}
