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
		if err := recordAudio(file, 15); err != nil {
			log.Fatal(err)
		}
		defer os.Remove(file)

		//file := "audioFiles/Nirvana - Smells Like Teen Spirit (Official Music Video).wav"
		//file := "audioFiles/Pink Floyd - Money (Official Music Video).wav"

		genHashes, err := ExtractHashesFromFile(file)
		if err != nil {
			log.Fatal(err)
		}

		matchingSong, err := IdentifyRecording(db, genHashes)
		if err != nil {
			log.Fatal(err)
		}
		songName, err := db.GetSong(matchingSong)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Matching song: id: %d, name:%s", matchingSong, songName)
		//TestPerfectMatch("audioFiles/The White Stripes - Seven Nation Army (Official Music Video).wav", db)
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
