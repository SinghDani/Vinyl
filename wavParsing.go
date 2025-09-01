package main

import (
	"encoding/binary"
	"fmt"
	"os"
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
	DataOffset    uint32
}

type IncorrectWavFormat struct{}

func (I IncorrectWavFormat) Error() string {
	return "Incorrect Wav Format"
}

func ParseWav(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	/*
		data = []byte{0x52, 0x49, 0x46, 0x46, 0x24, 0x08, 0x00, 0x00, 0x57, 0x41,
			0x56, 0x45, 0x66, 0x6d, 0x74, 0x20, 0x10, 0x00, 0x00, 0x00,
			0x01, 0x00, 0x02, 0x00, 0x22, 0x56, 0x00, 0x00, 0x88, 0x58,
			0x01, 0x00, 0x04, 0x00, 0x10, 0x00, 0x64, 0x61, 0x74, 0x61,
			0x00, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x24, 0x17,
			0x1e, 0xf3, 0x3c, 0x13, 0x3c, 0x14, 0x16, 0xf9, 0x18, 0xf9,
			0x34, 0xe7, 0x23, 0xa6, 0x3c, 0xf2, 0x24, 0xf2, 0x11, 0xce, 0x1a, 0x0d}
	*/

	header, err := ExtractWavHeader(data)
	if err != nil {
		return err
	}

	fmt.Printf("%+v", header)

	return nil
}

func ExtractWavHeader(headerData []byte) (*WavHeader, error) {
	if len(headerData) < 44 {
		return nil, IncorrectWavFormat{}
	}

	header := &WavHeader{}

	header.ChunkID = string(headerData[:4])
	if header.ChunkID != "RIFF" {
		return nil, IncorrectWavFormat{}
	}
	header.ChunkSize = binary.LittleEndian.Uint32(headerData[4:8])
	if int(header.ChunkSize+8) > len(headerData) {
		return nil, IncorrectWavFormat{}
	}
	header.Format = string(headerData[8:12])
	if header.Format != "WAVE" {
		return nil, IncorrectWavFormat{}
	}

	//skip optional chunks that might be added before the fmt chunk
	offset, err := skipChunks(headerData, 12, "fmt ")
	if err != nil {
		return nil, err
	}

	header.Subchunk1ID = string(headerData[offset : offset+4])
	header.Subchunk1Size = binary.LittleEndian.Uint32(headerData[offset+4 : offset+8])
	if header.Subchunk1Size < 16 {
		return nil, IncorrectWavFormat{}
	}
	header.AudioFormat = binary.LittleEndian.Uint16(headerData[offset+8 : offset+10])
	header.NumChannels = binary.LittleEndian.Uint16(headerData[offset+10 : offset+12])
	header.SampleRate = binary.LittleEndian.Uint32(headerData[offset+12 : offset+16])
	header.ByteRate = binary.LittleEndian.Uint32(headerData[offset+16 : offset+20])
	header.BlockAlign = binary.LittleEndian.Uint16(headerData[offset+20 : offset+22])
	header.BitsPerSample = binary.LittleEndian.Uint16(headerData[offset+22 : offset+24])

	offset += 8 + int(header.Subchunk1Size)

	//skip the optional List chunk
	offset, err = skipChunks(headerData, offset, "data")
	if err != nil {
		return nil, err
	}
	header.Subchunk2ID = string(headerData[offset : offset+4])
	header.Subchunk2Size = binary.LittleEndian.Uint32(headerData[offset+4 : offset+8])

	//signify start of data
	header.DataOffset = uint32(offset) + 8
	if int(header.DataOffset)+int(header.Subchunk2Size) > len(headerData) {
		return nil, IncorrectWavFormat{}
	}

	return header, nil
}

func skipChunks(headerData []byte, offset int, target string) (int, error) {
	length := len(headerData)
	for {
		if offset+8 > length {
			return -1, IncorrectWavFormat{}
		}

		entry := string(headerData[offset : offset+4])
		if entry == target {
			return offset, nil
		}

		chunkLength := binary.LittleEndian.Uint32(headerData[offset+4 : offset+8])
		offset += 8 + int(chunkLength)
		if chunkLength%2 == 1 {
			offset++
		}
	}
}
