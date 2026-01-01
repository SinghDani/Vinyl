package main

import (
	"fmt"
	"log"
)

const (
	windowSize = 1024
	hopSize    = windowSize / 2 //how much to slide each window by
)

func main() {
	/*
		//signal generation
		N := 20000
		FrequencyRate := 44100
		samples := make([]float64, N)
		for i := 0; i < N; i++ {
			t := float64(i) / FrequencyRate
			//signal := math.Cos(2*PI*426*t) + math.Cos(2*PI*200.5*t) + math.Sin(2*PI*100.20*t) + math.Sin(2*PI*20000*t) + math.Sin(2*PI*400*t)
			//signal := math.Cos(2*PI*400*t) + math.Cos(2*PI*200*t) + math.Sin(2*PI*100*t) + math.Sin(2*PI*10*t) + math.Sin(2*PI*400*t)
			signal := math.Sin(2*PI*20*t) + math.Sin(2*PI*256*t) + math.Sin(2*PI*50*t)
			samples[i] = signal
		}

		samples = lowpass(samples, FrequencyRate, 50)
		//frequencies := generateSpectogram(samples)[0]
		frequencies := generateSpectogram(samples)

		err := spectogramImage(frequencies)
		if err != nil {
			log.Fatal(err)
		}
	*/

	/*
		//frequenices
		for i := 0; i < len(frequencies); i++ {
			cur := frequencies[i]
			fmt.Printf("%.16f   |  i:%d 	|| 	sin: %.16f 	|| 	cos: %.16f \n", cmplx.Abs(cur), i, imag(cur), real(cur))
		}

	*/

	//file := "[Spectrogram] Aphex Twin ⧸ ΔMi−1 = −∂Σn=1NDi[n][Σj∈C{i}Fji[n − 1] + Fexti[[n−1]].wav"
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
		{1., 3., 0., 4., 9},
		{7., 3., 1., 4., 3},
		{2., 3., 5., 4., 0},
		{4., 2., 1., 8., 4},
	}

	printArray(samples)
	printArray(extractPeaks(samples, 5, 5, 0))
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
