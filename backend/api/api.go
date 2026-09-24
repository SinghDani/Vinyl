package api

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/SinghDani/audioRecognition/db"
	"github.com/SinghDani/audioRecognition/fingerprint"
	"github.com/SinghDani/audioRecognition/internal"
)

type Server struct {
	Addr string
	Db   *db.DBConnection
}

func NewServer(Addr string, Db *db.DBConnection) *Server {
	return &Server{
		Addr: Addr,
		Db:   Db,
	}
}

func cors(next http.Handler) http.Handler {
	allowedOrigin := os.Getenv("FRONTEND_ORIGIN")
	if allowedOrigin == "" {
		allowedOrigin = "http://localhost:5173"
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Access-Control-Allow-Origin", allowedOrigin)
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
	mux.HandleFunc("GET /api/songs", s.getSongs)
	//mux.HandleFunc("/recording", s.acceptRecording)
	mux.HandleFunc("POST /api/song", s.getMatchinSong)
}

func (s *Server) getMatchinSong(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var hashes []internal.GeneratedHash
	if err := json.NewDecoder(r.Body).Decode(&hashes); err != nil {
		http.Error(w, "could not decode hashes", http.StatusInternalServerError)
		return
	}

	song, err := fingerprint.IdentifyRecording(r.Context(), s.Db, hashes)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
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

	songs, err := s.Db.GetAllSongs(r.Context())
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
