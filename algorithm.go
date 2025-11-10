package main

import (
	"errors"
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

func extractPeaks(frequencyMatrix [][]float64, xDim, yDim int) [][]bool {
	if xDim == 0 || yDim == 0 {
		log.Fatal("xDim and yDim have to be odd to center grid around point")
	}

	xDim /= 2
	yDim /= 2
	numWindows := len(frequencyMatrix)
	numFrequencyBins := len(frequencyMatrix[0])
	peaks := make([][]bool, numWindows)

	for i := 0; i < numWindows; i++ {
		peaks[i] = make([]bool, numFrequencyBins)
		for j := 0; j < numFrequencyBins; j++ {
			isPeak := true
			curMagnitude := frequencyMatrix[i][j]
			//if statement needed since the last windows is zero padded (see generateSpectogram function)
			// which will cause all the points in that region to be set to true
			if curMagnitude == 0 {
				isPeak = false
				break
			}
			for k := max(0, i-yDim); k < min(numWindows, i+yDim+1); k++ {
				for l := max(0, j-xDim); l < min(numFrequencyBins, j+xDim+1); l++ {
					if curMagnitude < frequencyMatrix[k][l] {
						isPeak = false
						break
					}
				}
				if !isPeak {
					break
				}
			}
			if isPeak {
				peaks[i][j] = true
			}
		}
	}
	//printArray(filtered)
	//printArray(peaks)
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
