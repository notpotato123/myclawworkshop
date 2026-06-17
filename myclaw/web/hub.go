package web

import (
	"encoding/json"
	"sync"
)

// Hub manages all active WebSocket connections and broadcasts JSON messages to
// all of them concurrently.
type Hub struct {
	mu      sync.RWMutex
	clients map[chan []byte]struct{}
}

// NewHub returns a ready-to-use Hub.
func NewHub() *Hub {
	return &Hub{clients: make(map[chan []byte]struct{})}
}

func (h *Hub) register(ch chan []byte) {
	h.mu.Lock()
	h.clients[ch] = struct{}{}
	h.mu.Unlock()
}

// unregister removes ch before the caller closes it, preventing Broadcast from
// ever sending to a closed channel (Broadcast holds the read lock for its
// entire duration, so the write lock here guarantees mutual exclusion).
func (h *Hub) unregister(ch chan []byte) {
	h.mu.Lock()
	delete(h.clients, ch)
	h.mu.Unlock()
}

// Broadcast sends msg to all registered clients. Slow clients are skipped
// (non-blocking send) rather than blocking the caller.
func (h *Hub) Broadcast(msg []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for ch := range h.clients {
		select {
		case ch <- msg:
		default:
		}
	}
}

// ── JSON message constructors ─────────────────────────────────────────────────

type wsMsg struct {
	Type    string `json:"type"`
	Content string `json:"content,omitempty"`
	Name    string `json:"name,omitempty"`
	Status  string `json:"status,omitempty"`
}

func encodeMsg(m wsMsg) []byte {
	b, _ := json.Marshal(m)
	return b
}

// ChunkMsg is a streaming text token sent to the client.
func ChunkMsg(content string) []byte { return encodeMsg(wsMsg{Type: "chunk", Content: content}) }

// DoneMsg signals that the agent has finished its response.
func DoneMsg() []byte { return encodeMsg(wsMsg{Type: "done"}) }

// SystemMsg is an out-of-band notification (e.g. a scheduled task firing).
func SystemMsg(content string) []byte { return encodeMsg(wsMsg{Type: "system", Content: content}) }

// ToolMsg encodes a tool_call or tool_result event depending on status.
func ToolMsg(name, status string) []byte {
	typ := "tool_call"
	if status == "done" || status == "error" || status == "unknown" {
		typ = "tool_result"
	}
	return encodeMsg(wsMsg{Type: typ, Name: name, Status: status})
}
