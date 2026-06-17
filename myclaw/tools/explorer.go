package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand"
	"strings"
	"sync"
	"time"
)

type Explorer struct {
	ExplorerID *string // shared pointer, set after join

	visited map[string]int // "x,y" -> visit count
	lastDir string
	mu      sync.Mutex
}

func (e *Explorer) Start(ctx context.Context) {
	e.mu.Lock()
	e.visited = make(map[string]int)
	e.mu.Unlock()

	slog.Info("explorer started")

	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			e.tick(ctx)
		}
	}
}

func (e *Explorer) tick(ctx context.Context) {
	id := ""
	if e.ExplorerID != nil {
		id = *e.ExplorerID
	}
	if id == "" {
		return
	}

	// 1. Look (plain HTTP, no LLM)
	result, err := gamePost(ctx, "/api/look", map[string]string{"explorer_id": id})
	if err != nil {
		return
	}

	var lookResp struct {
		Description string `json:"description"`
	}
	if err := json.Unmarshal([]byte(result), &lookResp); err != nil {
		return
	}

	// 2. Parse exits from the text description
	exits := parseExits(lookResp.Description)
	if len(exits) == 0 {
		return
	}

	// 3. Track position and pick direction
	pos := parsePosition(lookResp.Description)
	e.mu.Lock()
	e.visited[pos]++
	dir := e.pickDirection(exits, pos)
	e.lastDir = dir
	e.mu.Unlock()

	// 4. Move (plain HTTP, no LLM)
	gamePost(ctx, "/api/move", map[string]string{ //nolint:errcheck
		"explorer_id": id,
		"direction":   dir,
	})
}

// parseExits extracts open passage directions from the look description.
func parseExits(desc string) []string {
	var exits []string
	inExits := false
	for _, line := range strings.Split(desc, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "Exits:" {
			inExits = true
			continue
		}
		if inExits {
			if trimmed == "" || (!strings.HasPrefix(trimmed, "-") && !strings.HasPrefix(trimmed, "None")) {
				break
			}
			if strings.Contains(trimmed, "open passage") {
				for _, dir := range []string{"north", "south", "east", "west"} {
					if strings.HasPrefix(trimmed, "- "+dir+":") {
						exits = append(exits, dir)
					}
				}
			}
		}
	}
	return exits
}

// parsePosition extracts "x,y" from "at position (x, y)".
func parsePosition(desc string) string {
	idx := strings.Index(desc, "at position (")
	if idx == -1 {
		return "0,0"
	}
	rest := desc[idx+len("at position ("):]
	end := strings.Index(rest, ")")
	if end == -1 {
		return "0,0"
	}
	return strings.ReplaceAll(rest[:end], " ", "")
}

func reverse(dir string) string {
	switch dir {
	case "north":
		return "south"
	case "south":
		return "north"
	case "east":
		return "west"
	case "west":
		return "east"
	}
	return ""
}

// pickDirection prefers unvisited cells and avoids immediate backtracking.
func (e *Explorer) pickDirection(exits []string, currentPos string) string {
	// Filter out reverse of last direction
	rev := reverse(e.lastDir)
	candidates := make([]string, 0, len(exits))
	for _, d := range exits {
		if d != rev {
			candidates = append(candidates, d)
		}
	}
	if len(candidates) == 0 {
		candidates = exits
	}

	// Score by visit count of destination
	minCount := -1
	for _, d := range candidates {
		dest := offsetPosition(currentPos, d)
		count := e.visited[dest]
		if minCount == -1 || count < minCount {
			minCount = count
		}
	}

	var best []string
	for _, d := range candidates {
		dest := offsetPosition(currentPos, d)
		if e.visited[dest] == minCount {
			best = append(best, d)
		}
	}
	return best[rand.Intn(len(best))]
}

func offsetPosition(pos, dir string) string {
	var x, y int
	fmt.Sscanf(pos, "%d,%d", &x, &y)
	switch dir {
	case "north":
		y--
	case "south":
		y++
	case "east":
		x++
	case "west":
		x--
	}
	return fmt.Sprintf("%d,%d", x, y)
}
