package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/bmordue/tssk/internal/store"
	"github.com/bmordue/tssk/internal/task"
)

func setupReadyTestStore(t *testing.T) (*store.Store, string) {
	t.Helper()
	dir := t.TempDir()
	s := store.New(dir)
	return s, dir
}

func TestReadyCommand_NoTasks(t *testing.T) {
	_, dir := setupReadyTestStore(t)

	t.Setenv("TSSK_ROOT", dir)

	cmd := readyCmd
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should output empty table with header
	output := buf.String()
	if output == "" {
		t.Error("expected output, got empty string")
	}
}

func TestReadyCommand_AllReady(t *testing.T) {
	s, dir := setupReadyTestStore(t)

	// Add tasks with no dependencies - all should be ready
	_, err := s.Add("Task 1", "detail 1", nil, nil)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	_, err = s.Add("Task 2", "detail 2", nil, nil)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	t.Setenv("TSSK_ROOT", dir)

	cmd := readyCmd
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	err = cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	// Should contain both tasks
	if !bytes.Contains(buf.Bytes(), []byte("Task 1")) {
		t.Errorf("expected 'Task 1' in output, got: %s", output)
	}
	if !bytes.Contains(buf.Bytes(), []byte("Task 2")) {
		t.Errorf("expected 'Task 2' in output, got: %s", output)
	}
}

func TestReadyCommand_BlockedByTodo(t *testing.T) {
	s, dir := setupReadyTestStore(t)

	// Add a todo task
	task1, err := s.Add("Blocking task", "detail 1", nil, nil)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	// Add another task that depends on it
	_, err = s.Add("Dependent task", "detail 2", []string{task1.ID}, nil)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	t.Setenv("TSSK_ROOT", dir)

	cmd := readyCmd
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	err = cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	// Should contain blocking task but not dependent task
	if !bytes.Contains(buf.Bytes(), []byte("Blocking task")) {
		t.Errorf("expected 'Blocking task' in output, got: %s", output)
	}
	if bytes.Contains(buf.Bytes(), []byte("Dependent task")) {
		t.Errorf("did not expect 'Dependent task' in output, got: %s", output)
	}
}

func TestReadyCommand_BlockedByInProgress(t *testing.T) {
	s, dir := setupReadyTestStore(t)

	// Add an in-progress task
	task1, err := s.Add("In-progress task", "detail 1", nil, nil)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if _, err = s.UpdateStatus(task1.ID, task.StatusInProgress); err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}

	// Add another task that depends on it
	_, err = s.Add("Dependent task", "detail 2", []string{task1.ID}, nil)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	t.Setenv("TSSK_ROOT", dir)

	cmd := readyCmd
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	err = cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	// Should not contain dependent task (blocked by in-progress)
	if bytes.Contains(buf.Bytes(), []byte("Dependent task")) {
		t.Errorf("did not expect 'Dependent task' in output, got: %s", output)
	}
}

func TestReadyCommand_DoneDependency(t *testing.T) {
	s, dir := setupReadyTestStore(t)

	// Add a done task
	task1, err := s.Add("Done task", "detail 1", nil, nil)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if _, err = s.UpdateStatus(task1.ID, task.StatusDone); err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}

	// Add another task that depends on it - should be ready
	_, err = s.Add("Dependent task", "detail 2", []string{task1.ID}, nil)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	t.Setenv("TSSK_ROOT", dir)

	cmd := readyCmd
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	err = cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	// Should contain dependent task (dependency is done)
	if !bytes.Contains(buf.Bytes(), []byte("Dependent task")) {
		t.Errorf("expected 'Dependent task' in output, got: %s", output)
	}
	// Should not contain done task (it's not todo)
	if bytes.Contains(buf.Bytes(), []byte("Done task")) {
		t.Errorf("did not expect 'Done task' in output, got: %s", output)
	}
}

func TestReadyCommand_BlockedDependency(t *testing.T) {
	s, dir := setupReadyTestStore(t)

	// Add a blocked task
	task1, err := s.Add("Blocked task", "detail 1", nil, nil)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if _, err = s.UpdateStatus(task1.ID, task.StatusBlocked); err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}

	// Add another task that depends on it - should be ready (blocked doesn't block)
	_, err = s.Add("Dependent task", "detail 2", []string{task1.ID}, nil)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	t.Setenv("TSSK_ROOT", dir)

	cmd := readyCmd
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	err = cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	// Should contain dependent task (dependency is blocked, not todo/in-progress)
	if !bytes.Contains(buf.Bytes(), []byte("Dependent task")) {
		t.Errorf("expected 'Dependent task' in output, got: %s", output)
	}
}

