package main

import (
	"fmt"
	"log"
	"math"

	"github.com/joho/godotenv"
)

// Todo make sure windowSize is not used anywhere!!!
const (
	PI           = math.Pi
	samplingRate = 11025
	windowSize   = 1024
	hopFactor    = 2
	hopSize      = windowSize / hopFactor //how much to slide each window by
)

// takes a seconds amount and computes how many windows will be needed when considering the hop size
// in order to reach that amount of seconds
func SecondsToWindows(seconds float64) int {
	if seconds == 0 {
		return 0
	}
	secondsPerHop := float64(hopSize) / samplingRate
	//fmt.Println("Seconds per Hop: ", secondsPerHop)
	return int(math.Ceil(seconds / secondsPerHop))
}

func main() {
	/*
		//signal generation
		N := 1024
		FrequencyRate := 1024
		samples := make([]float64, N)
		for i := 0; i < N; i++ {
			t := float64(i) / float64(FrequencyRate)
			//signal := math.Cos(2*PI*426*t) + math.Cos(2*PI*200.5*t) + math.Sin(2*PI*100.20*t) + math.Sin(2*PI*20000*t) + math.Sin(2*PI*400*t)
			//signal := math.Cos(2*PI*400*t) + math.Cos(2*PI*200*t) + math.Sin(2*PI*100*t) + math.Sin(2*PI*10*t) + math.Sin(2*PI*400*t)
			signal := math.Sin(2*PI*20*t) + math.Sin(2*PI*50*t) + math.Cos(2*PI*120*t)
			samples[i] = signal
		}

		frequencies := generateSpectogram(samples, windowSize, hopSize)[0]
			samples = lowpass(samples, FrequencyRate, 50)
			frequencies := generateSpectogram(samples)

			err := spectogramImage(frequencies)
			if err != nil {
				log.Fatal(err)
			}

		//frequenices
		for i := 0; i < len(frequencies); i++ {
			cur := frequencies[i]
			fmt.Printf("%.16f   |  i:%d\n", cur, i)
		}
	*/

	//file := "[Spectrogram] Aphex Twin ⧸ ΔMi−1 = −∂Σn=1NDi[n][Σj∈C{i}Fji[n − 1] + Fexti[[n−1]].wav"
	/*
			file := "filtered.wav"

			//file := "output.wav"
			wavData, err := ParseWav("./audioFiles/" + file)
			if err != nil {
				log.Fatal(err)
			}

			if wavData.header.NumChannels == 2 {
				wavData.samples, err = stereoToMono(wavData.samples)
				if err != nil {
					log.Fatal(err)
				}
			}

			//wavData.samples = lowpass(wavData.samples, int(wavData.header.SampleRate), 5000)
			//wavData.samples = downsample(wavData.samples, 4)

			spectogram := generateSpectogram(wavData.samples, windowSize, hopSize)
			err = spectogramImage(spectogram)
			if err != nil {
				log.Fatal(err)
			}

			fmt.Println("length audio: ", amoundData(spectogram))

			peaks := extractPeaks(spectogram, 21, 21, 0)
			err = displayPeaks(peaks)
			if err != nil {
				log.Fatal(err)
			}

			fmt.Println("max distance until next peak", maxTimeBeteenNeighbourPeaks(peaks))
			fmt.Println("averagae distance until next peak", averageTimeBeteenNeighbourPeaks(peaks))

		samples := [][]float64{
			{1., 5., 0., 9., 7, 1, 2, 3},
			{7., 3., 1., 4., 0., 4, 5, 0},
			{2., 3., 5., 4., 0., 2, 4, 3},
			{4., 9., 1., 8., 10., 1, 3, 9},
			{1, 2, 3, 4, 5, 7, 8, 5},
			{2., 7., 5., 4., 0., 0, 0, 1},
			{7., 3., 1., 6., 0., 1, 1, 1},
		}

		printArray(samples)
		peaks := extractPeaks(samples, 3, 3, 0)
		printArray(peaks)
		seconds := 0.
		fmt.Printf("%f seconds to window: %d\n", seconds, SecondsToWindows(seconds))

		hashes := generateHashes(peaks, 0, 10, 1, 1, 100)
		fmt.Printf("%+v\n", hashes)

	*/
	err := godotenv.Load()
	if err != nil {
		log.Fatal(err)
	}

	NewDBConnection()

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
