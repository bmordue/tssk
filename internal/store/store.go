package store

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
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

// Store manages persistence of tasks using a pluggable Backend.
// All high-level task operations (Add, Get, UpdateStatus, …) are implemented
// here; raw I/O is delegated to the Backend.
type Store struct {
	backend Backend
}

// New creates a Store backed by the local filesystem rooted at root.
// It is equivalent to NewWithBackend(NewLocalBackend(root)).
func New(root string) *Store {
	return NewWithBackend(NewLocalBackend(root))
}

// NewWithBackend creates a Store that uses the supplied Backend for all I/O.
func NewWithBackend(b Backend) *Store {
	return &Store{backend: b}
}

// HealthCheck delegates to the underlying Backend's HealthCheck.
func (s *Store) HealthCheck() error {
	return s.backend.HealthCheck()
}

// LoadAll reads all tasks from the backend.  Returns an empty slice when
// the store is empty or has not been initialised yet.
func (s *Store) LoadAll() ([]*task.Task, error) {
	data, err := s.backend.ReadTasksData()
	if err != nil {
		return nil, fmt.Errorf("loading tasks: %w", err)
	}
	if len(data) == 0 {
		return []*task.Task{}, nil
	}

	var tasks []*task.Task
	lineNum := 0
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		t := new(task.Task)
		if err := json.Unmarshal([]byte(line), t); err != nil {
			return nil, fmt.Errorf("parsing tasks line %d: %w", lineNum, err)
		}
		tasks = append(tasks, t)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanning tasks data: %w", err)
	}
	return tasks, nil
}

// saveAll serialises the task slice as JSONL and persists it via the backend.
func (s *Store) saveAll(tasks []*task.Task) error {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	for _, t := range tasks {
		if err := enc.Encode(t); err != nil {
			return fmt.Errorf("serialising task %s: %w", t.ID, err)
		}
	}
	if err := s.backend.WriteTasksData(buf.Bytes()); err != nil {
		return fmt.Errorf("saving tasks: %w", err)
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

// Add creates a new task with the given title and detail text, writes the
// detail markdown via the backend, and appends the task metadata.
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

	if err := s.backend.WriteDetail(t.DocHash, []byte(detail)); err != nil {
		return nil, fmt.Errorf("writing detail: %w", err)
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
	data, err := s.backend.ReadDetail(t.DocHash)
	if err != nil {
		return "", fmt.Errorf("reading detail: %w", err)
	}
	return string(data), nil
}

// generateID produces the next sequential task ID (T-1, T-2, …).
func generateID(tasks []*task.Task) string {
	return fmt.Sprintf("T-%d", len(tasks)+1)
}
