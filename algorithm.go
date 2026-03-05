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

type GeneratedHash struct {
	Hash       uint32 // anchor freq | target freq | dt
	AnchorTime uint32
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
		dt      int
	}

	//TODO maybe prealocate space
	timedMatches := make(map[deltaKey]int)

	var max int
	var matchingSongId uint32
	for _, hash := range recordingHashes {
		for _, fingerPrint := range fingerPrints {
			if hash.Hash == fingerPrint.Hash {
				key := deltaKey{
					song_id: fingerPrint.SongId,
					dt:      int(fingerPrint.AnchorTime) - int(hash.AnchorTime),
				}
				timedMatches[key]++
				cur := timedMatches[key]
				if cur > max {
					max = cur
					matchingSongId = key.song_id
				}
			}
		}
	}
	fmt.Println(timedMatches)
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
