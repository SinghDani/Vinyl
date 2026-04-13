package wav

import (
	"encoding/binary"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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

type WAV struct {
	Header  *WavHeader
	Samples []float64
}

type IncorrectWavFormat struct{ message string }

func WavToSamples(file string) (*WAV, error) {
	//TODO clean up audioFiless and donwsample files, probably don't want to keep that
	outPath := "./audioFiles/downsampledWAVs/" + filepath.Base(file)

	cmd := exec.Command(
		"ffmpeg",
		"-y",
		"-i", file,
		"-ac", "1",
		"-ar", "11025",
		"-c:a", "pcm_s16le",
		outPath,
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(string(out))
		return nil, err
	}

	return ParseWav(outPath)
	/*
		if wavData.header.NumChannels == 2 {
			wavData.samples, err = stereoToMono(wavData.samples)
			if err != nil {
				return nil, err
			}
		}

		//#TODO use ffmpeg rather
		wavData.samples = lowpass(wavData.samples, int(wavData.header.SampleRate), 5000)
		wavData.samples = downsample(wavData.samples, 4)
	*/
}

func (I IncorrectWavFormat) Error() string {
	return "Incorrect Wav Format: " + I.message
}

func ParseWav(path string) (*WAV, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	/*
		data = []byte{0x52, 0x49, 0x46, 0x46, 0x24, 0x08, 0x00, 0x00, 0x57, 0x41,
			0x56, 0x45, 0x66, 0x6d, 0x74, 0x20, 0x10, 0x00, 0x00, 0x00,
			0x01, 0x00, 0x02, 0x00, 0x22, 0x56, 0x00, 0x00, 0x88, 0x58,
			0x01, 0x00, 0x04, 0x00, 0x10, 0x00, 0x64, 0x61, 0x74, 0x61,
			0x00, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x24, 0x17,
			0x1e, 0xf3, 0x3c, 0x13, 0x3c, 0x14, 0x16, 0xf9, 0x18, 0xf9,
			0x34, 0xe7, 0x23, 0xa6, 0x3c, 0xf2, 0x24, 0xf2, 0x11, 0xce,
			0x00, 0x80}
	*/
	header, err := extractWavHeader(data)
	if err != nil {
		return nil, err
	}
	fmt.Printf("%+v\n", header)

	samples, err := BytesToSamples(data[header.DataOffset:])
	if err != nil {
		return nil, err
	}

	return &WAV{header, samples}, nil
}

func stereoToMono(samples []float64) ([]float64, error) {
	length := len(samples)
	if length%2 != 0 {
		return nil, IncorrectWavFormat{"odd sample size"}
	}
	res := make([]float64, length/2)
	for i := 0; i < length; i += 2 {
		res[i/2] = (samples[i] + samples[i+1]) / 2
	}
	return res, nil
}

func BytesToSamples(rawData []byte) ([]float64, error) {
	length := len(rawData)
	if length%2 != 0 {
		return nil, IncorrectWavFormat{"odd sample size"}
	}

	samples := make([]float64, length/2)
	for i := 0; i < length; i += 2 {
		//only wav files with 16bit samples will be processed
		bits := int16(binary.LittleEndian.Uint16(rawData[i : i+2]))

		//divding by 2^15 = 32768 (because of int16) to
		//normalise the data from int to float between [-1, 1]
		samples[i/2] = float64(bits) / 32768
	}
	return samples, nil
}

func extractWavHeader(headerData []byte) (*WavHeader, error) {
	if len(headerData) < 44 {
		return nil, IncorrectWavFormat{"header lenght < 44"}
	}

	header := &WavHeader{}

	header.ChunkID = string(headerData[:4])
	if header.ChunkID != "RIFF" {
		return nil, IncorrectWavFormat{"RIFF not contained"}
	}
	header.ChunkSize = binary.LittleEndian.Uint32(headerData[4:8])
	if int(header.ChunkSize+8) > len(headerData) {
		return nil, IncorrectWavFormat{"Chunksize exceeds length of WAV file"}
	}
	header.Format = string(headerData[8:12])
	if header.Format != "WAVE" {
		return nil, IncorrectWavFormat{"WAVE not included"}
	}

	//skip optional chunks that might be added before the fmt chunk
	offset, err := skipChunks(headerData, 12, "fmt ")
	if err != nil {
		return nil, err
	}

	header.Subchunk1ID = string(headerData[offset : offset+4])
	header.Subchunk1Size = binary.LittleEndian.Uint32(headerData[offset+4 : offset+8])
	if header.Subchunk1Size < 16 {
		return nil, IncorrectWavFormat{"Subchunk1Size < 16"}
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
		return nil, IncorrectWavFormat{"Subchunk2Size exceeds length of WAV file"}
	}
	return header, nil
}

func skipChunks(headerData []byte, offset int, target string) (int, error) {
	length := len(headerData)
	for {
		if offset+8 > length {
			return -1, IncorrectWavFormat{"An optional chunk exceeds the WAV file length"}
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
