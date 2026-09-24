package fingerprint

import (
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"math"
	"os"

	"github.com/SinghDani/vinyl/db"
	"github.com/SinghDani/vinyl/internal"
	"github.com/SinghDani/vinyl/internal/audioconfig"
	"github.com/SinghDani/vinyl/wav"
)

const (
	PI           = math.Pi
	SamplingRate = audioconfig.SampleRate
)

func ExtractHashesFromFile(file string, timeOffset int) ([]internal.GeneratedHash, error) {
	wavData, err := wav.WavToSamples(file)
	if err != nil {
		return nil, err
	}

	return SamplesToHashes(wavData.Samples, timeOffset)
}

func SamplesToHashes(samples []float64, timeOffset int) ([]internal.GeneratedHash, error) {
	spectogram := generateSpectogram(samples, audioconfig.WindowSize, audioconfig.HopSize)
	/*
		err := spectogramImage(spectogram)
		if err != nil {
			return nil, err
		}
	*/

	peaks := extractPeaks(spectogram, audioconfig.PeakFrequencyBins, audioconfig.PeakTimeWindows, getMeanArr(spectogram))

	/*
		err = displayPeaks(peaks)
		if err != nil {
			return nil, err
		}
	*/

	hashes := generateHashes(peaks,
		audioconfig.TargetStartSeconds, audioconfig.TargetEndSeconds,
		audioconfig.TargetFrequencyBinsAbove, audioconfig.TargetFrequencyBinsBelow,
		audioconfig.PairsPerAnchor, timeOffset)
	fmt.Println("hashes amount:", len(hashes))
	return hashes, nil
}

// will compare recordings hashes to the ones in the db to find the matching song
func IdentifyRecording(ctx context.Context, db *db.DBConnection, genHashes []internal.GeneratedHash) (internal.MatchingSong, error) {
	fingerPrints, err := db.ExtractHashes(ctx, genHashes)
	if err != nil {
		return internal.MatchingSong{}, err
	}
	//fmt.Printf("Fingerprints: %+v\n", fingerPrints)
	//fmt.Println("\nsong comparison:")
	return findMatchingSong(ctx, fingerPrints, genHashes, db)
}

// takes a seconds amount and computes how many windows will be needed when considering the hop size
// in order to reach that amount of seconds
func SecondsToWindows(seconds float64) int {
	if seconds == 0 {
		return 0
	}
	secondsPerHop := float64(audioconfig.HopSize) / SamplingRate
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
	if count == 0 {
		return 0
	}
	return sum / count
}

// get the mean of each row
func getMeanArr(frequencyMatrix [][]float64) []float64 {
	numFrequencyBins := len(frequencyMatrix[0])
	means := make([]float64, len(frequencyMatrix))
	for i := 0; i < len(frequencyMatrix); i++ {
		count := 0.0
		sum := 0.0
		for j := 0; j < numFrequencyBins; j++ {
			if frequencyMatrix[i][j] != 0 {
				count++
				sum += frequencyMatrix[i][j]
			}
		}
		if count == 0 {
			continue
		}
		means[i] = sum / count
	}
	return means
}

func extractPeaksBands(frequencyMatrix [][]float64) [][]bool {
	type band struct {
		min int
		max int
	}
	type peak struct {
		magnitude float64
		index     int
	}
	bands := []band{
		{0, 10}, {10, 20}, {20, 40}, {40, 80}, {80, 160}, {160, 512},
	}
	peaks := make([][]bool, len(frequencyMatrix))
	numFrequencyBins := len(frequencyMatrix[0])

	var count int
	for windowIndex, window := range frequencyMatrix {
		peaks[windowIndex] = make([]bool, numFrequencyBins)
		curPeaks := make([]peak, 6)
		for bandIndex, band := range bands {
			var maxMag float64
			var peakIndex int
			for bin := band.min; bin < band.max; bin++ {
				curFreq := window[bin]
				if curFreq > maxMag {
					maxMag = curFreq
					peakIndex = bin
				}
			}
			curPeaks[bandIndex] = peak{maxMag, peakIndex}
		}
		var sum float64
		for _, peak := range curPeaks {
			sum += peak.magnitude
		}
		avg := sum / 6
		for _, peak := range curPeaks {
			if peak.magnitude >= avg && peak.magnitude != 0 {
				count++
				peaks[windowIndex][peak.index] = true
			}
		}
	}

	fmt.Println("peak amount", count)
	return peaks
}

