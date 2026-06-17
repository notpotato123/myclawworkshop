package a2a

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// Registry holds known peers discovered at runtime.
type Registry struct {
	mu    sync.RWMutex
	peers map[string]AgentCard // key: base URL
}

// NewRegistry returns an empty peer Registry.
func NewRegistry() *Registry {
	return &Registry{peers: make(map[string]AgentCard)}
}

// Add stores a peer card keyed by its URL.
func (r *Registry) Add(card AgentCard) {
	r.mu.Lock()
	r.peers[card.URL] = card
	r.mu.Unlock()
}

// All returns a snapshot of all known peers.
func (r *Registry) All() []AgentCard {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]AgentCard, 0, len(r.peers))
	for _, c := range r.peers {
		out = append(out, c)
	}
	return out
}

// FindBySkill returns peers whose Skills list contains a case-insensitive
// substring match against skill.
func (r *Registry) FindBySkill(skill string) []AgentCard {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []AgentCard
	for _, c := range r.peers {
		for _, s := range c.Skills {
			if containsFold(s, skill) {
				out = append(out, c)
				break
			}
		}
	}
	return out
}

// Discover fetches the Agent Card from baseURL/.well-known/agent-card.json.
func Discover(baseURL string) (*AgentCard, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	url := baseURL + "/.well-known/agent-card.json"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching agent card from %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("agent card request to %s returned %d", url, resp.StatusCode)
	}

	var card AgentCard
	if err := json.NewDecoder(resp.Body).Decode(&card); err != nil {
		return nil, fmt.Errorf("decoding agent card: %w", err)
	}
	if card.URL == "" {
		card.URL = baseURL
	}
	return &card, nil
}

func containsFold(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	// manual case-insensitive contains to avoid importing strings here
	sl, subl := toLower(s), toLower(substr)
	for i := 0; i <= len(sl)-len(subl); i++ {
		if sl[i:i+len(subl)] == subl {
			return true
		}
	}
	return false
}

func toLower(s string) string {
	b := make([]byte, len(s))
	for i := range s {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		b[i] = c
	}
	return string(b)
}
