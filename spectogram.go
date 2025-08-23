package main

import (
	"image"
	"image/color"
	"image/png"
	"log"
	"math"
	"math/cmplx"
	"os"
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

	//remove second half of frequncies since they are exact mirrors of first half
	for i := 0; i < windowCount; i++ {
		freqMatrix[i] = freqMatrix[i][:windowSize/2]
	}
	return freqMatrix
}

func spectogramImage(freqMatrix [][]complex128) {
	width := len(freqMatrix)
	if width == 0 {
		return
	}
	height := len(freqMatrix[0])
	if height == 0 {
		return
	}

	img := image.NewGray(image.Rect(0, 0, width, height))

	//find the max magnitude for normalisation
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
			value := (mag / maxMagnitude) * 255
			img.SetGray(x, height-1-y, color.Gray{uint8(value)})
		}
	}

	f, err := os.Create("spectogram-image.png")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	png.Encode(f, img)
}
