package fingerprint

import (
	"errors"
	"image"
	"image/color"
	"image/png"
	"math"
	"math/cmplx"
	"os"
)

func generateSpectogram(samples []float64, windowSize, hopSize int) [][]float64 {
	N := len(samples)
	if N == 0 {
		return [][]float64{}
	}

	windowCount := int(math.Ceil(float64(N-windowSize)/float64(hopSize))) + 1

	freqMatrix := make([][]float64, windowCount)

	//process the signal using overlapping windows
	for i := 0; i < windowCount; i++ {
		start := i * hopSize
		end := start + windowSize

		if end > N {
			end = N
		}

		curSamples := make([]float64, windowSize)
		//if last window goes beyond the signals length N, it gets padded with zeros
		copy(curSamples, samples[start:end])

		//apply hamming window
		for n := 0; n < windowSize; n++ {
			curSamples[n] *= 0.54 - 0.46*math.Cos(2*PI*float64(n)/float64(windowSize-1))
		}

		//inplace fft
		fftResult := make([]complex128, windowSize)
		fft(curSamples, windowSize, fftResult, 0, 0, 1)

		//remove second half of frequencies since they are exact mirrors of the first half
		freqMatrix[i] = make([]float64, windowSize/2)
		for j := 0; j < windowSize/2; j++ {
			//convert the complex values to the float64 magnitudes of the frequencies
			freqMatrix[i][j] = math.Log1p(cmplx.Abs(fftResult[j]))
			//freqMatrix[i][j] = cmplx.Abs(fftResult[j])
		}
	}
	return freqMatrix
}

func spectogramImage(freqMatrix [][]float64) error {
	numWindows := len(freqMatrix)
	if numWindows == 0 {
		return errors.New("no frames available")
	}
	numFrequencyBins := len(freqMatrix[0])
	if numFrequencyBins == 0 {
		return errors.New("no frequencies available")
	}

	img := image.NewGray(image.Rect(0, 0, numWindows, numFrequencyBins))

	// find the max magnitude which will be used for normalisation
	maxMagnitude := 0.0
	for i := 0; i < numWindows; i++ {
		for j := 0; j < numFrequencyBins; j++ {
			mag := freqMatrix[i][j]
			if mag > maxMagnitude {
				maxMagnitude = mag
			}
		}
	}

	// lower frequencies are written to the bottom, higher ones to the top
	for x := 0; x < numWindows; x++ {
		for y := 0; y < numFrequencyBins; y++ {
			mag := freqMatrix[x][y]
			//value := 255 * mag / maxMagnitude
			//log based scaling so that higher frequencies are visible in the image
			value := math.Log10(1+mag) / math.Log10(1+maxMagnitude) * 255
			img.SetGray(x, numFrequencyBins-1-y, color.Gray{uint8(value)})
		}
	}

	f, err := os.Create("spectogram-image.png")
	if err != nil {
		return err
	}
	defer f.Close()

	return png.Encode(f, img)
}
