package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
)

func recordAudio(db *DBConnection) error {
	file := "recording.wav"
	recordingBatchTime := 5 //records in batches of 5 sec
	var masterHashList [][]GeneratedHash
	var matchingSong MatchingSong
	var match bool
	for i := range 8 { //record for max of 40 sec
		os.Remove(file)
		timeOffset := i * SecondsToWindows(float64(recordingBatchTime))

		cmd := exec.Command(
			"ffmpeg",
			"-y",
			"-f", "avfoundation",
			"-i", ":1",
			"-t", strconv.Itoa(recordingBatchTime),
			"-ac", "2",
			"-ar", "44100",
			"-c:a", "pcm_s16le",
			file,
		)
		out, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Println(string(out))
			return err
		}

		genHashes, err := ExtractHashesFromFile(file, timeOffset) //shift current anchor points by 0, 5, 10, ... sec
		if err != nil {
			return err
		}
		masterHashList = append(masterHashList, genHashes)

		batchWindow := 3 //only look at last 15 seconds
		var hashBatch []GeneratedHash

		start := max(0, len(masterHashList)-batchWindow)
		for _, batch := range masterHashList[start:] {
			hashBatch = append(hashBatch, batch...)
		}

		matchingSong, err = IdentifyRecording(db, hashBatch)
		if err != nil {
			return err
		}

		if err := printVerdict(matchingSong, db); err != nil {
			return err
		}

		match = EvalMatch(matchingSong)
		if match {
			break
		}
	}
	os.Remove(file)
	if err := printVerdict(matchingSong, db); err != nil {
		return err
	}

	return nil
}
