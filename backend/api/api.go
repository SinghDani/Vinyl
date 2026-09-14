package api

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/SinghDani/audioRecognition/db"
	"github.com/SinghDani/audioRecognition/fingerprint"
	"github.com/SinghDani/audioRecognition/internal"
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
	}
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Access-Control-Allow-Origin", "http://localhost:5173")
		next.ServeHTTP(w, r)
	})
}

func (s *Server) Run() {
	mux := http.NewServeMux()
	s.registerRoutes(mux)

	if err := http.ListenAndServe(s.Addr, cors(mux)); err != nil {
		log.Fatal(err)
	}
}

func (s *Server) registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /songs", s.getSongs)
	//mux.HandleFunc("/recording", s.acceptRecording)
	mux.HandleFunc("POST /song", s.getMatchinSong)
}

func (s *Server) getMatchinSong(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var hashes []internal.GeneratedHash
	if err := json.NewDecoder(r.Body).Decode(&hashes); err != nil {
		http.Error(w, "could not decode hashes", http.StatusInternalServerError)
		return
	}

	song, err := fingerprint.IdentifyRecording(s.Db, hashes)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	res := internal.Match{
		SongName: song.Name, Artist: song.Artist, Match: fingerprint.EvalMatch(song), Verdict: fingerprint.GetVerdict(song),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(res); err != nil {
		http.Error(w, "could not send song", http.StatusInternalServerError)
		return
	}
}

func (s *Server) getSongs(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	songs, err := s.Db.GetAllSongs()
	if err != nil {
		http.Error(w, "db access failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(songs); err != nil {
		http.Error(w, "could not send songs", http.StatusInternalServerError)
		return
	}
}
