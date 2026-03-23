package store

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bmordue/tssk/internal/task"
)

const (
	tasksFile = "tasks.jsonl"
	docsDir   = "docs"
)

// ErrNotFound is returned when a task with the given ID cannot be located.
var ErrNotFound = errors.New("task not found")

// Store manages persistence of tasks in a JSONL metadata file and
// content-addressed markdown detail files.
type Store struct {
	root string // project root directory
}

// New creates a Store rooted at the given directory.
func New(root string) *Store {
	return &Store{root: root}
}

// tasksPath returns the absolute path to the JSONL metadata file.
func (s *Store) tasksPath() string {
	return filepath.Join(s.root, tasksFile)
}

// docPath returns the absolute path to the markdown detail file for a task.
func (s *Store) docPath(docHash string) string {
	return filepath.Join(s.root, docsDir, docHash+".md")
}

// ensureDocsDir creates the docs directory if it does not already exist.
func (s *Store) ensureDocsDir() error {
	return os.MkdirAll(filepath.Join(s.root, docsDir), 0o755)
}

// LoadAll reads all tasks from the JSONL file.  Returns an empty slice if the
// file does not exist yet.
func (s *Store) LoadAll() ([]*task.Task, error) {
	f, err := os.Open(s.tasksPath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []*task.Task{}, nil
		}
		return nil, fmt.Errorf("opening tasks file: %w", err)
	}
	defer f.Close()

	var tasks []*task.Task
	scanner := bufio.NewScanner(f)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var t task.Task
		if err := json.Unmarshal([]byte(line), &t); err != nil {
			return nil, fmt.Errorf("parsing tasks file line %d: %w", lineNum, err)
		}
		tasks = append(tasks, &t)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading tasks file: %w", err)
	}
	return tasks, nil
}

// saveAll overwrites the JSONL file with the given task slice.
func (s *Store) saveAll(tasks []*task.Task) error {
	f, err := os.Create(s.tasksPath())
	if err != nil {
		return fmt.Errorf("creating tasks file: %w", err)
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	for _, t := range tasks {
		if err := enc.Encode(t); err != nil {
			return fmt.Errorf("writing task %s: %w", t.ID, err)
		}
	}
	return nil
}

// Get returns the task with the given ID.
func (s *Store) Get(id string) (*task.Task, error) {
	tasks, err := s.LoadAll()
	if err != nil {
		return nil, err
	}
	for _, t := range tasks {
		if t.ID == id {
			return t, nil
		}
	}
	return nil, fmt.Errorf("%w: %s", ErrNotFound, id)
}

// Add creates a new task with the given title and detail text, persists its
// metadata to the JSONL file, and writes the detail markdown file.
func (s *Store) Add(title, detail string, deps []string) (*task.Task, error) {
	tasks, err := s.LoadAll()
	if err != nil {
		return nil, err
	}

	t := &task.Task{
		ID:           generateID(tasks),
		Title:        title,
		Status:       task.StatusTodo,
		Dependencies: deps,
		CreatedAt:    time.Now().UTC(),
	}

	if err := t.ComputeDocHash(); err != nil {
		return nil, fmt.Errorf("computing doc hash: %w", err)
	}

	if err := s.ensureDocsDir(); err != nil {
		return nil, fmt.Errorf("creating docs directory: %w", err)
	}

	if err := os.WriteFile(s.docPath(t.DocHash), []byte(detail), 0o644); err != nil {
		return nil, fmt.Errorf("writing detail file: %w", err)
	}

	tasks = append(tasks, t)
	if err := s.saveAll(tasks); err != nil {
		return nil, err
	}
	return t, nil
}

// UpdateStatus changes the status of the task with the given ID.
func (s *Store) UpdateStatus(id string, status task.Status) (*task.Task, error) {
	tasks, err := s.LoadAll()
	if err != nil {
		return nil, err
	}

	var found *task.Task
	for _, t := range tasks {
		if t.ID == id {
			t.Status = status
			found = t
			break
		}
	}
	if found == nil {
		return nil, fmt.Errorf("%w: %s", ErrNotFound, id)
	}

	if err := s.saveAll(tasks); err != nil {
		return nil, err
	}
	return found, nil
}

// AddDep appends depID to the dependency list of the task with id.
func (s *Store) AddDep(id, depID string) error {
	tasks, err := s.LoadAll()
	if err != nil {
		return err
	}

	var found *task.Task
	for _, t := range tasks {
		if t.ID == id {
			found = t
			break
		}
	}
	if found == nil {
		return fmt.Errorf("%w: %s", ErrNotFound, id)
	}

	if !found.AddDependency(depID) {
		return fmt.Errorf("task %s already depends on %s", id, depID)
	}

	return s.saveAll(tasks)
}

// RemoveDep removes depID from the dependency list of the task with id.
func (s *Store) RemoveDep(id, depID string) error {
	tasks, err := s.LoadAll()
	if err != nil {
		return err
	}

	var found *task.Task
	for _, t := range tasks {
		if t.ID == id {
			found = t
			break
		}
	}
	if found == nil {
		return fmt.Errorf("%w: %s", ErrNotFound, id)
	}

	if !found.RemoveDependency(depID) {
		return fmt.Errorf("task %s does not depend on %s", id, depID)
	}

	return s.saveAll(tasks)
}

// ReadDetail returns the markdown detail text for a task.
func (s *Store) ReadDetail(t *task.Task) (string, error) {
	b, err := os.ReadFile(s.docPath(t.DocHash))
	if err != nil {
		return "", fmt.Errorf("reading detail file: %w", err)
	}
	return string(b), nil
}

// generateID produces the next sequential task ID (T-1, T-2, …).
func generateID(tasks []*task.Task) string {
	return fmt.Sprintf("T-%d", len(tasks)+1)
}
