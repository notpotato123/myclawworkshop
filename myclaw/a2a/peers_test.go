package a2a

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDiscover(t *testing.T) {
	card := AgentCard{Name: "TestBot", Description: "a test", Version: "1.0", Skills: []string{"foo"}}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/.well-known/agent-card.json" {
			http.NotFound(w, r)
			return
		}
		json.NewEncoder(w).Encode(card)
	}))
	defer srv.Close()

	got, err := Discover(srv.URL)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if got.Name != card.Name {
		t.Errorf("name: got %q want %q", got.Name, card.Name)
	}
	if got.URL == "" {
		t.Error("URL should be populated from base URL")
	}
}

func TestDiscoverBadURL(t *testing.T) {
	if _, err := Discover("http://127.0.0.1:1"); err == nil {
		t.Fatal("expected error for unreachable host")
	}
}

func TestRegistry(t *testing.T) {
	r := NewRegistry()

	r.Add(AgentCard{Name: "A", URL: "http://a", Skills: []string{"run_command", "memory"}})
	r.Add(AgentCard{Name: "B", URL: "http://b", Skills: []string{"schedule"}})

	all := r.All()
	if len(all) != 2 {
		t.Fatalf("expected 2 peers, got %d", len(all))
	}

	mem := r.FindBySkill("memory")
	if len(mem) != 1 || mem[0].Name != "A" {
		t.Errorf("FindBySkill(memory): %+v", mem)
	}

	sched := r.FindBySkill("SCHEDULE") // case-insensitive
	if len(sched) != 1 || sched[0].Name != "B" {
		t.Errorf("FindBySkill(SCHEDULE): %+v", sched)
	}

	none := r.FindBySkill("nonexistent")
	if len(none) != 0 {
		t.Errorf("expected no matches, got %v", none)
	}
}
