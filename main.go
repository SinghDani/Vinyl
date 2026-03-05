package main

import (
	"fmt"
	"log"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal(err)
	}

	db, err := NewDBConnection()
	if err != nil {
		log.Fatal(err)
	}
	defer db.CloseDBConnection()

	file := "filtered.wav"
	genHashes, err := ExtractHashesFromFile(file)
	if err != nil {
		log.Fatal(err)
	}

	store := false

	if store {
		if err := db.StoreHashes(genHashes, file); err != nil {
			log.Fatal(err)
		}
	} else {
		matchingSong, err := IdentifyRecording(db, genHashes)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("Matching song:", matchingSong)
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
