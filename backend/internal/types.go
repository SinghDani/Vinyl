package internal

type Fingerprint struct {
	Hash       uint32 // anchor freq | target freq | dt
	SongId     uint32
	AnchorTime uint32
}

type GeneratedHash struct {
	Hash       uint32 // anchor freq | target freq | dt
	AnchorTime uint32
}

type MatchingSong struct {
	SongName   string
	Confidence float64
	SongId     uint32
	Score      int
}
