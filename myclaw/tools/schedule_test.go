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

func TestScheduleCancel(t *testing.T) {
	tool := Schedule{Sched: newTestSched(t)}

	// Schedule a task and grab its ID from the list.
	runTool(t, tool, map[string]any{"action": "schedule", "description": "to be cancelled", "delay": "1h"})
	listOut, _ := runTool(t, tool, map[string]any{"action": "list"})
	// Extract ID from the "[<id>]" in the list output.
	start := strings.Index(listOut, "[")
	end := strings.Index(listOut, "]")
	if start < 0 || end < 0 {
		t.Fatalf("could not find task ID in list output: %q", listOut)
	}
	id := listOut[start+1 : end]

	out, err := runTool(t, tool, map[string]any{"action": "cancel", "id": id})
	if err != nil {
		t.Fatalf("cancel: %v", err)
	}
	if !strings.Contains(out, "cancelled") {
		t.Fatalf("expected cancellation confirmation, got %q", out)
	}

	listOut2, _ := runTool(t, tool, map[string]any{"action": "list"})
	if !strings.Contains(strings.ToLower(listOut2), "no tasks") {
		t.Fatalf("expected empty list after cancel, got %q", listOut2)
	}
}

func TestScheduleCancelRequiresID(t *testing.T) {
	tool := Schedule{Sched: newTestSched(t)}
	if _, err := runTool(t, tool, map[string]any{"action": "cancel"}); err == nil {
		t.Fatal("expected error when id is missing")
	}
}

func TestScheduleUnknownAction(t *testing.T) {
	tool := Schedule{Sched: newTestSched(t)}
	if _, err := runTool(t, tool, map[string]any{"action": "delete"}); err == nil {
		t.Fatal("expected error for unknown action")
	}
}
