package main

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"math"
	"os"
)

// Todo make sure windowSize is not used anywhere!!!
const (
	PI           = math.Pi
	samplingRate = 11025
	windowSize   = 1024
	hopFactor    = 2
	hopSize      = windowSize / hopFactor //how much to slide each window by
)

type GeneratedHash struct {
	Hash       uint32 // anchor freq | target freq | dt
	AnchorTime uint32
}

func ExtractHashesFromFile(file string) ([]GeneratedHash, error) {
	wavData, err := WavToSamples(file)
	if err != nil {
		return nil, err
	}

	return SamplesToHashes(wavData.samples)
}

func SamplesToHashes(samples []float64) ([]GeneratedHash, error) {
	spectogram := generateSpectogram(samples, windowSize, hopSize)
	/*
		err := spectogramImage(spectogram)
		if err != nil {
			return nil, err
		}
	*/

	//TODO don't hard code the window and threshold
	peaks := extractPeaks(spectogram, 21, 21, getMean(spectogram))
	/*
		err = displayPeaks(peaks)
		if err != nil {
			return nil, err
		}
	*/

	//fmt.Println("max distance until next peak", maxTimeBeteenNeighbourPeaks(peaks))
	//fmt.Println("averagae distance until next peak", averageTimeBeteenNeighbourPeaks(peaks))

	/*
		samples2 := [][]float64{
			{1., 5., 0., 9., 7, 1, 2, 3},
			{7., 3., 1., 4., 0., 4, 5, 0},
			{2., 3., 5., 4., 0., 2, 4, 3},
			{4., 9., 1., 8., 10., 1, 3, 9},
			{1, 2, 3, 4, 5, 7, 8, 5},
			{2., 7., 5., 4., 0., 0, 0, 1},
			{7., 3., 1., 6., 0., 1, 1, 1},
		}

		printArray(samples2)
		peaks = extractPeaks(samples2, 3, 3, 0)
		printArray(peaks)
		seconds := 0.
		fmt.Printf("%f seconds to window: %d\n", seconds, SecondsToWindows(seconds))
	*/

	//TODO change the thresholds and targetzone bounds
	hashes := generateHashes(peaks, 0, 10, 1, 1, 100)
	//fmt.Printf("Generated Hashes: %+v\n", hashes)

	return hashes, nil
}

func IdentifyRecording(db *DBConnection, genHashes []GeneratedHash) (uint32, error) {
	hashValues := make([]uint32, len(genHashes))
	for i, hash := range genHashes {
		hashValues[i] = hash.Hash
	}
	fingerPrints, err := db.ExtractHashes(hashValues)
	if err != nil {
		return 0, err
	}
	//fmt.Printf("Fingerprints: %+v\n", fingerPrints)
	//fmt.Println("\nsong comparison:")
	matchingSong := findMatchingSong(fingerPrints, genHashes)
	return matchingSong, nil
}

