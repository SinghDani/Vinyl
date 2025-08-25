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

func ParseWav(path string) error {
	/*
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
	*/
	samples := []byte{0x52, 0x49, 0x46, 0x46, 0x24, 0x08, 0x00, 0x00, 0x57, 0x41,
		0x56, 0x45, 0x66, 0x6d, 0x74, 0x20, 0x10, 0x00, 0x00, 0x00,
		0x01, 0x00, 0x02, 0x00, 0x22, 0x56, 0x00, 0x00, 0x88, 0x58,
		0x01, 0x00, 0x04, 0x00, 0x10, 0x00, 0x64, 0x61, 0x74, 0x61,
		0x00, 0x08, 0x00, 0x00}

	header, err := ExtractWavHeader(samples)
	if err != nil {
		return err
	}
	fmt.Printf("%+v", header)

	return nil
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

	return header, nil
}
