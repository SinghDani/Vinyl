package main

import (
	"encoding/binary"
	"fmt"
)

type WavHeader struct {
	ChunkID       string
	Format        string
	Subchunk1ID   string
	Subchunk2ID   string
	ChunkSize     uint32
	Subchunk1Size uint32
	SampleRate    uint32
	ByteRate      uint32
	Subchunk2Size uint32
	AudioFormat   uint16
	NumChannels   uint16
	BlockAlign    uint16
	BitsPerSample uint16
}

type IncorrectWavFormat struct{}

func (I IncorrectWavFormat) Error() string {
	return "Incorrect Wav Format"
}

func ExtractWavHeader(headerData []byte) (*WavHeader, error) {
	if len(headerData) != 44 {
		return nil, IncorrectWavFormat{}
	}

	header := &WavHeader{}

	header.ChunkID = string(headerData[:4])
	if header.ChunkID != "RIFF" {
		return nil, IncorrectWavFormat{}
	}
	header.ChunkSize = binary.LittleEndian.Uint32(headerData[4:8])
	header.Format = string(headerData[8:12])
	if header.Format != "WAVE" {
		return nil, IncorrectWavFormat{}
	}
	header.Subchunk1ID = string(headerData[12:16])
	if header.Subchunk1ID != "fmt " {
		return nil, IncorrectWavFormat{}
	}
	header.Subchunk1Size = binary.LittleEndian.Uint32(headerData[16:20])
	header.AudioFormat = binary.LittleEndian.Uint16(headerData[20:22])
	header.NumChannels = binary.LittleEndian.Uint16(headerData[22:24])
	header.SampleRate = binary.LittleEndian.Uint32(headerData[24:28])
	header.ByteRate = binary.LittleEndian.Uint32(headerData[28:32])
	header.BlockAlign = binary.LittleEndian.Uint16(headerData[32:34])
	header.BitsPerSample = binary.LittleEndian.Uint16(headerData[34:36])
	header.Subchunk2ID = string(headerData[36:40])
	if header.Subchunk2ID != "data" {
		return nil, IncorrectWavFormat{}
	}
	header.Subchunk2Size = binary.LittleEndian.Uint32(headerData[40:44])

	fmt.Printf("%+v", header)
	return header, nil
}
