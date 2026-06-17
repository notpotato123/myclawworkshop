package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"myclaw/a2a"
)

// ── discover_peer ─────────────────────────────────────────────────────────────

type DiscoverPeer struct{ Registry *a2a.Registry }

func (t DiscoverPeer) Name() string { return "discover_peer" }
func (t DiscoverPeer) Description() string {
	return "Fetch an Agent Card from a peer's base URL, add them to the local peer registry, and return their name, description, and skills."
}
func (t DiscoverPeer) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"url": map[string]any{
				"type":        "string",
				"description": "Base URL of the peer agent (e.g. http://host:8080).",
			},
		},
		"required":             []string{"url"},
		"additionalProperties": false,
	}
}
func (t DiscoverPeer) Execute(_ context.Context, params json.RawMessage) (string, error) {
	var p struct{ URL string `json:"url"` }
	if err := json.Unmarshal(params, &p); err != nil {
		return "", fmt.Errorf("invalid parameters: %w", err)
	}
	if p.URL == "" {
		return "", fmt.Errorf("url is required")
	}
	card, err := a2a.Discover(p.URL)
	if err != nil {
		return "", err
	}
	t.Registry.Add(*card)
	return fmt.Sprintf("Discovered %q at %s\nDescription: %s\nSkills: %s",
		card.Name, card.URL, card.Description, strings.Join(card.Skills, ", ")), nil
}

// ── ask_peer ──────────────────────────────────────────────────────────────────

type AskPeer struct{ Registry *a2a.Registry }

func (t AskPeer) Name() string { return "ask_peer" }
func (t AskPeer) Description() string {
	return "Send a message to a peer agent and return their response. peer_url is the peer's base URL or relay URL."
}
func (t AskPeer) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"peer_url": map[string]any{
				"type":        "string",
				"description": "Base URL of the peer (e.g. http://host:8080) or a relay URL.",
			},
			"message": map[string]any{
				"type":        "string",
				"description": "The message to send to the peer.",
			},
		},
		"required":             []string{"peer_url", "message"},
		"additionalProperties": false,
	}
}
func (t AskPeer) Execute(ctx context.Context, params json.RawMessage) (string, error) {
	var p struct {
		PeerURL string `json:"peer_url"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return "", fmt.Errorf("invalid parameters: %w", err)
	}
	if p.PeerURL == "" {
		return "", fmt.Errorf("peer_url is required")
	}
	if p.Message == "" {
		return "", fmt.Errorf("message is required")
	}

	// If peer_url ends with /a2a it's already an endpoint; otherwise append it.
	endpoint := p.PeerURL
	if !strings.HasSuffix(endpoint, "/a2a") {
		endpoint = a2a.A2AEndpoint(endpoint)
	}

	reply, err := a2a.SendMessage(ctx, endpoint, p.Message)
	if err != nil {
		return "", err
	}
	return reply, nil
}

// ── broadcast ─────────────────────────────────────────────────────────────────

type Broadcast struct{ Registry *a2a.Registry }

func (t Broadcast) Name() string { return "broadcast" }
func (t Broadcast) Description() string {
	return "Send a message to ALL discovered peers in parallel and return a summary of their responses."
}
func (t Broadcast) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"message": map[string]any{
				"type":        "string",
				"description": "The message to broadcast to all peers.",
			},
		},
		"required":             []string{"message"},
		"additionalProperties": false,
	}
}

type peerResult struct {
	name  string
	reply string
	err   error
}

func (t Broadcast) Execute(ctx context.Context, params json.RawMessage) (string, error) {
	var p struct{ Message string `json:"message"` }
	if err := json.Unmarshal(params, &p); err != nil {
		return "", fmt.Errorf("invalid parameters: %w", err)
	}
	if p.Message == "" {
		return "", fmt.Errorf("message is required")
	}

	peers := t.Registry.All()
	if len(peers) == 0 {
		return "No peers discovered yet. Use discover_peer first.", nil
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	resultCh := make(chan peerResult, len(peers))
	var wg sync.WaitGroup
	for _, card := range peers {
		wg.Add(1)
		go func(c a2a.AgentCard) {
			defer wg.Done()
			reply, err := a2a.SendMessage(ctx, a2a.A2AEndpoint(c.URL), p.Message)
			resultCh <- peerResult{name: c.Name, reply: reply, err: err}
		}(card)
	}

	// Close channel once all goroutines finish.
	go func() { wg.Wait(); close(resultCh) }()

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Broadcast to %d peer(s):\n", len(peers)))
	for r := range resultCh {
		if r.err != nil {
			sb.WriteString(fmt.Sprintf("  [%s] ERROR: %v\n", r.name, r.err))
		} else {
			sb.WriteString(fmt.Sprintf("  [%s]: %s\n", r.name, r.reply))
		}
	}
	return sb.String(), nil
}

// ── find_peer_with_skill ──────────────────────────────────────────────────────

type FindPeerWithSkill struct{ Registry *a2a.Registry }

func (t FindPeerWithSkill) Name() string { return "find_peer_with_skill" }
func (t FindPeerWithSkill) Description() string {
	return "Search discovered peers for agents that have a matching skill. Returns peer names and URLs."
}
func (t FindPeerWithSkill) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"skill": map[string]any{
				"type":        "string",
				"description": "Skill name or substring to search for (case-insensitive).",
			},
		},
		"required":             []string{"skill"},
		"additionalProperties": false,
	}
}
func (t FindPeerWithSkill) Execute(_ context.Context, params json.RawMessage) (string, error) {
	var p struct{ Skill string `json:"skill"` }
	if err := json.Unmarshal(params, &p); err != nil {
		return "", fmt.Errorf("invalid parameters: %w", err)
	}
	if p.Skill == "" {
		return "", fmt.Errorf("skill is required")
	}

	matches := t.Registry.FindBySkill(p.Skill)
	if len(matches) == 0 {
		return fmt.Sprintf("No peers found with skill %q.", p.Skill), nil
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%d peer(s) with skill %q:\n", len(matches), p.Skill))
	for _, c := range matches {
		sb.WriteString(fmt.Sprintf("  %s — %s\n", c.Name, c.URL))
	}
	return sb.String(), nil
}