func TestReadyCommand_JSONOutput(t *testing.T) {
	s, dir := setupReadyTestStore(t)

	// Add a ready task
	_, err := s.Add("Ready task", "detail 1", nil, nil)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	t.Setenv("TSSK_ROOT", dir)

	// Set JSON flag
	readyJSON = true
	defer func() { readyJSON = false }()

	// Capture stdout since printJSON writes to it
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmd := readyCmd
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

	// Should be valid JSON
	var tasks []task.Task
	if err := json.Unmarshal(outBuf.Bytes(), &tasks); err != nil {
		t.Fatalf("invalid JSON output: %v, got: %q", err, outBuf.String())
	}

	if len(tasks) != 1 {
		t.Errorf("expected 1 task in JSON, got %d", len(tasks))
	}
	if len(tasks) > 0 && tasks[0].Title != "Ready task" {
		t.Errorf("expected 'Ready task', got %q", tasks[0].Title)
	}
}

func TestReadyCommand_MissingDependency(t *testing.T) {
	s, dir := setupReadyTestStore(t)

	// Add a task with a non-existent dependency
	_, err := s.Add("Task with missing dep", "detail 1", []string{"999"}, nil)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	t.Setenv("TSSK_ROOT", dir)

	cmd := readyCmd
	var outBuf bytes.Buffer
	var errBuf bytes.Buffer
	cmd.SetOut(&outBuf)
	cmd.SetErr(&errBuf)

	err = cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := outBuf.String()
	// Should not contain the task (missing dep treated as blocking)
	if bytes.Contains(outBuf.Bytes(), []byte("Task with missing dep")) {
		t.Errorf("did not expect 'Task with missing dep' in output (missing dep should block), got: %s", output)
	}

	if errBuf.Len() == 0 {
		t.Errorf("expected warning output on stderr for missing dependency, got none")
	}
}

func TestReadyCommand_NoDependencies(t *testing.T) {
	s, dir := setupReadyTestStore(t)

	// Add tasks with various statuses but no dependencies
	_, err := s.Add("Todo task 1", "detail 1", nil, nil)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	task2, err2 := s.Add("Todo task 2", "detail 2", nil, nil)
	if err2 != nil {
		t.Fatalf("Add: %v", err2)
	}
	// Set to in-progress
	if _, err2 = s.UpdateStatus(task2.ID, task.StatusInProgress); err2 != nil {
		t.Fatalf("UpdateStatus: %v", err2)
	}

	task3, err3 := s.Add("Todo task 3", "detail 3", nil, nil)
	if err3 != nil {
		t.Fatalf("Add: %v", err3)
	}
	// Set to done
	if _, err3 = s.UpdateStatus(task3.ID, task.StatusDone); err3 != nil {
		t.Fatalf("UpdateStatus: %v", err3)
	}

	t.Setenv("TSSK_ROOT", dir)

	// Verify store is set up correctly
	allTasks, err := s.LoadAll()
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	if len(allTasks) != 3 {
		t.Fatalf("expected 3 tasks in store, got %d", len(allTasks))
	}

	cmd := readyCmd
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	err = cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	// Should only contain todo tasks without blocking deps
	// Task 1 should be ready (no deps, status todo)
	if !bytes.Contains(buf.Bytes(), []byte("Todo task 1")) {
		t.Errorf("expected 'Todo task 1' in output, got: %s", output)
	}
	// Task 2 should not be ready (status in-progress)
	if bytes.Contains(buf.Bytes(), []byte("Todo task 2")) {
		t.Errorf("did not expect 'Todo task 2' in output (in-progress), got: %s", output)
	}
	// Task 3 should not be ready (status done)
	if bytes.Contains(buf.Bytes(), []byte("Todo task 3")) {
		t.Errorf("did not expect 'Todo task 3' in output (done), got: %s", output)
	}
}

func TestReadyCommand_FlagReset(t *testing.T) {
	// Ensure the flag is reset after tests
	original := readyJSON
	defer func() { readyJSON = original }()

	// Create a temp dir for this test
	dir, err := os.MkdirTemp("", "tssk-ready-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() {
		if rmErr := os.RemoveAll(dir); rmErr != nil {
			t.Logf("warning: failed to remove temp dir: %v", rmErr)
		}
	}()

	t.Setenv("TSSK_ROOT", dir)

	cmd := readyCmd
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	// Should work without error even with flag reset
	err = cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
