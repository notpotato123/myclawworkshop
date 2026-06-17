package web

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"strings"

	"myclaw/a2a"
	"myclaw/agent"
)

//go:embed static system_prompt.md
var staticFiles embed.FS

// SystemPrompt returns the embedded system prompt text.
func SystemPrompt() string {
	b, err := staticFiles.ReadFile("system_prompt.md")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

// Server owns the HTTP mux, the WebSocket hub, and a reference to the agent's
// message channel.
type Server struct {
	hub     *Hub
	msgCh   chan<- agent.Message
	httpSrv *http.Server
	port    string
	// sendA2A is called by the A2A handler to synchronously process a message
	// and return the full response. Set via SetA2ASender before Start.
	sendA2A func(text string) (string, error)
}

// NewServer creates a Server. hub is shared with callers so they can
// broadcast to WebSocket clients from outside the web package (e.g. scheduler
// callbacks in main.go).
func NewServer(hub *Hub, msgCh chan<- agent.Message, port string) *Server {
	return &Server{hub: hub, msgCh: msgCh, port: port}
}

// SetA2ASender installs the function used by the A2A handler to synchronously
// send a message through the agent and collect the full response.
func (s *Server) SetA2ASender(fn func(text string) (string, error)) {
	s.sendA2A = fn
}

// Start begins serving HTTP and WebSocket traffic on the given port. It blocks
// until the server fails and is intended to be run in its own goroutine.
func (s *Server) Start(port string) error {
	sub, err := fs.Sub(staticFiles, "static")
	if err != nil {
		return fmt.Errorf("preparing static files: %w", err)
	}

	card := a2a.NewClawCard(s.port)

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", s.handleWS)
	mux.HandleFunc("/.well-known/agent-card.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(card)
	})
	if s.sendA2A != nil {
		mux.Handle("/a2a", a2a.NewHandler(s.sendA2A))
	}
	mux.Handle("/", http.FileServer(http.FS(sub)))

	s.httpSrv = &http.Server{Addr: ":" + port, Handler: mux}
	fmt.Printf("Web UI: http://localhost:%s\n", port)
	if err := s.httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

// Shutdown gracefully stops the HTTP server and closes all WebSocket clients.
func (s *Server) Shutdown(ctx context.Context) {
	s.hub.CloseAll()
	if s.httpSrv != nil {
		s.httpSrv.Shutdown(ctx) //nolint:errcheck
	}
}
