package main

import (
	"math"
)

const windowSize = 1024
const hopSize = 512

func generateSpectogram(samples []float64) [][]complex128 {
	N := len(samples)
	if N == 0 {
		return [][]complex128{}
	}

	windowCount := int(math.Ceil(float64(N-windowSize)/hopSize)) + 1

	freqMatrix := make([][]complex128, windowCount)

	//process the signal using overlapping windows
	for i := 0; i < windowCount; i++ {
		freqMatrix[i] = make([]complex128, windowSize)
		start := i * hopSize
		end := start + windowSize

		if end > N {
			end = N
		}

		//if last bit to short, it gets zero padded
		curSamples := make([]float64, windowSize)
		copy(curSamples, samples[start:end])

		//apply hamming window
		for n := 0; n < windowSize; n++ {
			curSamples[n] *= 0.54 - 0.46*math.Cos(2*PI*float64(n)/float64(windowSize-1))
		}

		//inplace fft
		fft(curSamples, windowSize, freqMatrix[i], 0, 0, 1)

		//freqMatrix[i] = fft2(curSamples, windowSize)
		//freqMatrix[i] = dft(curSamples)
	}
	return freqMatrix
}
