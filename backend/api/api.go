package api

import (
	"encoding/json"
	"net/http"

	"github.com/SinghDani/audioRecognition/db"
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

func (s *Server) Run() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		songs, err := s.Db.GetAllSongs()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err := json.NewEncoder(w).Encode(songs); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
	http.ListenAndServe(":8080", mux)
}
