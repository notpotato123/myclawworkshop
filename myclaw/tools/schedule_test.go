package tools

import (
	"path/filepath"
	"strings"
	"testing"

	"myclaw/scheduler"
)

func newTestSched(t *testing.T) *scheduler.Scheduler {
	t.Helper()
	s, err := scheduler.New(filepath.Join(t.TempDir(), "tasks.json"), nil)
	if err != nil {
		t.Fatalf("scheduler.New: %v", err)
	}
	return s
}

func TestScheduleCreateAndList(t *testing.T) {
	tool := Schedule{Sched: newTestSched(t)}

	out, err := runTool(t, tool, map[string]any{"action": "schedule", "description": "say hello", "delay": "1h"})
	if err != nil {
		t.Fatalf("schedule: %v", err)
	}
	if !strings.Contains(out, "scheduled") {
		t.Fatalf("expected confirmation, got %q", out)
	}

	out, err = runTool(t, tool, map[string]any{"action": "list"})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if !strings.Contains(out, "say hello") {
		t.Fatalf("expected task in listing, got %q", out)
	}
}

func TestScheduleListEmpty(t *testing.T) {
	tool := Schedule{Sched: newTestSched(t)}
	out, err := runTool(t, tool, map[string]any{"action": "list"})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if !strings.Contains(strings.ToLower(out), "no tasks") {
		t.Fatalf("expected empty message, got %q", out)
	}
}

func TestScheduleRecurringFlag(t *testing.T) {
	tool := Schedule{Sched: newTestSched(t)}
	out, err := runTool(t, tool, map[string]any{"action": "schedule", "description": "tick", "delay": "30m", "recurring": true})
	if err != nil {
		t.Fatalf("schedule: %v", err)
	}
	if !strings.Contains(out, "recurring") {
		t.Fatalf("expected recurring in confirmation, got %q", out)
	}
}

func TestScheduleRequiresDescription(t *testing.T) {
	tool := Schedule{Sched: newTestSched(t)}
	if _, err := runTool(t, tool, map[string]any{"action": "schedule", "delay": "1h"}); err == nil {
		t.Fatal("expected error when description is missing")
	}
}

func TestScheduleRequiresDelay(t *testing.T) {
	tool := Schedule{Sched: newTestSched(t)}
	if _, err := runTool(t, tool, map[string]any{"action": "schedule", "description": "x"}); err == nil {
		t.Fatal("expected error when delay is missing")
	}
}

func TestScheduleInvalidDelay(t *testing.T) {
	tool := Schedule{Sched: newTestSched(t)}
	if _, err := runTool(t, tool, map[string]any{"action": "schedule", "description": "x", "delay": "notaduration"}); err == nil {
		t.Fatal("expected error for invalid delay")
	}
}

func TestScheduleUnknownAction(t *testing.T) {
	tool := Schedule{Sched: newTestSched(t)}
	if _, err := runTool(t, tool, map[string]any{"action": "delete"}); err == nil {
		t.Fatal("expected error for unknown action")
	}
}
