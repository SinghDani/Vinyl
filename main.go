package main

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

func main() {
	if len(os.Args) < 3 {
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

	//file := "audioFiles/[Spectrogram] Aphex Twin ⧸ ΔMi−1 = −∂Σn=1NDi[n][Σj∈C{i}Fji[n − 1] + Fexti[[n−1]].wav"
	//file := "audioFiles/Pink Floyd - Money (Official Music Video).wav"
	file := os.Args[1]

	if os.Args[2] == "2" {

	} else {
		genHashes, err := ExtractHashesFromFile(file)
		if err != nil {
			log.Fatal(err)
		}

		store := os.Args[2] == "1"

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
