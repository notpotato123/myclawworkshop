// Package game manages state and background goroutines for the maze heist game.
package game

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"

	"myclaw/a2a"
)

func defaultGameServerURL() string {
	if u := os.Getenv("GAME_SERVER_URL"); u != "" {
		return u
	}
	return "http://localhost:9090"
}

// MessageSink accepts incoming game messages so the game package can feed the
// agent loop without creating an import cycle with the agent package.
type MessageSink interface {
	Send(source, content string)
}

// Position is a 2D grid coordinate.
type Position struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// JoinResponse is the server's response to POST /api/join.
type JoinResponse struct {
	ExplorerID    string   `json:"explorer_id"`
	Role          string   `json:"role"`
	Position      Position `json:"position"`
	RelayURL      string   `json:"relay_url"`
	GameServerURL string   `json:"game_server_url"`
}

// InboxMessage is a single entry from GET /api/inbox/{explorer_id}.
type InboxMessage struct {
	From    string `json:"from"`
	Content string `json:"content"`
	SentAt  string `json:"sent_at"`
}

// PeerEntry is one entry from GET /api/peers.
type PeerEntry struct {
	ExplorerID   string `json:"explorer_id"`
	AgentCardURL string `json:"agent_card_url"`
	RelayURL     string `json:"relay_url"`
}

// State holds everything learned after joining the game.
type State struct {
	mu sync.RWMutex

	ExplorerID    string
	Role          string
	Position      Position
	RelayURL      string
	GameServerURL string

	joined bool
}

// Join POSTs to {gameServerURL}/api/join and populates State.
// agentCardURL is the public URL of this claw's agent card endpoint.
func (s *State) Join(gameServerURL, agentCardURL string) error {
	body, _ := json.Marshal(map[string]string{"agent_card_url": agentCardURL})
	resp, err := http.Post(gameServerURL+"/api/join", "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("join request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("join returned %d", resp.StatusCode)
	}

	var jr JoinResponse
	if err := json.NewDecoder(resp.Body).Decode(&jr); err != nil {
		return fmt.Errorf("decoding join response: %w", err)
	}

	s.mu.Lock()
	s.ExplorerID = jr.ExplorerID
	s.Role = jr.Role
	s.Position = jr.Position
	s.RelayURL = jr.RelayURL
	if jr.GameServerURL != "" {
		s.GameServerURL = jr.GameServerURL
	} else {
		s.GameServerURL = gameServerURL
	}
	s.joined = true
	s.mu.Unlock()
	slog.Info("game server URL", "url", s.GameServerURL)
	return nil
}

// Joined reports whether the agent has successfully joined the game.
func (s *State) Joined() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.joined
}

// Snapshot returns a copy of the current state fields (safe to call concurrently).
// gameURL falls back to GAME_SERVER_URL env var if not set from the join response.
func (s *State) Snapshot() (explorerID, role string, pos Position, gameURL string) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	gameURL = s.GameServerURL
	if gameURL == "" {
		gameURL = defaultGameServerURL()
	}
	return s.ExplorerID, s.Role, s.Position, gameURL
}

// PollInbox polls GET {gameServerURL}/api/inbox/{explorerID} every interval
// and calls sink.Send for each new message. Runs until ctx is cancelled.
func (s *State) PollInbox(ctx context.Context, interval time.Duration, sink MessageSink) {
	s.pollInbox(ctx, interval, func(from, content string) { sink.Send(from, content) })
}

func (s *State) pollInbox(ctx context.Context, interval time.Duration, onMessage func(from, content string)) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			id, _, _, gameURL := s.Snapshot()
			if id == "" {
				continue
			}
			msgs, err := fetchInbox(ctx, gameURL, id)
			if err != nil {
				slog.Warn("inbox poll failed", "err", err)
				continue
			}
			for _, m := range msgs {
				onMessage(m.From, m.Content)
			}
		}
	}
}

// RefreshPeers polls GET {gameServerURL}/api/peers every interval and adds
// newly discovered peers to registry. Runs until ctx is cancelled.
func (s *State) RefreshPeers(ctx context.Context, interval time.Duration, registry *a2a.Registry) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_, _, _, gameURL := s.Snapshot()
			if gameURL == "" {
				continue
			}
			peers, err := fetchPeers(ctx, gameURL)
			if err != nil {
				slog.Warn("peer refresh failed", "err", err)
				continue
			}
			for _, p := range peers {
				if p.AgentCardURL == "" {
					continue
				}
				// Use relay URL as the card's URL so ask_peer works without
				// direct connectivity.
				card, err := a2a.Discover(p.AgentCardURL)
				if err != nil {
					slog.Warn("could not fetch peer card", "url", p.AgentCardURL, "err", err)
					// Register with just the relay URL so we can still reach them.
					if p.RelayURL != "" {
						registry.Add(a2a.AgentCard{
							Name: p.ExplorerID,
							URL:  p.RelayURL,
						})
					}
					continue
				}
				if p.RelayURL != "" {
					card.URL = p.RelayURL
				}
				registry.Add(*card)
			}
		}
	}
}

func fetchInbox(ctx context.Context, gameURL, explorerID string) ([]InboxMessage, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		gameURL+"/api/inbox/"+explorerID, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("inbox returned %d", resp.StatusCode)
	}
	var msgs []InboxMessage
	return msgs, json.NewDecoder(resp.Body).Decode(&msgs)
}

func fetchPeers(ctx context.Context, gameURL string) ([]PeerEntry, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, gameURL+"/api/peers", nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("peers returned %d", resp.StatusCode)
	}
	var peers []PeerEntry
	return peers, json.NewDecoder(resp.Body).Decode(&peers)
}
