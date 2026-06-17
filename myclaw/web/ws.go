package web

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"myclaw/agent"
)

var upgrader = websocket.Upgrader{
	// Allow all origins for the workshop; tighten in production.
	CheckOrigin: func(r *http.Request) bool { return true },
}

func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	sendCh := make(chan []byte, 64)
	s.hub.register(sendCh)

	// closeOnce ensures sendCh is closed exactly once regardless of whether
	// the reader loop exits first or CloseAll fires during shutdown.
	var closeOnce sync.Once
	closeSend := func() {
		s.hub.unregister(sendCh)
		closeOnce.Do(func() { close(sendCh) })
	}

	// Writer goroutine: serialises writes to the WebSocket.
	go func() {
		defer conn.Close()
		for msg := range sendCh {
			if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				closeSend() // also drain the hub registration on write error
				return
			}
		}
	}()

	// Reader loop: receive messages from the browser and hand them to the agent.
	defer closeSend()

	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			return
		}

		var in struct {
			Type    string `json:"type"`
			Content string `json:"content"`
		}
		if err := json.Unmarshal(data, &in); err != nil || in.Type != "message" || in.Content == "" {
			continue
		}

		slog.Info("web message received", "content_len", len(in.Content))

		s.msgCh <- agent.Message{
			Content: in.Content,
			Source:  "web",
			ReplyTo: func(text string) { s.hub.Broadcast(ChunkMsg(text)) },
			Done:    func() { s.hub.Broadcast(DoneMsg()) },
			OnTool:  func(name, status string) { s.hub.Broadcast(ToolMsg(name, status)) },
		}
	}
}
