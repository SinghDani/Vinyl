package main

import "math"

const windowSize = 1024
const hopSize = 1024

func generateSpectogram(samples []float64) []complex128 {
	N := len(samples)

	if N == 0 {
		return []complex128{}
	}

	//apply hamming window
	for n := 0; n < N; n++ {
		samples[n] *= 0.54 - 0.46*math.Cos(2*PI*float64(n)/float64(N-1))
	}

	//frequencies := fft2(samples, len(samples))
	//frequencies := dft(samples)
	frequencies := make([]complex128, len(samples))
	fft(samples, len(samples), frequencies, 0, 0, 1)

	return frequencies

}
