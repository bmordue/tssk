package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/bmordue/tssk/internal/store"
	"github.com/bmordue/tssk/internal/task"
)

func setupListTestStore(t *testing.T) (*store.Store, string) {
	t.Helper()
	dir := t.TempDir()
	s := store.New(dir)
	return s, dir
}

func TestListCommand_TitleFilter_BasicMatch(t *testing.T) {
	s, dir := setupListTestStore(t)

	// Add tasks with different titles
	_, err := s.Add("Implement authentication", "detail 1", nil, nil, task.PriorityNone)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	_, err = s.Add("Fix database bug", "detail 2", nil, nil, task.PriorityNone)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	_, err = s.Add("Update documentation", "detail 3", nil, nil, task.PriorityNone)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	t.Setenv("TSSK_ROOT", dir)

	// Set title filter
	listTitle = "auth"
	defer func() { listTitle = "" }()

	// Capture stdout since printJSON writes to it
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmd := listCmd
	cmd.SetOut(w)
	cmd.SetErr(w)

	err = cmd.RunE(cmd, nil)

	// Restore stdout
	if closeErr := w.Close(); closeErr != nil {
		t.Fatalf("failed to close pipe: %v", closeErr)
	}
	os.Stdout = old

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Read the captured output
	var outBuf bytes.Buffer
	_, err = outBuf.ReadFrom(r)
	if err != nil {
		t.Fatalf("failed to read stdout: %v", err)
	}

	output := outBuf.String()
	// Should contain authentication task
	if !strings.Contains(output, "Implement authentication") {
		t.Errorf("expected 'Implement authentication' in output, got: %s", output)
	}
	// Should not contain other tasks
	if strings.Contains(output, "Fix database bug") {
		t.Errorf("did not expect 'Fix database bug' in output, got: %s", output)
	}
	if strings.Contains(output, "Update documentation") {
		t.Errorf("did not expect 'Update documentation' in output, got: %s", output)
	}
}

func TestListCommand_TitleFilter_CaseInsensitive(t *testing.T) {
	s, dir := setupListTestStore(t)

	_, err := s.Add("Task with DATABASE connection", "detail 1", nil, nil, task.PriorityNone)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	_, err = s.Add("Another task", "detail 2", nil, nil, task.PriorityNone)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	t.Setenv("TSSK_ROOT", dir)

	// Test case-insensitive matching
	listTitle = "database"
	defer func() { listTitle = "" }()

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmd := listCmd
	cmd.SetOut(w)
	cmd.SetErr(w)

	err = cmd.RunE(cmd, nil)

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
	if !strings.Contains(output, "Task with DATABASE connection") {
		t.Errorf("expected 'Task with DATABASE connection' in output, got: %s", output)
	}
	if strings.Contains(output, "Another task") {
		t.Errorf("did not expect 'Another task' in output, got: %s", output)
	}
}

func TestListCommand_TitleFilter_NoMatch(t *testing.T) {
	s, dir := setupListTestStore(t)

	_, err := s.Add("Task one", "detail 1", nil, nil, task.PriorityNone)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	_, err = s.Add("Task two", "detail 2", nil, nil, task.PriorityNone)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	t.Setenv("TSSK_ROOT", dir)

	// Search for non-existent title
	listTitle = "nonexistent"
	defer func() { listTitle = "" }()

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmd := listCmd
	cmd.SetOut(w)
	cmd.SetErr(w)

	err = cmd.RunE(cmd, nil)

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
	// Should only contain header, no tasks
	if strings.Contains(output, "Task one") || strings.Contains(output, "Task two") {
		t.Errorf("did not expect any tasks in output for non-matching title, got: %s", output)
	}
}

func TestListCommand_TitleFilter_EmptyFilter(t *testing.T) {
	s, dir := setupListTestStore(t)

	_, err := s.Add("Task alpha", "detail 1", nil, nil, task.PriorityNone)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	_, err = s.Add("Task beta", "detail 2", nil, nil, task.PriorityNone)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	t.Setenv("TSSK_ROOT", dir)

	// Empty filter should show all tasks
	listTitle = ""

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmd := listCmd
	cmd.SetOut(w)
	cmd.SetErr(w)

	err = cmd.RunE(cmd, nil)

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
	if !strings.Contains(output, "Task alpha") {
		t.Errorf("expected 'Task alpha' in output, got: %s", output)
	}
	if !strings.Contains(output, "Task beta") {
		t.Errorf("expected 'Task beta' in output, got: %s", output)
	}
}

