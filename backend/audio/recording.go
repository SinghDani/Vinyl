package audio

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"

	"github.com/SinghDani/vinyl/db"
	"github.com/SinghDani/vinyl/fingerprint"
	"github.com/SinghDani/vinyl/internal"
)

func RecordAudio(ctx context.Context, db *db.DBConnection) error {
	file := "recording.wav"
	recordingBatchTime := 5 //records in batches of 5 sec
	var masterHashList [][]internal.GeneratedHash
	var matchingSong internal.MatchingSong

	for i := range 8 { //record for max of 40 sec
		os.Remove(file)
		timeOffset := i * fingerprint.SecondsToWindows(float64(recordingBatchTime))

		cmd := exec.Command(
			"ffmpeg",
			"-y",
			"-f", "avfoundation",
			"-i", ":default",
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

		genHashes, err := fingerprint.ExtractHashesFromFile(file, timeOffset) //shift current anchor points by 0, 5, 10, ... sec
		if err != nil {
			return err
		}
		masterHashList = append(masterHashList, genHashes)

		batchWindow := 3 //only look at last 15 seconds
		var hashBatch []internal.GeneratedHash

		start := max(0, len(masterHashList)-batchWindow)
		for _, batch := range masterHashList[start:] {
			hashBatch = append(hashBatch, batch...)
		}

		matchingSong, err = fingerprint.IdentifyRecording(ctx, db, hashBatch)
		if err != nil {
			return err
		}

		fingerprint.PrintVerdict(matchingSong, db)
		if fingerprint.EvalMatch(matchingSong) {
			break
		}
	}
	os.Remove(file)
	fingerprint.PrintVerdict(matchingSong, db)
	return nil
}
