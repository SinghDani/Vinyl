package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/SinghDani/audioRecognition/db"
	"github.com/SinghDani/audioRecognition/fingerprint"
	"github.com/SinghDani/audioRecognition/internal"
	"github.com/SinghDani/audioRecognition/wav"
	"github.com/gorilla/websocket"
)

type Server struct {
	Addr       string
	Db         *db.DBConnection
	WsUpgrader websocket.Upgrader
}

func NewServer(Addr string, Db *db.DBConnection) *Server {
	return &Server{
		Addr: Addr,
		Db:   Db,
		//TODO change this for production
		WsUpgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return r.Header.Get("origin") == "http://localhost:5173"
			},
		},
	}
}

func (s *Server) Run() {
	mux := http.NewServeMux()
	s.registerRoutes(mux)

	if err := http.ListenAndServe(s.Addr, mux); err != nil {
		log.Fatal(err)
	}
}

func (s *Server) registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /songs", s.songs)
	mux.HandleFunc("/recording", s.acceptRecording) //Todo check if this should be a GET
}

func (s *Server) songs(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	songs, err := s.Db.GetAllSongs()
	if err != nil {
		http.Error(w, "db access failed", http.StatusInternalServerError)
		return
	}

	//TODO look up cors setUP
	w.Header().Add("Access-Control-Allow-Origin", "http://localhost:5173")

	if err := json.NewEncoder(w).Encode(songs); err != nil {
		http.Error(w, "could not send songs", http.StatusInternalServerError)
		return
	}
}

func (s *Server) acceptRecording(w http.ResponseWriter, r *http.Request) {
	conn, err := s.WsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("upgrade failed:", err)
		return
	}

	type match struct {
		SongName string `json:"songName"`
		Artist   string `json:"artist"`
		Verdict  string `json:"verdict"`
	}

	totalTime := 40     // record for max of 40 sec
	totalBatchTime := 5 // process in batches of 5 sec
	windowsPerBatch := fingerprint.SecondsToWindows(float64(totalBatchTime))
	samplesPerBatch := totalBatchTime * fingerprint.SamplingRate

	var masterHashList [][]internal.GeneratedHash
	var matchingSong internal.MatchingSong
	buffer := make([]float64, 0, samplesPerBatch*2)
	chunkIndex := 0

	defer conn.Close()

	for chunkIndex*totalBatchTime < totalTime {
		_, p, err := conn.ReadMessage()
		if err != nil {
			fmt.Println("ws read failed:", err)
			return
		}

		samples, err := wav.BytesToSamples(p)
		if err != nil {
			fmt.Println("invalid audio payload:", err)
			closeMsg := websocket.FormatCloseMessage(websocket.CloseUnsupportedData, err.Error())
			_ = conn.WriteControl(websocket.CloseMessage, closeMsg, time.Now().Add(time.Second))
			return
		}

		buffer = append(buffer, samples...)
		for len(buffer) >= samplesPerBatch {
			chunkSamples := make([]float64, samplesPerBatch)
			copy(chunkSamples, buffer[:samplesPerBatch])
			buffer = buffer[samplesPerBatch:]

			timeOffset := chunkIndex * windowsPerBatch
			chunkIndex++
			genHashes, err := fingerprint.SamplesToHashes(chunkSamples, timeOffset)
			if err != nil {
				fmt.Println("hash extraction failed:", err)
				closeMsg := websocket.FormatCloseMessage(websocket.CloseInternalServerErr, err.Error())
				_ = conn.WriteControl(websocket.CloseMessage, closeMsg, time.Now().Add(time.Second))
				return
			}

			masterHashList = append(masterHashList, genHashes)

			batchWindow := 3 //only look at last 15 seconds
			var hashBatch []internal.GeneratedHash

			start := max(0, len(masterHashList)-batchWindow)
			for _, batch := range masterHashList[start:] {
				hashBatch = append(hashBatch, batch...)
			}

			matchingSong, err = fingerprint.IdentifyRecording(s.Db, hashBatch)
			if err != nil {
				fmt.Println("identify failed:", err)
				closeMsg := websocket.FormatCloseMessage(websocket.CloseInternalServerErr, err.Error())
				_ = conn.WriteControl(websocket.CloseMessage, closeMsg, time.Now().Add(time.Second))
				return
			}

			if fingerprint.EvalMatch(matchingSong) {
				fingerprint.PrintVerdict(matchingSong, s.Db)
				if err := conn.WriteJSON(match{SongName: matchingSong.Name, Artist: matchingSong.Artist, Verdict: fingerprint.GetVerdict(matchingSong)}); err != nil {
					fmt.Println("ws write winner failed:", err)
				}
				return
			}
		}
	}

	fingerprint.PrintVerdict(matchingSong, s.Db)
	if err := conn.WriteJSON(match{SongName: matchingSong.Name, Artist: matchingSong.Artist, Verdict: fingerprint.GetVerdict(matchingSong)}); err != nil {
		fmt.Println("ws write final result failed:", err)
	}
}
