package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"myclaw/a2a"
	"myclaw/game"
	"myclaw/msgs"
)

const (
	inboxPollInterval   = 3 * time.Second
	peerRefreshInterval = 30 * time.Second
)

// agentSink adapts a msgs.Message channel to game.MessageSink.
type agentSink struct{ ch chan<- msgs.Message }

func (s agentSink) Send(from, content string) {
	s.ch <- msgs.Message{
		Content: fmt.Sprintf("[inbox from %s]: %s", from, content),
		Source:  "inbox",
		ReplyTo: func(string) {},
		Done:    func() {},
		OnTool:  func(_, _ string) {},
	}
}

// JoinGame joins the maze heist game server, starts the inbox poller, and
// starts the peer-refresh goroutine.
type JoinGame struct {
	State    *game.State
	Registry *a2a.Registry
	MsgCh    chan<- msgs.Message
	AppCtx   context.Context // app-lifetime context for background goroutines
}

func (t JoinGame) Name() string { return "join_game" }
func (t JoinGame) Description() string {
	return `Join the maze heist game server. Provide the game server URL and this claw's public base URL.
After joining you'll receive your explorer_id, role, and starting position.
The inbox poller and peer-refresh goroutines start automatically.`
}
func (t JoinGame) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"game_server_url": map[string]any{
				"type":        "string",
				"description": "Base URL of the game server, e.g. http://game.example.com",
			},
			"agent_card_url": map[string]any{
				"type":        "string",
				"description": "Public URL where this claw's agent card is reachable, e.g. http://my-ip:8080",
			},
		},
		"required":             []string{"game_server_url", "agent_card_url"},
		"additionalProperties": false,
	}
}

func (t JoinGame) Execute(_ context.Context, params json.RawMessage) (string, error) {
	var p struct {
		GameServerURL string `json:"game_server_url"`
		AgentCardURL  string `json:"agent_card_url"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return "", fmt.Errorf("invalid parameters: %w", err)
	}
	if p.GameServerURL == "" {
		return "", fmt.Errorf("game_server_url is required")
	}
	if p.AgentCardURL == "" {
		return "", fmt.Errorf("agent_card_url is required")
	}

	if t.State.Joined() {
		id, role, pos, _ := t.State.Snapshot()
		return fmt.Sprintf("Already joined as %q (role: %s, position: %d,%d).", id, role, pos.X, pos.Y), nil
	}

	if err := t.State.Join(p.GameServerURL, p.AgentCardURL); err != nil {
		return "", fmt.Errorf("joining game: %w", err)
	}

	id, role, pos, _ := t.State.Snapshot()
	slog.Info("joined game", "explorer_id", id, "role", role, "position", pos)

	// Start inbox poller — delivers incoming messages to the agent loop.
	go t.State.PollInbox(t.AppCtx, inboxPollInterval, agentSink{t.MsgCh})

	// Start peer refresh — keeps the peer registry current.
	go t.State.RefreshPeers(t.AppCtx, peerRefreshInterval, t.Registry)

	return fmt.Sprintf(
		"Joined! explorer_id: %s | role: %s | position: (%d, %d)\nInbox poller and peer refresh started.",
		id, role, pos.X, pos.Y,
	), nil
}
