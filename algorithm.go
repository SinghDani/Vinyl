package main

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"os"
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

func getMean(frequencyMatrix [][]float64) float64 {
	count := 0.0
	sum := 0.0
	for i := 0; i < len(frequencyMatrix); i++ {
		for j := 0; j < len(frequencyMatrix[i]); j++ {
			if frequencyMatrix[i][j] != 0 {
				count++
				sum += frequencyMatrix[i][j]
			}
		}
	}
	return sum / count
}

// find the most prominent frequencies through a 2d filter which will find the
// frequencies with the highest magnitude in a neighbourhood grid specified by xDim and yDim and only keep those
func extractPeaks(frequencyMatrix [][]float64, xDim, yDim int, threshhold float64) [][]bool {
	if xDim == 0 || yDim == 0 {
		log.Fatal("xDim and yDim have to be odd to center grid around point")
	}

	xDim /= 2
	yDim /= 2
	numWindows := len(frequencyMatrix)
	numFrequencyBins := len(frequencyMatrix[0])
	peaks := make([][]bool, numWindows)
	count := 0

	for row := 0; row < numWindows; row++ {
		peaks[row] = make([]bool, numFrequencyBins)
		for col := 0; col < numFrequencyBins; col++ {
			isPeak := true
			curMagnitude := frequencyMatrix[row][col]
			//exclude all points that have to low of a magnitude or are 0
			if curMagnitude < threshhold || curMagnitude == 0 {
				continue
			}

			for k := max(0, row-yDim); k < min(numWindows, row+yDim+1); k++ {
				for l := max(0, col-xDim); l < min(numFrequencyBins, col+xDim+1); l++ {
					//next 2 if statements needed for finding strict peaks in a neighbourhood
					//and to avoid plateaus in which multiple neighbourhood points have the same magnitude
					if k == row && l == col {
						continue
					}
					if curMagnitude <= frequencyMatrix[k][l] {
						isPeak = false
						break
					}
				}
				if !isPeak {
					break
				}
			}
			if isPeak {
				peaks[row][col] = true
				count++
			}
		}
	}
	//printArray(filtered)
	//printArray(peaks)
	fmt.Println("peak amount", count)
	return peaks
}

func displayPeaks(peaks [][]bool) error {
	numWindows := len(peaks)
	if numWindows == 0 {
		return errors.New("no frames available")
	}
	numFrequencyBins := len(peaks[0])
	if numFrequencyBins == 0 {
		return errors.New("no frequencies available")
	}

	img := image.NewGray(image.Rect(0, 0, numWindows, numFrequencyBins))

	//lower frequencies are written to the bottom, higher ones to the top
	for x := 0; x < numWindows; x++ {
		for y := 0; y < numFrequencyBins; y++ {
			var val uint8 = 0
			if peaks[x][y] {
				val = 255
			}
			img.SetGray(x, numFrequencyBins-1-y, color.Gray{val})
		}
	}

	f, err := os.Create("constelationMap.png")
	if err != nil {
		return err
	}
	defer f.Close()

	return png.Encode(f, img)
}
