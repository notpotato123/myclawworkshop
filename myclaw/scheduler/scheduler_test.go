package scheduler

import (
	"context"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestOneShotFires(t *testing.T) {
	dir := t.TempDir()
	var fired atomic.Int32
	s, err := New(filepath.Join(dir, "tasks.json"), func(string) { fired.Add(1) })
	if err != nil {
		t.Fatal(err)
	}
	s.Add(Task{ID: "a", Description: "hi", ExecuteAt: time.Now().Add(500 * time.Millisecond)})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go s.Run(ctx)

	time.Sleep(2 * time.Second)
	if fired.Load() != 1 {
		t.Fatalf("expected 1 fire, got %d", fired.Load())
	}
	if len(s.List()) != 0 {
		t.Fatalf("one-shot task should be removed after firing")
	}
}

func TestRecurringRefires(t *testing.T) {
	dir := t.TempDir()
	var fired atomic.Int32
	s, err := New(filepath.Join(dir, "tasks.json"), func(string) { fired.Add(1) })
	if err != nil {
		t.Fatal(err)
	}
	s.Add(Task{
		ID:        "r",
		ExecuteAt: time.Now().Add(500 * time.Millisecond),
		Recurring: true,
		Interval:  time.Second,
	})

	ctx, cancel := context.WithCancel(context.Background())
	go s.Run(ctx)
	time.Sleep(3 * time.Second)
	cancel()

	if n := fired.Load(); n < 2 {
		t.Fatalf("expected recurring task to fire at least twice, got %d", n)
	}
	if len(s.List()) != 1 {
		t.Fatalf("recurring task should remain scheduled")
	}
}

func TestPersistAcrossRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tasks.json")
	s1, _ := New(path, nil)
	s1.Add(Task{ID: "p", Description: "persist", ExecuteAt: time.Now().Add(time.Hour)})

	s2, err := New(path, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(s2.List()) != 1 {
		t.Fatalf("expected task to persist across restart, got %d", len(s2.List()))
	}
}

func TestConcurrentAccess(t *testing.T) {
	s, _ := New(filepath.Join(t.TempDir(), "tasks.json"), func(string) {})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go s.Run(ctx)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			s.Add(Task{ID: string(rune('a' + i%26)), ExecuteAt: time.Now()})
			s.List()
		}(i)
	}
	wg.Wait()
}
