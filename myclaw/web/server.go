package web

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"

	"myclaw/agent"
)

//go:embed static
var staticFiles embed.FS

// Server owns the HTTP mux, the WebSocket hub, and a reference to the agent's
// message channel.
type Server struct {
	hub   *Hub
	msgCh chan<- agent.Message
}

// NewServer creates a Server. hub is shared with callers so they can
// broadcast to WebSocket clients from outside the web package (e.g. scheduler
// callbacks in main.go).
func NewServer(hub *Hub, msgCh chan<- agent.Message) *Server {
	return &Server{hub: hub, msgCh: msgCh}
}

// Start begins serving HTTP and WebSocket traffic on the given port. It blocks
// until the server fails and is intended to be run in its own goroutine.
func (s *Server) Start(port string) error {
	sub, err := fs.Sub(staticFiles, "static")
	if err != nil {
		return fmt.Errorf("preparing static files: %w", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", s.handleWS)
	mux.Handle("/", http.FileServer(http.FS(sub)))

	addr := ":" + port
	fmt.Printf("Web UI: http://localhost%s\n", addr)
	return http.ListenAndServe(addr, mux)
}
