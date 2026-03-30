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
	defaultTasksFile         = ".tsks/tasks.jsonl"
	defaultTasksFileExt      = ".jsonl"
	defaultDocsDir           = ".tsks/docs"
	DefaultDisplayHashLength = 9
)

// ErrNotFound is returned when a task with the given ID cannot be located.
var ErrNotFound = errors.New("task not found")

// ErrAmbiguous is returned when a prefix matches more than one task ID.
var ErrAmbiguous = errors.New("ambiguous task prefix")

// Store manages persistence of tasks using a pluggable Backend.
// All high-level task operations (Add, Get, UpdateStatus, …) are implemented
// here; raw I/O is delegated to the Backend.
type Store struct {
	backend           Backend
	metrics           *Metrics
	displayHashLength int
}

// New creates a Store backed by the local filesystem rooted at root.
// It is equivalent to NewWithBackend(NewLocalBackend(root)).
func New(root string) *Store {
	return NewWithBackend(NewLocalBackend(root))
}

// NewWithBackend creates a Store that uses the supplied Backend for all I/O.
func NewWithBackend(b Backend) *Store {
	return &Store{backend: b, displayHashLength: DefaultDisplayHashLength}
}

// HealthCheck delegates to the underlying Backend's HealthCheck.
func (s *Store) HealthCheck() error {
	return s.backend.HealthCheck()
}

// Metrics returns the metrics collector associated with this Store, or nil
// when the Store was not created via NewFromConfig (i.e. no MeteredBackend).
func (s *Store) Metrics() *Metrics {
	return s.metrics
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

// Get returns the task whose ID exactly matches id, or – when no exact match
// exists – the unique task whose ID has id as a unique prefix.
func (s *Store) Get(id string) (*task.Task, error) {
	tasks, err := s.LoadAll()
	if err != nil {
		return nil, err
	}
	return resolveOne(tasks, id)
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

	displayKey := t.DocHash[:s.displayHashLength]
	if err := s.backend.WriteDetail(displayKey, []byte(detail)); err != nil {
		return nil, fmt.Errorf("writing detail: %w", err)
	}

	tasks = append(tasks, t)
	if err := s.saveAll(tasks); err != nil {
		// Best-effort rollback: remove the detail we just wrote to avoid
		// leaving it orphaned while the task list does not reference it.
		if derr := s.backend.DeleteDetail(displayKey); derr != nil {
			return nil, fmt.Errorf("failed to save tasks: %w; rollback also failed: %v", err, derr)
		}
		return nil, err
	}
	return t, nil
}

// UpdateStatus changes the status of the task identified by id (exact or unique prefix).
func (s *Store) UpdateStatus(id string, status task.Status) (*task.Task, error) {
	tasks, err := s.LoadAll()
	if err != nil {
		return nil, err
	}

	found, err := resolveOne(tasks, id)
	if err != nil {
		return nil, err
	}
	found.Status = status
	if err := s.saveAll(tasks); err != nil {
		return nil, err
	}
	return found, nil
}

// AddDep appends dep to the dependency list of the task identified by id.
// Both id and dep may be full IDs or unique prefixes.
func (s *Store) AddDep(id, dep string) error {
	tasks, err := s.LoadAll()
	if err != nil {
		return err
	}

	found, err := resolveOne(tasks, id)
	if err != nil {
		return err
	}
	depTask, err := resolveOne(tasks, dep)
	if err != nil {
		return fmt.Errorf("dependency: %w", err)
	}

	if !found.AddDependency(depTask.ID) {
		return fmt.Errorf("task %s already depends on %s", found.ID, depTask.ID)
	}

	return s.saveAll(tasks)
}

// RemoveDep removes dep from the dependency list of the task identified by id.
// Both id and dep may be full IDs or unique prefixes.
func (s *Store) RemoveDep(id, dep string) error {
	tasks, err := s.LoadAll()
	if err != nil {
		return err
	}

	found, err := resolveOne(tasks, id)
	if err != nil {
		return err
	}
	depTask, err := resolveOne(tasks, dep)
	if err != nil {
		return fmt.Errorf("dependency: %w", err)
	}

	if !found.RemoveDependency(depTask.ID) {
		return fmt.Errorf("task %s does not depend on %s", found.ID, depTask.ID)
	}

	return s.saveAll(tasks)
}

// ReadDetail returns the markdown detail text for a task.
func (s *Store) ReadDetail(t *task.Task) (string, error) {
	displayKey := t.DocHash[:s.displayHashLength]
	data, err := s.backend.ReadDetail(displayKey)
	if err != nil {
		return "", fmt.Errorf("reading detail: %w", err)
	}
	return string(data), nil
}

// resolveOne returns the unique task whose ID equals prefix exactly, or – if
// no exact match exists – the unique task whose ID begins with prefix.
// Returns ErrNotFound when no task matches, ErrAmbiguous when multiple tasks
// share the same prefix.
func resolveOne(tasks []*task.Task, prefix string) (*task.Task, error) {
	// Exact match always wins over prefix matching.
	for _, t := range tasks {
		if t.ID == prefix {
			return t, nil
		}
	}
	// Collect all tasks whose ID begins with prefix.
	var matches []*task.Task
	for _, t := range tasks {
		if strings.HasPrefix(t.ID, prefix) {
			matches = append(matches, t)
		}
	}
	switch len(matches) {
	case 1:
		return matches[0], nil
	case 0:
		return nil, fmt.Errorf("%w: %s", ErrNotFound, prefix)
	default:
		ids := make([]string, len(matches))
		for i, m := range matches {
			ids[i] = m.ID
		}
		return nil, fmt.Errorf("%w: %q matches: %s", ErrAmbiguous, prefix, strings.Join(ids, ", "))
	}
}

// generateID produces the next sequential task ID (1, 2, 3, …).
func generateID(tasks []*task.Task) string {
	return fmt.Sprintf("%d", len(tasks)+1)
}
