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

	tk, err := s.Add("My first task", "Some detail text", nil, nil)
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
	tk, err := s.Add("Test detail file", detail, nil, nil)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	// DocHash is the full 64-char hash; the file is named with the default display length (9).
	docPath := filepath.Join(dir, ".tsks", "docs", tk.DocHash[:9]+".md")
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
	_, err := s.Get("999")
	if !errors.Is(err, store.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}
func TestPrefixMatching(t *testing.T) {
	s := newTempStore(t)
	// Add 10 tasks to produce IDs "1" through "10".
	for i := 0; i < 10; i++ {
		if _, err := s.Add("Task", "", nil, nil); err != nil {
			t.Fatalf("Add task %d: %v", i, err)
		}
	}

	// Full two-digit ID lookup.
	got, err := s.Get("10")
	if err != nil {
		t.Fatalf("Get(10): %v", err)
	}
	if got.ID != "10" {
		t.Errorf("expected ID 10, got %s", got.ID)
	}

	// Exact single-digit match wins: "1" resolves to task "1", not a prefix of "10".
	got, err = s.Get("1")
	if err != nil {
		t.Fatalf("Get(1): %v", err)
	}
	if got.ID != "1" {
		t.Errorf("expected exact match ID 1, got %s", got.ID)
	}

	// Unmatched prefix returns ErrNotFound.
	_, err = s.Get("99")
	if !errors.Is(err, store.ErrNotFound) {
		t.Errorf("expected ErrNotFound for unmatched prefix, got %v", err)
	}

	// Prefix resolution also works for UpdateStatus.
	updated, err := s.UpdateStatus("10", task.StatusDone)
	if err != nil {
		t.Fatalf("UpdateStatus by exact ID: %v", err)
	}
	if updated.ID != "10" || updated.Status != task.StatusDone {
		t.Errorf("unexpected result: id=%s status=%s", updated.ID, updated.Status)
	}
}
func TestAmbiguousPrefix(t *testing.T) {
	dir := t.TempDir()
	// Seed the tasks file directly with two tasks whose IDs share a common
	// prefix but neither of which is an exact match for that prefix.
	tasksDir := filepath.Join(dir, ".tsks")
	if err := os.MkdirAll(tasksDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	tasksData := `{"id":"10","title":"Task Ten","status":"todo","created_at":"2024-01-01T00:00:00Z","doc_hash":""}
{"id":"11","title":"Task Eleven","status":"todo","created_at":"2024-01-01T00:00:00Z","doc_hash":""}
`
	if err := os.WriteFile(filepath.Join(tasksDir, "tasks.jsonl"), []byte(tasksData), 0o644); err != nil {
		t.Fatalf("write tasks: %v", err)
	}
	s := store.New(dir)

	// "1" is not an exact ID but is a prefix of both "10" and "11".
	_, err := s.Get("1")
	if !errors.Is(err, store.ErrAmbiguous) {
		t.Errorf("expected ErrAmbiguous for ambiguous prefix, got %v", err)
	}
}

func TestUpdateStatus(t *testing.T) {
	s := newTempStore(t)
	tk, _ := s.Add("Status test", "", nil, nil)

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
	t1, err := s.Add("First task", "", nil, nil)
	if err != nil {
		t.Fatalf("Add t1: %v", err)
	}
	t2, err := s.Add("Second task", "", nil, nil)
	if err != nil {
		t.Fatalf("Add t2: %v", err)
	}

	// Both tasks must be independently retrievable after writing two JSONL lines.
	got1, err := s.Get(t1.ID)
	if err != nil {
		t.Fatalf("Get t1 after adding t2: %v", err)
	}
	if got1.ID != t1.ID || got1.Title != t1.Title {
		t.Errorf("t1 data corrupted: got id=%q title=%q", got1.ID, got1.Title)
	}

	err = s.AddDep(t2.ID, t1.ID)
	if err != nil {
		t.Fatalf("AddDep: %v", err)
	}

	reloaded, _ := s.Get(t2.ID)
	if !reloaded.HasDependency(t1.ID) {
		t.Error("expected dependency to be present after AddDep")
	}

	// t1 must remain intact after modifying t2's deps.
	got1, err = s.Get(t1.ID)
	if err != nil {
		t.Fatalf("Get t1 after AddDep: %v", err)
	}
	if got1.ID != t1.ID {
		t.Errorf("t1 corrupted after AddDep: got id=%q", got1.ID)
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
		if _, err := s.Add("Task", "", nil, nil); err != nil {
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
	tk, _ := s.Add("Task with detail", detail, nil, nil)

	got, err := s.ReadDetail(tk)
	if err != nil {
		t.Fatalf("ReadDetail: %v", err)
	}
	if got != detail {
		t.Errorf("unexpected detail: %q", got)
	}
}

func TestAddTagsDeduplication(t *testing.T) {
	s := newTempStore(t)
	tk, addErr := s.Add("Tag test task", "", nil, []string{"alpha"})
	if addErr != nil {
		t.Fatalf("Add: %v", addErr)
	}

	// Adding a duplicate tag should not produce a second entry.
	if tagsErr := s.AddTags(tk.ID, []string{"alpha", "beta"}); tagsErr != nil {
		t.Fatalf("AddTags: %v", tagsErr)
	}

	reloaded, getErr := s.Get(tk.ID)
	if getErr != nil {
		t.Fatalf("Get after AddTags: %v", getErr)
	}
	if len(reloaded.Tags) != 2 {
		t.Errorf("expected 2 tags after dedup, got %d: %v", len(reloaded.Tags), reloaded.Tags)
	}
	if !reloaded.HasTag("alpha") || !reloaded.HasTag("beta") {
		t.Errorf("expected tags alpha and beta, got %v", reloaded.Tags)
	}
}

func TestRemoveTagsNonExistent(t *testing.T) {
	s := newTempStore(t)
	tk, addErr := s.Add("Tag remove test", "", nil, []string{"alpha", "beta"})
	if addErr != nil {
		t.Fatalf("Add: %v", addErr)
	}

	// Removing a tag that does not exist should succeed without error.
	if removeErr := s.RemoveTags(tk.ID, []string{"beta", "gamma"}); removeErr != nil {
		t.Fatalf("RemoveTags: %v", removeErr)
	}

	reloaded, err := s.Get(tk.ID)
	if err != nil {
		t.Fatalf("Get after RemoveTags: %v", err)
	}
	if len(reloaded.Tags) != 1 || !reloaded.HasTag("alpha") {
		t.Errorf("expected only tag alpha, got %v", reloaded.Tags)
	}
}

func TestSetTagsPersistence(t *testing.T) {
	s := newTempStore(t)
	tk, addErr := s.Add("SetTags persistence test", "", nil, []string{"old"})
	if addErr != nil {
		t.Fatalf("Add: %v", addErr)
	}

	if setErr := s.SetTags(tk.ID, []string{"new1", "new2"}); setErr != nil {
		t.Fatalf("SetTags: %v", setErr)
	}

	// Reload from disk to confirm persistence.
	reloaded, err := s.Get(tk.ID)
	if err != nil {
		t.Fatalf("Get after SetTags: %v", err)
	}
	if len(reloaded.Tags) != 2 {
		t.Errorf("expected 2 tags, got %d: %v", len(reloaded.Tags), reloaded.Tags)
	}
	if !reloaded.HasTag("new1") || !reloaded.HasTag("new2") {
		t.Errorf("expected tags new1 and new2, got %v", reloaded.Tags)
	}
	if reloaded.HasTag("old") {
		t.Error("expected old tag to be replaced by SetTags")
	}
}

func TestUpdateTitle(t *testing.T) {
	s := newTempStore(t)
	tk, err := s.Add("Original title", "Detail text", nil, nil)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	oldHash := tk.DocHash

	updated, err := s.UpdateTitle(tk.ID, "New title")
	if err != nil {
		t.Fatalf("UpdateTitle: %v", err)
	}

	if updated.Title != "New title" {
		t.Errorf("expected title 'New title', got %q", updated.Title)
	}
	if updated.DocHash == oldHash {
		t.Error("expected DocHash to change after title update")
	}

	// Verify persistence
	reloaded, err := s.Get(tk.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if reloaded.Title != "New title" {
		t.Errorf("expected persisted title 'New title', got %q", reloaded.Title)
	}
}

func TestUpdateTitleWithDetail(t *testing.T) {
	s := newTempStore(t)
	tk, err := s.Add("Title one", "Some detail", nil, nil)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	oldHash := tk.DocHash

	_, err = s.UpdateTitle(tk.ID, "Title two")
	if err != nil {
		t.Fatalf("UpdateTitle: %v", err)
	}

	// Verify detail file migrated
	updated, err := s.Get(tk.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	detail, err := s.ReadDetail(updated)
	if err != nil {
		t.Fatalf("ReadDetail: %v", err)
	}
	if detail != "Some detail" {
		t.Errorf("expected detail 'Some detail', got %q", detail)
	}

	// Verify hash changed
	if updated.DocHash == oldHash {
		t.Error("expected DocHash to change after title update")
	}
}

func TestUpdateDetail(t *testing.T) {
	s := newTempStore(t)
	tk, err := s.Add("Test task", "Original detail", nil, nil)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	updated, err := s.UpdateDetail(tk.ID, "Updated detail text")
	if err != nil {
		t.Fatalf("UpdateDetail: %v", err)
	}

	if updated.ID != tk.ID {
		t.Errorf("expected ID %q, got %q", tk.ID, updated.ID)
	}

	// Verify persistence
	detail, err := s.ReadDetail(updated)
	if err != nil {
		t.Fatalf("ReadDetail: %v", err)
	}
	if detail != "Updated detail text" {
		t.Errorf("expected detail 'Updated detail text', got %q", detail)
	}
}

func TestUpdateTitleNotFound(t *testing.T) {
	s := newTempStore(t)
	_, err := s.UpdateTitle("999", "New title")
	if err == nil {
		t.Fatal("expected error for non-existent task, got none")
	}
	if !errors.Is(err, store.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

func TestUpdateDetailNotFound(t *testing.T) {
	s := newTempStore(t)
	_, err := s.UpdateDetail("999", "New detail")
	if err == nil {
		t.Fatal("expected error for non-existent task, got none")
	}
	if !errors.Is(err, store.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}
