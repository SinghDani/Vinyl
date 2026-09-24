//go:build js && wasm

package main

import (
	"encoding/json"
	"fmt"
	"syscall/js"

	"github.com/SinghDani/vinyl/fingerprint"
	"github.com/SinghDani/vinyl/internal"
	"github.com/SinghDani/vinyl/internal/audioconfig"
	"github.com/SinghDani/vinyl/wav"
)

func main() {
	js.Global().Set("samplesToHashes", js.FuncOf(samplesToHashes))
	js.Global().Set("getFingerprintConfig", js.FuncOf(func(this js.Value, args []js.Value) any {
		return map[string]any{"sampleRate": audioconfig.SampleRate}
	}))
	js.Global().Set("secondsToWindows", js.FuncOf(func(this js.Value, args []js.Value) any {
		return fingerprint.SecondsToWindows(args[0].Float())
	}))
	select {}
}

func samplesToHashes(this js.Value, args []js.Value) any {
	inputBuffer := make([]byte, args[0].Get("byteLength").Int())
	js.CopyBytesToGo(inputBuffer, args[0])

	samples, err := wav.BytesToSamples(inputBuffer)
	if err != nil {
		fmt.Println("invalid audio payload:", err)
		return formatResult(nil, err)
	}

	genHashes, err := fingerprint.SamplesToHashes(samples, args[1].Int())
	if err != nil {
		fmt.Println("hash extraction failed:", err)
		return formatResult(nil, err)
	}
	return formatResult(genHashes, nil)
}

func formatResult(result []internal.GeneratedHash, err error) string {
	var errMsg any
	if err != nil {
		errMsg = err.Error()
	}

	temp := map[string]any{
		"hashes": result,
		"error":  errMsg,
	}
	res, err := json.Marshal(temp)
	if err != nil {
		return `{"hashes": "nil", "error":"failed to marshal result"}`
	}
	return string(res)
}