// takes a seconds amount and computes how many windows will be needed when considering the hop size
// in order to reach that amount of seconds
func SecondsToWindows(seconds float64) int {
	if seconds == 0 {
		return 0
	}
	secondsPerHop := float64(hopSize) / samplingRate
	//fmt.Println("Seconds per Hop: ", secondsPerHop)
	return int(math.Ceil(seconds / secondsPerHop))
}

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
	numFrequencyBins := len(frequencyMatrix[0])
	for i := 0; i < len(frequencyMatrix); i++ {
		for j := 0; j < numFrequencyBins; j++ {
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
// TODO maybe store index where peak is instead of true false grid
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

	for window := 0; window < numWindows; window++ {
		peaks[window] = make([]bool, numFrequencyBins)
		for bin := 0; bin < numFrequencyBins; bin++ {
			isPeak := true
			curMagnitude := frequencyMatrix[window][bin]
			//exclude all points that have to low of a magnitude or are 0
			if curMagnitude < threshhold || curMagnitude == 0 {
				continue
			}

			for i := max(0, window-yDim); i < min(numWindows, window+yDim+1); i++ {
				for j := max(0, bin-xDim); j < min(numFrequencyBins, bin+xDim+1); j++ {
					//next 2 if statements needed for finding strict peaks in a neighbourhood
					//and to avoid plateaus in which multiple neighbourhood points have the same magnitude
					if i == window && j == bin {
						continue
					}
					if curMagnitude <= frequencyMatrix[i][j] {
						isPeak = false
						break
					}
				}
				if !isPeak {
					break
				}
			}
			if isPeak {
				peaks[window][bin] = true
				count++
			}
		}
	}
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

// TODO: optimise maybe by passing in also the amount peaks
func generateHashes(peaks [][]bool, secondsOffset float64, secondsThreshold float64, frequencyBinUpperBound, frequencyBinLowerBound, numPairsPerAnchor int) []GeneratedHash {
	numWindows := len(peaks)
	numBins := len(peaks[0])
	hashes := []GeneratedHash{}

	for window := 0; window < numWindows; window++ {
		for bin := 0; bin < numBins; bin++ {
			if !peaks[window][bin] { // if anchor is not peak
				continue
			}

			targetZoneStart := window + max(SecondsToWindows(secondsOffset), 1)
			targetZoneEnd := window + max(SecondsToWindows(secondsThreshold), 1)

			generatedHashes := 0
			for curWindow := targetZoneStart; curWindow < min(numWindows, targetZoneEnd+1); curWindow++ {
				if generatedHashes >= numPairsPerAnchor {
					break
				}
				for curBin := max(0, bin-frequencyBinLowerBound); curBin < min(numBins, bin+frequencyBinUpperBound+1); curBin++ {
					if generatedHashes >= numPairsPerAnchor {
						break
					}
					if !peaks[curWindow][curBin] { // if point is not a peak
						continue
					}

					/* hash structure
							9 bit 					9 bit 				14 bit
					hash: anchor frequency	|	point frequency	|	delta time
					*/
					hash := (uint32(bin) << 23) | (uint32(curBin) << 14) | (uint32(curWindow - window))
					anchorTime := uint32(window)
					//fmt.Printf("Hash(window, bin) between: (%d, %d) | (%d, %d)	:=	%.32b : data=%d\n", window, bin, curWindow, curBin, hash, data)

					hashes = append(hashes, GeneratedHash{hash, anchorTime})
					generatedHashes++
				}
			}
		}
	}
	return hashes
}

func findMatchingSong(fingerPrints []Fingerprint, recordingHashes []GeneratedHash) uint32 {
	type deltaKey struct {
		song_id uint32
		dt      int64
	}

	timedMatches := make(map[deltaKey]int, len(recordingHashes))
	fingerPrintsMap := make(map[uint32][]Fingerprint, len(fingerPrints))

	for _, fingerPrint := range fingerPrints {
		fingerPrintsMap[fingerPrint.Hash] = append(fingerPrintsMap[fingerPrint.Hash], fingerPrint)
	}

	var max int
	var matchingSongId uint32
	for _, hash := range recordingHashes {
		matches, ok := fingerPrintsMap[hash.Hash]
		if !ok {
			continue
		}
		for _, fingerPrint := range matches {
			//TODO add binning
			key := deltaKey{
				song_id: fingerPrint.SongId,
				dt:      int64(fingerPrint.AnchorTime) - int64(hash.AnchorTime),
			}
			timedMatches[key]++
			cur := timedMatches[key]
			if cur > max {
				max = cur
				matchingSongId = key.song_id
			}
		}
	}
	//fmt.Println(timedMatches)
	return matchingSongId
}

func maxTimeBeteenNeighbourPeaks(peaks [][]bool) int {
	max := 0
	for window := 0; window < len(peaks); window++ {
		found := false
		for bin := 0; bin < len(peaks[window]); bin++ {
			if peaks[window][bin] {
				for i := window + 1; i < len(peaks); i++ {
					for j := 0; j < len(peaks[i]); j++ {
						if peaks[i][j] {
							found = true
							if max < i-window {
								max = i - window
							}
							break
						}
					}
					if found {
						break
					}
				}
				break
			}
		}
	}
	return max
}

func averageTimeBeteenNeighbourPeaks(peaks [][]bool) float64 {
	sum := 0.0
	count := 0.0
	for window := 0; window < len(peaks); window++ {
		found := false
		for bin := 0; bin < len(peaks[window]); bin++ {
			if peaks[window][bin] {
				for i := window + 1; i < len(peaks); i++ {
					for j := 0; j < len(peaks[i]); j++ {
						if peaks[i][j] {
							found = true
							sum += float64((i - window))
							count++
							break
						}
					}
					if found {
						break
					}
				}
				break
			}
		}
	}
	if count == 0 {
		return 0
	}
	return sum / count
}
