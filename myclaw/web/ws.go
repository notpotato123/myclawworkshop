package web

import (
	"encoding/json"
	"net/http"

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

	// Each client gets a buffered send channel. The writer goroutine drains it;
	// the reader goroutine feeds it via hub.Broadcast.
	sendCh := make(chan []byte, 64)
	s.hub.register(sendCh)

	// Writer goroutine: serialises writes to the WebSocket (gorilla requires
	// that writes come from a single goroutine).
	go func() {
		defer conn.Close()
		for msg := range sendCh {
			if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		}
	}()

	// Cleanup: unregister before close so Broadcast never touches a closed channel.
	defer func() {
		s.hub.unregister(sendCh)
		close(sendCh)
	}()

	// Reader loop: receive messages from the browser and hand them to the agent.
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

		s.msgCh <- agent.Message{
			Content: in.Content,
			Source:  "web",
			ReplyTo: func(text string) { s.hub.Broadcast(ChunkMsg(text)) },
			Done:    func() { s.hub.Broadcast(DoneMsg()) },
			OnTool:  func(name, status string) { s.hub.Broadcast(ToolMsg(name, status)) },
		}
	}
}
