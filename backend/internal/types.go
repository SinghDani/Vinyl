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

type Song struct {
	Name   string `json:"name"`
	Artist string `json:"artist"`
}

type MatchingSong struct {
	Name       string
	Artist     string
	Confidence float64
	SongId     uint32
	Score      int
}
