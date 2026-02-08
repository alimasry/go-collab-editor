package server

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// NewHandler creates the HTTP handler with all routes.
func NewHandler(hub *Hub) http.Handler {
	mux := http.NewServeMux()

	// Serve static files.
	fs := http.FileServer(http.Dir("static"))
	mux.Handle("/", fs)

	// WebSocket endpoint.
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("websocket upgrade error: %v", err)
			return
		}
		client := newClient(hub, conn)
		go client.WritePump()
		go client.ReadPump()
	})

	return mux
}
