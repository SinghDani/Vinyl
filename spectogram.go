package main

import (
	"errors"
	"image"
	"image/color"
	"image/png"
	"math"
	"math/cmplx"
	"os"
)

const windowSize = 1024
const hopSize = windowSize / 2 //how much to slide each window by

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

		curSamples := make([]float64, windowSize)
		//if last window goes bejond the signals length N, it gets padded with zeros
		copy(curSamples, samples[start:end])

		//apply hamming window
		for n := 0; n < windowSize; n++ {
			curSamples[n] *= 0.54 - 0.46*math.Cos(2*PI*float64(n)/float64(windowSize-1))
		}

		//inplace fft
		fft(curSamples, windowSize, freqMatrix[i], 0, 0, 1)
	}

	//remove second half of frequncies since they are exact mirrors of the first half
	for i := 0; i < windowCount; i++ {
		freqMatrix[i] = freqMatrix[i][:windowSize/2]
	}
	return freqMatrix
}

func spectogramImage(freqMatrix [][]complex128) error {
	//width and height naming is swapped in comparison to what is usual matrix naming convention
	//since the inner slice is going to hold the frequency range which is the y-axis in a spectogram
	//while the outer slice will determine time on the the x-axis
	width := len(freqMatrix)
	if width == 0 {
		return errors.New("no frames available")
	}
	height := len(freqMatrix[0])
	if height == 0 {
		return errors.New("no frequencies available")
	}

	img := image.NewGray(image.Rect(0, 0, width, height))

	//find the max magnitude which will be used for normalisation
	maxMagnitude := 0.0
	for i := 0; i < width; i++ {
		for j := 0; j < height; j++ {
			mag := cmplx.Abs(freqMatrix[i][j])
			if mag > maxMagnitude {
				maxMagnitude = mag
			}
		}
	}

	//lower frequencies are written to the bottom, higher ones to the top
	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			mag := cmplx.Abs(freqMatrix[x][y])
			//value := 255 * mag / maxMagnitude
			//log based scaling so that higher frequencies are visible in the image
			value := math.Log10(1+mag) / math.Log10(1+maxMagnitude) * 255
			img.SetGray(x, height-1-y, color.Gray{uint8(value)})
		}
	}

	f, err := os.Create("spectogram-image.png")
	if err != nil {
		return err
	}
	defer f.Close()

	return png.Encode(f, img)
}
