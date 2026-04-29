package store_test

import (
	"os"
	"testing"

	"pgregory.net/rapid"

	"github.com/bmordue/tssk/internal/store"
	"github.com/bmordue/tssk/internal/task"
)

// propTempStore creates a store in a fresh temporary directory.
// The directory is cleaned up when the outer test ends.
func propTempStore(t *rapid.T) *store.Store {
	dir, err := os.MkdirTemp("", "tssk-prop-*")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	return store.New(dir)
}

func TestProperty_AddGet_RoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		s := propTempStore(t)
		title := rapid.StringN(1, 100, 100).Draw(t, "title")
		added, err := s.Add(title, "", nil, nil, task.PriorityNone)
		if err != nil {
			t.Fatalf("Add: %v", err)
		}
		got, err := s.Get(added.ID)
		if err != nil {
			t.Fatalf("Get(%s): %v", added.ID, err)
		}
		if got.Title != title {
			t.Fatalf("title mismatch: got %q, want %q", got.Title, title)
		}
		if got.Status != task.StatusTodo {
			t.Fatalf("new task status: got %q, want %q", got.Status, task.StatusTodo)
		}
	})
}

func TestProperty_AddMultiple_UniqueIDs(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		s := propTempStore(t)
		n := rapid.IntRange(1, 20).Draw(t, "count")
		ids := make(map[string]bool, n)
		for i := 0; i < n; i++ {
			tk, err := s.Add("task", "", nil, nil, task.PriorityNone)
			if err != nil {
				t.Fatalf("Add: %v", err)
			}
			if ids[tk.ID] {
				t.Fatalf("duplicate ID: %s", tk.ID)
			}
			ids[tk.ID] = true
		}
	})
}

func TestProperty_UpdateStatus_Persists(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		s := propTempStore(t)
		tk, err := s.Add("task", "", nil, nil, task.PriorityNone)
		if err != nil {
			t.Fatalf("Add: %v", err)
		}
		statuses := []task.Status{task.StatusTodo, task.StatusInProgress, task.StatusDone, task.StatusBlocked}
		status := rapid.SampledFrom(statuses).Draw(t, "status")
		updated, err := s.UpdateStatus(tk.ID, status)
		if err != nil {
			t.Fatalf("UpdateStatus: %v", err)
		}
		if updated.Status != status {
			t.Fatalf("status: got %q, want %q", updated.Status, status)
		}
		got, err := s.Get(tk.ID)
		if err != nil {
			t.Fatalf("Get: %v", err)
		}
		if got.Status != status {
			t.Fatalf("persisted status: got %q, want %q", got.Status, status)
		}
	})
}

func TestProperty_AddRemoveTags_SetBehavior(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		s := propTempStore(t)
		tk, addErr := s.Add("task", "", nil, nil, task.PriorityNone)
		if addErr != nil {
			t.Fatalf("Add: %v", addErr)
		}
		tags := rapid.SliceOfN(rapid.StringN(1, 20, 20), 1, 10).Draw(t, "tags")
		if tagErr := s.AddTags(tk.ID, tags); tagErr != nil {
			t.Fatalf("AddTags: %v", tagErr)
		}
		got, getErr := s.Get(tk.ID)
		if getErr != nil {
			t.Fatal(getErr)
		}
		// All added tags must be present with no duplicates.
		for _, tag := range tags {
			if !got.HasTag(tag) {
				t.Fatalf("tag %q missing after AddTags", tag)
			}
		}
		seen := make(map[string]bool, len(got.Tags))
		for _, tg := range got.Tags {
			if seen[tg] {
				t.Fatalf("duplicate tag: %q", tg)
			}
			seen[tg] = true
		}
		// Remove a random subset of tags and verify they're gone.
		removeCount := rapid.IntRange(0, len(tags)).Draw(t, "removeCount")
		toRemove := tags[:removeCount]
		if rmErr := s.RemoveTags(tk.ID, toRemove); rmErr != nil {
			t.Fatalf("RemoveTags: %v", rmErr)
		}
		got, getErr = s.Get(tk.ID)
		if getErr != nil {
			t.Fatal(getErr)
		}
		for _, tag := range toRemove {
			if got.HasTag(tag) {
				t.Fatalf("tag %q still present after RemoveTags", tag)
			}
		}
	})
}

func TestProperty_Priority_Validation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		s := propTempStore(t)
		tk, err := s.Add("task", "", nil, nil, task.PriorityNone)
		if err != nil {
			t.Fatalf("Add: %v", err)
		}
		p := task.Priority(rapid.String().Draw(t, "priority"))
		_, err = s.UpdatePriority(tk.ID, p)
		if p.IsValid() {
			if err != nil {
				t.Fatalf("valid priority %q rejected: %v", p, err)
			}
		} else {
			if err == nil {
				t.Fatalf("invalid priority %q accepted", p)
			}
		}
	})
}
