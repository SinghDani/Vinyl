package main

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

func (I *IncorrectWavFormat) Error() string {
	return "Incorrect wav format"
}
