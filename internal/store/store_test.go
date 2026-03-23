package store_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/bmordue/tssk/internal/store"
	"github.com/bmordue/tssk/internal/task"
)

func newTempStore(t *testing.T) *store.Store {
	t.Helper()
	dir := t.TempDir()
	return store.New(dir)
}

func TestLoadAllEmpty(t *testing.T) {
	s := newTempStore(t)
	tasks, err := s.LoadAll()
	if err != nil {
		t.Fatalf("LoadAll on empty dir: %v", err)
	}
	if len(tasks) != 0 {
		t.Errorf("expected 0 tasks, got %d", len(tasks))
	}
}

func TestAddAndGet(t *testing.T) {
	s := newTempStore(t)

	tk, err := s.Add("My first task", "Some detail text", nil)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	if tk.ID == "" {
		t.Error("expected non-empty ID")
	}
	if tk.Title != "My first task" {
		t.Errorf("unexpected title: %q", tk.Title)
	}
	if tk.Status != task.StatusTodo {
		t.Errorf("unexpected status: %q", tk.Status)
	}
	if tk.DocHash == "" {
		t.Error("expected non-empty DocHash")
	}

	got, err := s.Get(tk.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != tk.ID {
		t.Errorf("ID mismatch: %s vs %s", got.ID, tk.ID)
	}
}

func TestAddCreatesDetailFile(t *testing.T) {
	dir := t.TempDir()
	s := store.New(dir)

	detail := "# My Task\n\nSome details here."
	tk, err := s.Add("Test detail file", detail, nil)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	docPath := filepath.Join(dir, "docs", tk.DocHash+".md")
	b, err := os.ReadFile(docPath)
	if err != nil {
		t.Fatalf("detail file not created: %v", err)
	}
	if string(b) != detail {
		t.Errorf("unexpected detail content: %q", string(b))
	}
}

func TestGetNotFound(t *testing.T) {
	s := newTempStore(t)
	_, err := s.Get("T-999")
	if !errors.Is(err, store.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestUpdateStatus(t *testing.T) {
	s := newTempStore(t)
	tk, _ := s.Add("Status test", "", nil)

	updated, err := s.UpdateStatus(tk.ID, task.StatusDone)
	if err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}
	if updated.Status != task.StatusDone {
		t.Errorf("expected done, got %q", updated.Status)
	}

	// Reload to verify persistence.
	reloaded, err := s.Get(tk.ID)
	if err != nil {
		t.Fatalf("Get after UpdateStatus: %v", err)
	}
	if reloaded.Status != task.StatusDone {
		t.Errorf("status not persisted: %q", reloaded.Status)
	}
}

func TestAddAndRemoveDep(t *testing.T) {
	s := newTempStore(t)
	t1, _ := s.Add("First task", "", nil)
	t2, _ := s.Add("Second task", "", nil)

	if err := s.AddDep(t2.ID, t1.ID); err != nil {
		t.Fatalf("AddDep: %v", err)
	}

	reloaded, _ := s.Get(t2.ID)
	if !reloaded.HasDependency(t1.ID) {
		t.Error("expected dependency to be present after AddDep")
	}

	if err := s.RemoveDep(t2.ID, t1.ID); err != nil {
		t.Fatalf("RemoveDep: %v", err)
	}

	reloaded, _ = s.Get(t2.ID)
	if reloaded.HasDependency(t1.ID) {
		t.Error("expected dependency to be absent after RemoveDep")
	}
}

func TestMultipleTasksLoadAll(t *testing.T) {
	s := newTempStore(t)
	for i := 0; i < 5; i++ {
		if _, err := s.Add("Task", "", nil); err != nil {
			t.Fatalf("Add task %d: %v", i, err)
		}
	}

	tasks, err := s.LoadAll()
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	if len(tasks) != 5 {
		t.Errorf("expected 5 tasks, got %d", len(tasks))
	}
}

func TestReadDetail(t *testing.T) {
	s := newTempStore(t)
	detail := "# Detail\n\nContent."
	tk, _ := s.Add("Task with detail", detail, nil)

	got, err := s.ReadDetail(tk)
	if err != nil {
		t.Fatalf("ReadDetail: %v", err)
	}
	if got != detail {
		t.Errorf("unexpected detail: %q", got)
	}
}
