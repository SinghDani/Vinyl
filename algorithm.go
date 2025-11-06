package main

import (
	"log"
)

// will decrease the sample rate by factor
// in this case factor 4 to go from 44.1khz -> 11.025 hz
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
	output := make([]float64, len(samples))

	output[0] = alpha * samples[0]
	for i := 1; i < len(samples); i++ {
		output[i] = alpha*samples[i] + (1-alpha)*output[i-1]
	}
	return output
}

func max2dFilter(samples [][]int, xDim, yDim int) [][]int {
	if xDim%2 == 0 || yDim%2 == 0 {
		log.Fatal("xDim and YDim have to be uneven")
	}

	xDim = xDim / 2 //center the grid in the x-dimension
	yDim = yDim / 2 //center the grid in the y-dimension
	width := len(samples)
	height := len(samples[0])
	output := make([][]int, height)
	for i := 0; i < height; i++ {
		output[i] = make([]int, width)
		for j := 0; j < width; j++ {
			cur := samples[i][j]
			for k := max(0, i-yDim); k < min(width, i+yDim+1); k++ {
				for l := max(0, j-xDim); l < min(height, j+xDim+1); l++ {
					cur = max(cur, samples[k][l])
				}
			}
			output[i][j] = cur
		}
	}

	for i := 0; i < height; i++ {
		for j := 0; j < width; j++ {
			if samples[i][j] != output[i][j] {
				samples[i][j] = 0
			}
		}
	}
	return output
}
