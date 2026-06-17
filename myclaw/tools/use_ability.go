package tools

import (
	"context"
	"encoding/json"
	"fmt"
)

type UseAbility struct {
	ExplorerID *string
}

func (t UseAbility) Name() string { return "use_ability" }
func (t UseAbility) Description() string {
	return "Use your role's special ability on a target (e.g., a door ID). Only works if your role matches the door's requirement and you are close enough."
}

func (t UseAbility) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"target": map[string]any{
				"type":        "string",
				"description": "The target to use your ability on (e.g., a door ID like 'door-3').",
			},
		},
		"required":             []string{"target"},
		"additionalProperties": false,
	}
}

func (t UseAbility) Execute(ctx context.Context, params json.RawMessage) (string, error) {
	var p struct {
		Target string `json:"target"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return "", fmt.Errorf("invalid parameters: %w", err)
	}
	explorerID := ""
	if t.ExplorerID != nil {
		explorerID = *t.ExplorerID
	}
	return gamePost(ctx, "/api/ability", map[string]string{
		"explorer_id": explorerID,
		"target":      p.Target,
	})
}
