package main

import (
	"fmt"
	"os/exec"
	"strconv"
)

func recordAudio(outPath string, seconds int) error {
	cmd := exec.Command(
		"ffmpeg",
		"-y",
		"-f", "avfoundation",
		"-i", ":1",
		"-t", strconv.Itoa(seconds),
		"-ac", "2",
		"-ar", "44100",
		"-c:a", "pcm_s16le",
		outPath,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(string(out))
		return err
	}
	return nil
}