func TestListCommand_TitleFilter_JSONOutput(t *testing.T) {
	s, dir := setupListTestStore(t)

	_, err := s.Add("Build API endpoint", "detail 1", nil, nil, task.PriorityNone)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	_, err = s.Add("Write tests", "detail 2", nil, nil, task.PriorityNone)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	t.Setenv("TSSK_ROOT", dir)

	// Set JSON and title filter
	listJSON = true
	listTitle = "API"
	defer func() {
		listJSON = false
		listTitle = ""
	}()

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmd := listCmd
	cmd.SetOut(w)
	cmd.SetErr(w)

	err = cmd.RunE(cmd, nil)

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

	// Should be valid JSON
	var tasks []task.Task
	if err := json.Unmarshal(outBuf.Bytes(), &tasks); err != nil {
		t.Fatalf("invalid JSON output: %v, got: %q", err, outBuf.String())
	}

	// Should only contain API task
	if len(tasks) != 1 {
		t.Errorf("expected 1 task in JSON, got %d", len(tasks))
	}
	if len(tasks) > 0 && tasks[0].Title != "Build API endpoint" {
		t.Errorf("expected 'Build API endpoint', got %q", tasks[0].Title)
	}
}

func TestListCommand_TitleFilter_CombinedWithStatus(t *testing.T) {
	s, dir := setupListTestStore(t)

	task1, err := s.Add("Todo task", "detail 1", nil, nil, task.PriorityNone)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	_, err = s.UpdateStatus(task1.ID, task.StatusDone)
	if err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}

	_, err = s.Add("Done item", "detail 2", nil, nil, task.PriorityNone)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	t.Setenv("TSSK_ROOT", dir)

	// Filter by both title and status
	listTitle = "task"
	listStatus = "done"
	defer func() {
		listTitle = ""
		listStatus = ""
	}()

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmd := listCmd
	cmd.SetOut(w)
	cmd.SetErr(w)

	err = cmd.RunE(cmd, nil)

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
	// Should contain "Todo task" (matches title and status done)
	if !strings.Contains(output, "Todo task") {
		t.Errorf("expected 'Todo task' in output, got: %s", output)
	}
	// Should not contain "Done item" (doesn't match title)
	if strings.Contains(output, "Done item") {
		t.Errorf("did not expect 'Done item' in output, got: %s", output)
	}
}

func TestListCommand_TitleFilter_CombinedWithTag(t *testing.T) {
	s, dir := setupListTestStore(t)

	_, err := s.Add("Backend task", "detail 1", nil, []string{"api"}, task.PriorityNone)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	_, err = s.Add("Frontend task", "detail 2", nil, []string{"ui"}, task.PriorityNone)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	_, err = s.Add("Backend item", "detail 3", nil, []string{"ui"}, task.PriorityNone)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	t.Setenv("TSSK_ROOT", dir)

	// Filter by both title and tag
	listTitle = "Backend"
	listTag = "api"
	defer func() {
		listTitle = ""
		listTag = ""
	}()

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmd := listCmd
	cmd.SetOut(w)
	cmd.SetErr(w)

	err = cmd.RunE(cmd, nil)

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
	// Should only contain "Backend task" (matches title and tag)
	if !strings.Contains(output, "Backend task") {
		t.Errorf("expected 'Backend task' in output, got: %s", output)
	}
	if strings.Contains(output, "Frontend task") {
		t.Errorf("did not expect 'Frontend task' in output, got: %s", output)
	}
	if strings.Contains(output, "Backend item") {
		t.Errorf("did not expect 'Backend item' in output, got: %s", output)
	}
}

func TestListCommand_TitleFilter_SubstringMatch(t *testing.T) {
	s, dir := setupListTestStore(t)

	_, err := s.Add("Initialize project structure", "detail 1", nil, nil, task.PriorityNone)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	_, err = s.Add("Final configuration", "detail 2", nil, nil, task.PriorityNone)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	_, err = s.Add("Run validation tests", "detail 3", nil, nil, task.PriorityNone)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	t.Setenv("TSSK_ROOT", dir)

	// Search for "tion" substring - should match multiple tasks
	listTitle = "tion"
	defer func() { listTitle = "" }()

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmd := listCmd
	cmd.SetOut(w)
	cmd.SetErr(w)

	err = cmd.RunE(cmd, nil)

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
	// Should match "Final configuraTion" and "validaTion tests"
	if !strings.Contains(output, "Final configuration") {
		t.Errorf("expected 'Final configuration' in output, got: %s", output)
	}
	if !strings.Contains(output, "Run validation tests") {
		t.Errorf("expected 'Run validation tests' in output, got: %s", output)
	}
	// Should not match "Initialize project structure"
	if strings.Contains(output, "Initialize project structure") {
		t.Errorf("did not expect 'Initialize project structure' in output, got: %s", output)
	}
}
