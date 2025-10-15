package main

import "fmt"

func downsample(samples []float64, factor int) []float64 {
	N := len(samples) / factor
	output := make([]float64, N)

	for i := 0; i < N; i++ {
		sum := 0.0
		for j := 0; j < factor; j++ {
			sum += samples[factor*i+j]
		}
		output[i] = sum / float64(factor)
	}
	return output
}

// low pass filter will decrease the magnitude of frequencies above the cutoff frequency
// to then be able to downsample
func lowpass(samples []float64, sampleRate int, cutoff int) []float64 {
	dt := 1.0 / float64(sampleRate)
	RC := 1.0 / (2 * PI * float64(cutoff))
	alpha := dt / (RC + dt)
	fmt.Printf("\n\nalpha: %v \n\n", alpha)
	output := make([]float64, len(samples))

	output[0] = alpha * samples[0]
	for i := 1; i < len(samples); i++ {
		output[i] = alpha*samples[i] + (1-alpha)*output[i-1]
	}
	return output
}
