package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"myclaw/memory"
)

func newTestStore(t *testing.T) *memory.Store {
	t.Helper()
	s, err := memory.NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	return s
}

func runTool(t *testing.T, tool Tool, params any) (string, error) {
	t.Helper()
	raw, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("marshal params: %v", err)
	}
	return tool.Execute(context.Background(), raw)
}

func TestRememberSaves(t *testing.T) {
	store := newTestStore(t)
	r := Remember{Store: store}

	out, err := runTool(t, r, map[string]string{"key": "fact", "content": "the sky is blue"})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out, "fact") {
		t.Fatalf("confirmation missing key: %q", out)
	}

	got, err := store.Load("fact")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got != "the sky is blue" {
		t.Fatalf("got %q, want %q", got, "the sky is blue")
	}
}

func TestRememberRequiresFields(t *testing.T) {
	r := Remember{Store: newTestStore(t)}

	if _, err := runTool(t, r, map[string]string{"content": "x"}); err == nil {
		t.Fatal("expected error when key is missing")
	}
	if _, err := runTool(t, r, map[string]string{"key": "k"}); err == nil {
		t.Fatal("expected error when content is missing")
	}
}

func TestRecallWithQuery(t *testing.T) {
	store := newTestStore(t)
	store.Save("greeting", "hello world")
	store.Save("other", "goodbye")

	rc := Recall{Store: store}
	out, err := runTool(t, rc, map[string]string{"query": "hello"})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out, "greeting") || !strings.Contains(out, "hello world") {
		t.Fatalf("expected matching memory, got %q", out)
	}
	if strings.Contains(out, "goodbye") {
		t.Fatalf("unexpected non-matching memory in result: %q", out)
	}
}

func TestRecallNoMatch(t *testing.T) {
	store := newTestStore(t)
	store.Save("a", "something")

	rc := Recall{Store: store}
	out, err := runTool(t, rc, map[string]string{"query": "absent"})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(strings.ToLower(out), "no memories") {
		t.Fatalf("expected no-match message, got %q", out)
	}
}

func TestRecallEmptyListsKeys(t *testing.T) {
	store := newTestStore(t)
	store.Save("alpha", "1")
	store.Save("beta", "2")

	rc := Recall{Store: store}
	out, err := runTool(t, rc, map[string]string{})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out, "alpha") || !strings.Contains(out, "beta") {
		t.Fatalf("expected key listing, got %q", out)
	}
}

func TestRecallEmptyNoMemories(t *testing.T) {
	rc := Recall{Store: newTestStore(t)}
	out, err := runTool(t, rc, map[string]string{})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(strings.ToLower(out), "no memories") {
		t.Fatalf("expected empty-store message, got %q", out)
	}
}
