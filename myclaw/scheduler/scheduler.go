// Package scheduler provides a simple persistent task scheduler. Tasks are
// checked on a regular tick in a background goroutine; when due, a task's
// description is delivered to a callback. Recurring tasks reschedule
// themselves, and all tasks are persisted to a JSON file so they survive
// restarts.
package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// tickInterval is how often the scheduler checks for due tasks. One second
// keeps tasks responsive without meaningful CPU cost.
const tickInterval = time.Second

// Task is a unit of scheduled work.
type Task struct {
	ID          string        `json:"id"`
	Description string        `json:"description"`
	ExecuteAt   time.Time     `json:"execute_at"`
	Recurring   bool          `json:"recurring"`
	Interval    time.Duration `json:"interval"`
}

// Callback receives the description of a task when it becomes due.
type Callback func(description string)

// Scheduler manages a set of tasks and fires them via a callback.
type Scheduler struct {
	mu    sync.Mutex
	tasks map[string]Task
	path  string
	cb    Callback
}

// New creates a Scheduler that persists tasks to path and delivers due tasks
// to cb. Any tasks already saved at path are loaded.
func New(path string, cb Callback) (*Scheduler, error) {
	s := &Scheduler{
		tasks: make(map[string]Task),
		path:  path,
		cb:    cb,
	}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

// Add schedules a task and persists it.
func (s *Scheduler) Add(t Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tasks[t.ID] = t
	return s.save()
}

// Remove deletes a task by ID and persists the change.
func (s *Scheduler) Remove(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.tasks, id)
	return s.save()
}

// List returns a snapshot of all scheduled tasks, sorted by ExecuteAt.
func (s *Scheduler) List() []Task {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Task, 0, len(s.tasks))
	for _, t := range s.tasks {
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].ExecuteAt.Before(out[j].ExecuteAt)
	})
	return out
}

// Run starts the scheduler loop. It blocks until ctx is cancelled, at which
// point it returns cleanly. Run should be invoked in its own goroutine.
func (s *Scheduler) Run(ctx context.Context) {
	ticker := time.NewTicker(tickInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			s.fireDue(now)
		}
	}
}

// fireDue collects tasks that are due as of now, reschedules or removes them,
// persists the change, and then invokes the callback for each — all without
// holding the lock during callbacks.
func (s *Scheduler) fireDue(now time.Time) {
	s.mu.Lock()
	var due []Task
	dirty := false
	for id, t := range s.tasks {
		if t.ExecuteAt.After(now) {
			continue
		}
		due = append(due, t)
		if t.Recurring && t.Interval > 0 {
			t.ExecuteAt = t.ExecuteAt.Add(t.Interval)
			// If we fell far behind, skip ahead to the next future tick so we
			// don't fire repeatedly to catch up.
			for !t.ExecuteAt.After(now) {
				t.ExecuteAt = t.ExecuteAt.Add(t.Interval)
			}
			s.tasks[id] = t
		} else {
			delete(s.tasks, id)
		}
		dirty = true
	}
	if dirty {
		_ = s.save()
	}
	s.mu.Unlock()

	for _, t := range due {
		if s.cb != nil {
			s.cb(t.Description)
		}
	}
}

// save writes all tasks to disk. The caller must hold s.mu.
func (s *Scheduler) save() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("creating scheduler dir: %w", err)
	}
	list := make([]Task, 0, len(s.tasks))
	for _, t := range s.tasks {
		list = append(list, t)
	}
	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding tasks: %w", err)
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("writing tasks: %w", err)
	}
	return os.Rename(tmp, s.path)
}

// load reads tasks from disk. A missing file is not an error. The caller need
// not hold s.mu (called only from New, before the scheduler is shared).
func (s *Scheduler) load() error {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("reading tasks: %w", err)
	}
	var list []Task
	if err := json.Unmarshal(data, &list); err != nil {
		return fmt.Errorf("decoding tasks: %w", err)
	}
	for _, t := range list {
		s.tasks[t.ID] = t
	}
	return nil
}