// find the most prominent frequencies through a 2d filter which will find the
// frequencies with the highest magnitude in a neighbourhood grid specified by xDim and yDim and only keep those
// TODO maybe store index where peak is instead of true false grid
func extractPeaks(frequencyMatrix [][]float64, xDim, yDim int, threshhold []float64) [][]bool {
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
		for bin := 25; bin < numFrequencyBins; bin++ {
			isPeak := true
			curMagnitude := frequencyMatrix[window][bin]
			//exclude all points that have to low of a magnitude or are 0
			if curMagnitude < threshhold[window] || curMagnitude == 0 {
				continue
			}

			for i := max(0, window-yDim); i < min(numWindows, window+yDim+1); i++ {
				for j := max(25, bin-xDim); j < min(numFrequencyBins, bin+xDim+1); j++ {
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

// will combine anchor points (peaks) with other peaks from a target zones to create hashes
// by combining them as specified in the comments in the following function
// TODO: optimise maybe by passing in also the amount peaks
func generateHashes(peaks [][]bool, secondsOffset float64, secondsThreshold float64, frequencyBinUpperBound, frequencyBinLowerBound, numPairsPerAnchor, timeOffset int) []internal.GeneratedHash {
	numWindows := len(peaks)
	numBins := len(peaks[0])
	hashes := []internal.GeneratedHash{}

	for window := 0; window < numWindows; window++ {
		for bin := 25; bin < numBins; bin++ {
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
				for curBin := max(25, bin-frequencyBinLowerBound); curBin < min(numBins, bin+frequencyBinUpperBound+1); curBin++ {
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
					anchorTime := uint32(window + timeOffset)
					//fmt.Printf("Hash(window, bin) between: (%d, %d) | (%d, %d)	:=	%.32b : data=%d\n", window, bin, curWindow, curBin, hash, data)

					hashes = append(hashes, internal.GeneratedHash{Hash: hash, AnchorTime: anchorTime})
					generatedHashes++
				}
			}
		}
	}
	return hashes
}

// identifies the best matching song by building a time-delta histogram between recording hashes
// and db fingerprints for every song that was matched
// returning the top matching song with a confidence ratio
func findMatchingSong(ctx context.Context, fingerPrints []internal.Fingerprint, recordingHashes []internal.GeneratedHash, db *db.DBConnection) (internal.MatchingSong, error) {
	type deltaKey struct {
		song_id uint32
		dt      int64
	}

	timedMatches := make(map[deltaKey]int)
	fingerPrintsMap := make(map[uint32][]internal.Fingerprint)

	dtBinSize := int64(max(SecondsToWindows(0.3), 1))
	//fmt.Println("DtBinSize: ", dtBinSize)

	for _, fingerPrint := range fingerPrints {
		fingerPrintsMap[fingerPrint.Hash] = append(fingerPrintsMap[fingerPrint.Hash], fingerPrint)
	}

	//create time-delta histogram
	for _, hash := range recordingHashes {
		matches, ok := fingerPrintsMap[hash.Hash]
		if !ok {
			continue
		}
		for _, fingerPrint := range matches {
			rawDt := int64(fingerPrint.AnchorTime) - int64(hash.AnchorTime)
			binnedDt := int64(math.Round(float64(rawDt) / float64(dtBinSize)))
			key := deltaKey{
				song_id: fingerPrint.SongId,
				dt:      binnedDt,
			}
			timedMatches[key]++
		}
	}

	//exportAllHistogramsToCSV(timedMatches, "all_histograms.csv")

	songBestScores := make(map[uint32]int)

	//use binning to combine neighbouring histogram bars to improve robustness
	for key, count := range timedMatches {

		/*
			if count > songBestScores[key.song_id] {
				songBestScores[key.song_id] = count
			}
		*/

		clusterScore := count*2 +
			timedMatches[deltaKey{key.song_id, key.dt - 1}] +
			timedMatches[deltaKey{key.song_id, key.dt + 1}]

		if clusterScore > songBestScores[key.song_id] {
			songBestScores[key.song_id] = clusterScore
		}

	}

	//compute confidence ratio for the classification
	var topScore int
	var secondScore int
	var topSong uint32
	var secondSong uint32

	for songId, score := range songBestScores {
		if score > topScore {
			secondScore = topScore
			secondSong = topSong
			topScore = score
			topSong = songId
		} else if score > secondScore {
			secondScore = score
			secondSong = songId
		}
	}

	_ = secondSong
	/*
		if topScore == 0 {
			fmt.Println("Result: No matches found in database.")
			return 0
		}

		fmt.Printf("1st Place: Song %d (Score: %d)\n", topSong, topScore)
		if secondScore > 0 {
			fmt.Printf("2nd Place: Song %d (Score: %d)\n", secondSong, secondScore)
			confidence := float64(topScore) / float64(secondScore)
			fmt.Printf("Confidence Ratio: %.2f\n", confidence)

			if confidence >= 1.5 && topScore >= 15 {
				fmt.Println("Verdict: STRONG MATCH")
			} else if confidence > 1.2 && topScore >= 10 {
				fmt.Println("Verdict: WEAK MATCH")
			} else {
				fmt.Println("Verdict: UNRELIABLE (Likely False Positive)")
			}
		} else {
			fmt.Println("2nd Place: None")
			fmt.Println("Confidence Ratio: INFINITE (Only one song got hits)")
			if topScore >= 10 {
				fmt.Println("Verdict: STRONG MATCH")
			} else {
				fmt.Println("Verdict: WEAK MATCH (Not enough data points)")
			}
		}
	*/
	if topScore == 0 {
		return internal.MatchingSong{}, nil
	}
	var confidence float64
	if secondScore == 0 {
		confidence = 999
	} else {
		confidence = float64(topScore) / float64(secondScore)
	}
	songEntry, err := db.GetSong(ctx, topSong)
	if err != nil {
		return internal.MatchingSong{}, err
	}
	return internal.MatchingSong{SongId: topSong, Name: songEntry.Name, Artist: songEntry.Artist, Confidence: confidence, Score: topScore}, nil
}

// returns if we found a significant match or not
func EvalMatch(song internal.MatchingSong) bool {
	if song.Score == 0 {
		return false
	}
	if song.Confidence >= 1.5 && song.Score >= 30 {
		return true
	} else if song.Confidence > 1.2 && song.Score >= 20 {
		return false
	} else {
		return false
	}
}

func PrintVerdict(song internal.MatchingSong, db *db.DBConnection) {
	if song.Score == 0 {
		fmt.Println("Result: No matches found in database.")
		return
	}
	fmt.Printf("Matching Song: Id: %d, Song: %s \nScore: %d\n", song.SongId, song.Name, song.Score)
	fmt.Printf("Confidence Ratio: %.2f\n", song.Confidence)

	if song.Confidence >= 1.5 && song.Score >= 30 {
		fmt.Println("Verdict: STRONG MATCH")
	} else if song.Confidence > 1.2 && song.Score >= 20 {
		fmt.Println("Verdict: WEAK MATCH")
	} else {
		fmt.Println("Verdict: UNRELIABLE (Likely False Positive)")
	}
}

func GetVerdict(song internal.MatchingSong) string {
	if song.Score == 0 {
		return "No matches found in database"
	}
	if song.Confidence >= 1.5 && song.Score >= 30 {
		return "STRONG MATCH"
	} else if song.Confidence > 1.2 && song.Score >= 20 {
		return "WEAK MATCH"
	} else {
		return "UNRELIABLE (Likely False Positive)"
	}
}
