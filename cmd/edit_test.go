package cmd

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/bmordue/tssk/internal/store"
)

func setupEditTestStore(t *testing.T) (*store.Store, string) {
	t.Helper()
	dir := t.TempDir()
	s := store.New(dir)
	return s, dir
}

func TestEditCommand_UpdateTitle(t *testing.T) {
	s, dir := setupEditTestStore(t)

	task, err := s.Add("Original title", "Some detail", nil, nil)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	t.Setenv("TSSK_ROOT", dir)

	// Set title flag
	editTitle = "Updated title"
	defer func() { editTitle = "" }()

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmd := editCmd
	cmd.SetOut(w)
	cmd.SetErr(w)

	err = cmd.RunE(cmd, []string{task.ID})

	if closeErr := w.Close(); closeErr != nil {
		t.Fatalf("failed to close pipe: %v", closeErr)
	}
	os.Stdout = old

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var outBuf bytes.Buffer
	_, err = outBuf.ReadFrom(r)
	if err != nil {
		t.Fatalf("failed to read stdout: %v", err)
	}

	output := outBuf.String()
	if !strings.Contains(output, "Updated title") {
		t.Errorf("expected 'Updated title' in output, got: %s", output)
	}

	// Verify the task was actually updated by creating a new store instance
	s2 := store.New(dir)
	updated, err := s2.Get(task.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if updated.Title != "Updated title" {
		t.Errorf("expected title 'Updated title', got %q", updated.Title)
	}
}

func TestEditCommand_UpdateDetail(t *testing.T) {
	s, dir := setupEditTestStore(t)

	task, err := s.Add("Test task", "Original detail", nil, nil)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	t.Setenv("TSSK_ROOT", dir)

	// Set detail flag
	editDetail = "Updated detail text"
	defer func() { editDetail = "" }()

	cmd := editCmd
	cmd.SetOut(os.Stdout)
	cmd.SetErr(os.Stderr)

	err = cmd.RunE(cmd, []string{task.ID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the detail was actually updated with fresh store
	s2 := store.New(dir)
	updated, err := s2.Get(task.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	detail, err := s2.ReadDetail(updated)
	if err != nil {
		t.Fatalf("ReadDetail: %v", err)
	}
	if detail != "Updated detail text" {
		t.Errorf("expected detail 'Updated detail text', got %q", detail)
	}
}

func TestEditCommand_UpdateBothTitleAndDetail(t *testing.T) {
	s, dir := setupEditTestStore(t)

	task, err := s.Add("Original title", "Original detail", nil, nil)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	t.Setenv("TSSK_ROOT", dir)

	// Set both flags
	editTitle = "New title"
	editDetail = "New detail"
	defer func() {
		editTitle = ""
		editDetail = ""
	}()

	cmd := editCmd
	cmd.SetOut(os.Stdout)
	cmd.SetErr(os.Stderr)

	err = cmd.RunE(cmd, []string{task.ID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify both were updated with fresh store
	s2 := store.New(dir)
	updated, err := s2.Get(task.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if updated.Title != "New title" {
		t.Errorf("expected title 'New title', got %q", updated.Title)
	}
	detail, err := s2.ReadDetail(updated)
	if err != nil {
		t.Fatalf("ReadDetail: %v", err)
	}
	if detail != "New detail" {
		t.Errorf("expected detail 'New detail', got %q", detail)
	}
}

func TestEditCommand_NoFlagsProvided(t *testing.T) {
	s, dir := setupEditTestStore(t)

	_, err := s.Add("Test task", "Detail", nil, nil)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	t.Setenv("TSSK_ROOT", dir)

	// Don't set any flags
	cmd := editCmd
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	err = cmd.RunE(cmd, []string{"1"})
	if err == nil {
		t.Fatal("expected error when no flags provided, got none")
	}

	if !strings.Contains(err.Error(), "at least one of --title or --detail") {
		t.Errorf("expected error about providing --title or --detail, got: %v", err)
	}
}

func TestEditCommand_TaskNotFound(t *testing.T) {
	_, dir := setupEditTestStore(t)

	t.Setenv("TSSK_ROOT", dir)

	editTitle = "Some title"
	defer func() { editTitle = "" }()

	cmd := editCmd
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	err := cmd.RunE(cmd, []string{"999"})
	if err == nil {
		t.Fatal("expected error for non-existent task, got none")
	}

	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

func TestEditCommand_PrefixMatch(t *testing.T) {
	s, dir := setupEditTestStore(t)

	_, err := s.Add("Task one", "Detail 1", nil, nil)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	_, err = s.Add("Task two", "Detail 2", nil, nil)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	t.Setenv("TSSK_ROOT", dir)

	editTitle = "Updated task one"
	defer func() { editTitle = "" }()

	cmd := editCmd
	cmd.SetOut(os.Stdout)
	cmd.SetErr(os.Stderr)

	// Use prefix "1" to match task 1
	err = cmd.RunE(cmd, []string{"1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the correct task was updated with fresh store
	s2 := store.New(dir)
	updated, err := s2.Get("1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if updated.Title != "Updated task one" {
		t.Errorf("expected title 'Updated task one', got %q", updated.Title)
	}
}

func TestEditCommand_UpdateTitleWithEmptyDetail(t *testing.T) {
	s, dir := setupEditTestStore(t)

	task, err := s.Add("Original", "", nil, nil)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	t.Setenv("TSSK_ROOT", dir)

	editTitle = "New title"
	defer func() { editTitle = "" }()

	cmd := editCmd
	cmd.SetOut(os.Stdout)
	cmd.SetErr(os.Stderr)

	err = cmd.RunE(cmd, []string{task.ID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify with fresh store
	s2 := store.New(dir)
	updated, err := s2.Get(task.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if updated.Title != "New title" {
		t.Errorf("expected title 'New title', got %q", updated.Title)
	}
}

func TestEditCommand_UpdateDetailPreservesTitle(t *testing.T) {
	s, dir := setupEditTestStore(t)

	task, err := s.Add("Keep this title", "Old detail", nil, nil)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	t.Setenv("TSSK_ROOT", dir)

	editDetail = "New detail only"
	defer func() { editDetail = "" }()

	cmd := editCmd
	cmd.SetOut(os.Stdout)
	cmd.SetErr(os.Stderr)

	err = cmd.RunE(cmd, []string{task.ID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify title unchanged with fresh store
	s2 := store.New(dir)
	updated, err := s2.Get(task.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if updated.Title != "Keep this title" {
		t.Errorf("expected title 'Keep this title', got %q", updated.Title)
	}
	detail, err := s2.ReadDetail(updated)
	if err != nil {
		t.Fatalf("ReadDetail: %v", err)
	}
	if detail != "New detail only" {
		t.Errorf("expected detail 'New detail only', got %q", detail)
	}
}

func TestEditCommand_FlagReset(t *testing.T) {
	// Ensure the flags are reset after tests
	origTitle := editTitle
	origDetail := editDetail
	defer func() {
		editTitle = origTitle
		editDetail = origDetail
	}()

	// Create a temp dir for this test
	dir, err := os.MkdirTemp("", "tssk-edit-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() {
		if rmErr := os.RemoveAll(dir); rmErr != nil {
			t.Logf("warning: failed to remove temp dir: %v", rmErr)
		}
	}()

	t.Setenv("TSSK_ROOT", dir)

	cmd := editCmd
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	// Should fail without flags
	err = cmd.RunE(cmd, []string{"1"})
	if err == nil {
		t.Fatal("expected error without flags, got none")
	}
}
