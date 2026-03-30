package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/SinghDani/audioRecognition/audio"
	"github.com/SinghDani/audioRecognition/db"
	"github.com/SinghDani/audioRecognition/fingerprint"
	"github.com/joho/godotenv"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("not enough arguments")
	}
	_ = godotenv.Load()

	db, err := db.NewDBConnection()
	if err != nil {
		log.Fatal(err)
	}
	defer db.CloseDBConnection()

	store := os.Args[1] == "1"

	if store {
		if len(os.Args) < 3 {
			log.Fatal("not enough arguments")
		}
		file := os.Args[2]
		genHashes, err := fingerprint.ExtractHashesFromFile(file, 0)
		if err != nil {
			log.Fatal(err)
		}

		if err := db.StoreHashes(genHashes, filepath.Base(file)); err != nil {
			log.Fatal(err)
		}
	} else {
		if err := audio.RecordAudio(db); err != nil {
			log.Fatal(err)
		}
	}
}
