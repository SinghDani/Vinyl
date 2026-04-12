package api

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/SinghDani/audioRecognition/db"
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
				return true
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
	mux.HandleFunc("/", s.home)
	mux.HandleFunc("/recording", s.acceptRecording)
}

func (s *Server) home(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	songs, err := s.Db.GetAllSongs()
	if err != nil {
		http.Error(w, "db access failed", http.StatusInternalServerError)
		return
	}
	if err := json.NewEncoder(w).Encode(songs); err != nil {
		http.Error(w, "could not send songs", http.StatusInternalServerError)
		return
	}
}

func (s *Server) acceptRecording(w http.ResponseWriter, r *http.Request) {
}
