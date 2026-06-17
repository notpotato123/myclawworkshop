package memory

import (
	"os"
	"strings"
	"testing"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	return s
}

func TestSaveLoadRoundTrip(t *testing.T) {
	s := newTestStore(t)
	if err := s.Save("user_name", "Daniel"); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := s.Load("user_name")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got != "Daniel" {
		t.Fatalf("got %q, want %q", got, "Daniel")
	}
}

func TestLoadMissing(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.Load("nope"); err == nil {
		t.Fatal("expected error loading missing memory")
	}
}

func TestKeySanitization(t *testing.T) {
	s := newTestStore(t)
	// Keys with unsafe characters must still round-trip and not escape the dir.
	if err := s.Save("../etc/passwd", "data"); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := s.Load("../etc/passwd")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got != "data" {
		t.Fatalf("got %q, want %q", got, "data")
	}
}

func TestUpdatePreservesCreated(t *testing.T) {
	s := newTestStore(t)
	if err := s.Save("k", "first"); err != nil {
		t.Fatal(err)
	}
	created1 := extractFrontmatter(readRaw(t, s, "k"), "created")

	if err := s.Save("k", "second"); err != nil {
		t.Fatal(err)
	}
	raw := readRaw(t, s, "k")
	created2 := extractFrontmatter(raw, "created")
	updated2 := extractFrontmatter(raw, "updated")

	if created1 != created2 {
		t.Fatalf("created timestamp changed on update: %q -> %q", created1, created2)
	}
	if updated2 == "" {
		t.Fatal("updated timestamp missing")
	}
	got, _ := s.Load("k")
	if got != "second" {
		t.Fatalf("content not updated: got %q", got)
	}
}

func TestList(t *testing.T) {
	s := newTestStore(t)
	s.Save("a", "1")
	s.Save("b", "2")
	keys, err := s.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) != 2 {
		t.Fatalf("expected 2 keys, got %d: %v", len(keys), keys)
	}
	if !contains(keys, "a") || !contains(keys, "b") {
		t.Fatalf("missing expected keys: %v", keys)
	}
}

func TestSearchCaseInsensitive(t *testing.T) {
	s := newTestStore(t)
	s.Save("greeting", "The user said HELLO world")
	s.Save("other", "unrelated content")

	results, err := s.Search("hello")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d: %v", len(results), results)
	}
	if !strings.Contains(results[0], "greeting") {
		t.Fatalf("result missing key: %q", results[0])
	}
}

func TestSearchMatchesKey(t *testing.T) {
	s := newTestStore(t)
	s.Save("project_goal", "ship it")
	results, err := s.Search("goal")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected to match on key, got %d results", len(results))
	}
}

func TestSearchNoMatch(t *testing.T) {
	s := newTestStore(t)
	s.Save("a", "something")
	results, err := s.Search("missing")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 0 {
		t.Fatalf("expected no results, got %v", results)
	}
}

func TestDumpRespectsBudget(t *testing.T) {
	s := newTestStore(t)
	s.Save("a", strings.Repeat("x", 100))
	s.Save("b", strings.Repeat("y", 100))

	full := s.Dump(10000)
	if !strings.Contains(full, "a:") || !strings.Contains(full, "b:") {
		t.Fatalf("expected both memories in full dump: %q", full)
	}

	// A tiny budget must truncate.
	small := s.Dump(50)
	if len(small) > 50 {
		t.Fatalf("dump exceeded budget: %d chars", len(small))
	}
}

func TestDumpEmpty(t *testing.T) {
	s := newTestStore(t)
	if got := s.Dump(1000); got != "" {
		t.Fatalf("expected empty dump, got %q", got)
	}
}

func readRaw(t *testing.T, s *Store, key string) string {
	t.Helper()
	data, err := os.ReadFile(s.path(key))
	if err != nil {
		t.Fatalf("reading raw: %v", err)
	}
	return string(data)
}

func contains(ss []string, target string) bool {
	for _, s := range ss {
		if s == target {
			return true
		}
	}
	return false
}
