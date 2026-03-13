package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("not enough arguments")
	}
	if err := godotenv.Load(); err != nil {
		log.Fatal(err)
	}

	db, err := NewDBConnection()
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
		genHashes, err := ExtractHashesFromFile(file, 0)
		if err != nil {
			log.Fatal(err)
		}

		if err := db.StoreHashes(genHashes, filepath.Base(file)); err != nil {
			log.Fatal(err)
		}
	} else {
		file := "recording.wav"
		recordingBatchTime := 5 //records in batches of 5 sec
		var masterHashList []GeneratedHash
		var matchingSong MatchingSong
		var match bool
		for i := range 8 { //record for max of 40 sec
			os.Remove(file)
			timeOffset := i * SecondsToWindows(float64(recordingBatchTime))
			if err := recordAudio(file, recordingBatchTime); err != nil {
				log.Fatal(err)
			}

			genHashes, err := ExtractHashesFromFile(file, timeOffset) //shift current anchor points by 0, 5, 10, ... sec
			if err != nil {
				log.Fatal(err)
			}
			masterHashList = append(masterHashList, genHashes...)

			matchingSong, err = IdentifyRecording(db, masterHashList)
			if err != nil {
				log.Fatal(err)
			}
			match = EvalMatch(matchingSong)
			if match {
				break
			}
		}
		os.Remove(file)
		if err := printVerdict(matchingSong, db); err != nil {
			log.Fatal(err)
		}
	}
}
