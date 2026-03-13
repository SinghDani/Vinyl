package main

import (
	"fmt"
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
		genHashes, err := ExtractHashesFromFile(file)
		if err != nil {
			log.Fatal(err)
		}

		if err := db.StoreHashes(genHashes, filepath.Base(file)); err != nil {
			log.Fatal(err)
		}
	} else {
		file := "recording.wav"
		if err := recordAudio(file, 5); err != nil {
			log.Fatal(err)
		}
		defer os.Remove(file)

		genHashes, err := ExtractHashesFromFile(file)
		if err != nil {
			log.Fatal(err)
		}

		matchingSong, err := IdentifyRecording(db, genHashes)
		if err != nil {
			log.Fatal(err)
		}
		match, err := EvalMatch(matchingSong, db)
		if err != nil {
			log.Fatal(err)
		}
		_ = match
	}
}

func printArray[T any](samples [][]T) {
	for i := 0; i < len(samples); i++ {
		fmt.Println(samples[i])
	}
	fmt.Println()
}

func amoundData(samples [][]float64) int {
	result := 0
	for _, window := range samples {
		result += len(window)
	}
	return result
}
