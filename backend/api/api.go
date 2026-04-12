package api

import (
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
