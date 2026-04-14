package store

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/bmordue/tssk/internal/task"
)

const (
	defaultTasksFile         = ".tsks/tasks.jsonl"
	defaultTasksFileExt      = ".jsonl"
	defaultDocsDir           = ".tsks/docs"
	defaultDisplayHashLength = 9
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
	cache             []*task.Task
	idMap             map[string]*task.Task
	sortedIDs         []string
}

// New creates a Store backed by the local filesystem rooted at root.
// It is equivalent to NewWithBackend(NewLocalBackend(root)).
func New(root string) *Store {
	return NewWithBackend(NewLocalBackend(root))
}

// NewWithBackend creates a Store that uses the supplied Backend for all I/O.
func NewWithBackend(b Backend) *Store {
	return &Store{backend: b, displayHashLength: defaultDisplayHashLength}
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
//
// The returned slice and the task pointers within it are owned by the Store.
// Callers must not mutate the returned tasks or the slice.
func (s *Store) LoadAll() ([]*task.Task, error) {
	if s.cache != nil {
		return s.cache, nil
	}

	data, err := s.backend.ReadTasksData()
	if err != nil {
		return nil, fmt.Errorf("loading tasks: %w", err)
	}
	if len(data) == 0 {
		s.cache = []*task.Task{}
		s.idMap = make(map[string]*task.Task)
		return s.cache, nil
	}

	// Pre-allocate the tasks slice to reduce re-allocations.
	lineCount := bytes.Count(data, []byte("\n"))
	tasks := make([]*task.Task, 0, lineCount)
	idMap := make(map[string]*task.Task, lineCount)

	decoder := json.NewDecoder(bytes.NewReader(data))
	for decoder.More() {
		t := new(task.Task)
		if err := decoder.Decode(t); err != nil {
			return nil, fmt.Errorf("parsing tasks: %w", err)
		}
		tasks = append(tasks, t)
		idMap[t.ID] = t
	}
	s.cache = tasks
	s.idMap = idMap

	sortedIDs := make([]string, 0, len(idMap))
	for id := range idMap {
		sortedIDs = append(sortedIDs, id)
	}
	sort.Strings(sortedIDs)
	s.sortedIDs = sortedIDs

	return tasks, nil
}

// saveAll serialises the task slice as JSONL and persists it via the backend.
func (s *Store) saveAll(tasks []*task.Task) error {
	// Pre-allocate buffer with an estimated 256 bytes per task to reduce re-allocations.
	var buf bytes.Buffer
	buf.Grow(len(tasks) * 256)

	for _, t := range tasks {
		// json.Marshal is measurably faster than json.Encoder.Encode for this use case.
		data, err := json.Marshal(t)
		if err != nil {
			return fmt.Errorf("serialising task %s: %w", t.ID, err)
		}
		buf.Write(data)
		buf.WriteByte('\n')
	}

	if err := s.backend.WriteTasksData(buf.Bytes()); err != nil {
		// Invalidate the cache on failure because 'tasks' (which likely
		// contains in-memory mutations) was not successfully persisted.
		s.cache = nil
		s.idMap = nil
		s.sortedIDs = nil
		return fmt.Errorf("saving tasks: %w", err)
	}

	// Optimization: check if we can skip index rebuilding.
	// If the tasks slice contains the exact same pointers as our cache,
	// then neither the idMap nor the sortedIDs need to change.
	if s.idMap != nil && s.sortedIDs != nil && len(tasks) == len(s.cache) {
		same := true
		for i := range tasks {
			if tasks[i] != s.cache[i] {
				same = false
				break
			}
		}
		if same {
			return nil
		}
	}

	// Optimization: incremental update if exactly one task was appended.
	if s.idMap != nil && s.sortedIDs != nil && len(tasks) == len(s.cache)+1 {
		same := true
		for i := range s.cache {
			if tasks[i] != s.cache[i] {
				same = false
				break
			}
		}
		if same {
			newT := tasks[len(tasks)-1]
			if _, exists := s.idMap[newT.ID]; !exists {
				s.idMap[newT.ID] = newT
				idx := sort.SearchStrings(s.sortedIDs, newT.ID)
				s.sortedIDs = append(s.sortedIDs, "")
				copy(s.sortedIDs[idx+1:], s.sortedIDs[idx:])
				s.sortedIDs[idx] = newT.ID
				s.cache = tasks
				return nil
			}
		}
	}

	// Fallback: full rebuild of indexes.
	s.cache = tasks
	if s.idMap == nil {
		s.idMap = make(map[string]*task.Task, len(tasks))
	} else {
		clear(s.idMap)
	}

	for _, t := range tasks {
		s.idMap[t.ID] = t
	}

	s.sortedIDs = make([]string, 0, len(s.idMap))
	for id := range s.idMap {
		s.sortedIDs = append(s.sortedIDs, id)
	}
	sort.Strings(s.sortedIDs)

	return nil
}

// Get returns the task whose ID exactly matches id, or – when no exact match
// exists – the unique task whose ID has id as a unique prefix.
func (s *Store) Get(id string) (*task.Task, error) {
	if _, err := s.LoadAll(); err != nil {
		return nil, err
	}
	return s.resolveOne(id)
}

// Add creates a new task with the given title and detail text, using deps as
// the initial dependency task IDs and tags as the initial tags assigned to the
// task, writes the detail markdown via the backend, and appends the task
// metadata.
func (s *Store) Add(title, detail string, deps []string, tags []string) (*task.Task, error) {
	tasks, err := s.LoadAll()
	if err != nil {
		return nil, err
	}

	t := &task.Task{
		ID:           generateID(tasks),
		Title:        title,
		Status:       task.StatusTodo,
		Dependencies: deps,
		Tags:         tags,
		CreatedAt:    time.Now().UTC(),
	}

	if err := t.ComputeDocHash(); err != nil {
		return nil, fmt.Errorf("computing doc hash: %w", err)
	}

	// Use truncated hash for filename based on displayHashLength
	filenameHash := t.DocHash
	if s.displayHashLength > 0 && s.displayHashLength < len(t.DocHash) {
		filenameHash = t.DocHash[:s.displayHashLength]
	}
	if err := s.backend.WriteDetail(filenameHash, []byte(detail)); err != nil {
		return nil, fmt.Errorf("writing detail: %w", err)
	}

	tasks = append(tasks, t)
	if err := s.saveAll(tasks); err != nil {
		// Best-effort rollback: remove the detail we just wrote to avoid
		// leaving it orphaned while the task list does not reference it.
		if derr := s.backend.DeleteDetail(filenameHash); derr != nil {
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

	found, err := s.resolveOne(id)
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

	found, err := s.resolveOne(id)
	if err != nil {
		return err
	}
	depTask, err := s.resolveOne(dep)
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

	found, err := s.resolveOne(id)
	if err != nil {
		return err
	}
	depTask, err := s.resolveOne(dep)
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
	// Use truncated hash for filename based on displayHashLength.
	filenameHash := t.DocHash
	if s.displayHashLength > 0 && s.displayHashLength < len(t.DocHash) {
		filenameHash = t.DocHash[:s.displayHashLength]
	}

	data, err := s.backend.ReadDetail(filenameHash)
	if err != nil && errors.Is(err, ErrNotFound) && filenameHash != t.DocHash {
		// Backward compatibility: older versions stored details using the
		// full hash as the filename, so fall back to that legacy location.
		data, err = s.backend.ReadDetail(t.DocHash)
	}
	if err != nil {
		return "", fmt.Errorf("reading detail: %w", err)
	}
	return string(data), nil
}

// AddTags appends tags to the task identified by id.
func (s *Store) AddTags(id string, tags []string) error {
	tasks, err := s.LoadAll()
	if err != nil {
		return err
	}

	found, err := s.resolveOne(id)
	if err != nil {
		return err
	}

	for _, tag := range tags {
		found.AddTag(tag)
	}

	return s.saveAll(tasks)
}

// RemoveTags removes tags from the task identified by id.
func (s *Store) RemoveTags(id string, tags []string) error {
	tasks, err := s.LoadAll()
	if err != nil {
		return err
	}

	found, err := s.resolveOne(id)
	if err != nil {
		return err
	}

	for _, tag := range tags {
		found.RemoveTag(tag)
	}

	return s.saveAll(tasks)
}

// SetTags replaces all tags on the task identified by id.
func (s *Store) SetTags(id string, tags []string) error {
	tasks, err := s.LoadAll()
	if err != nil {
		return err
	}

	found, err := s.resolveOne(id)
	if err != nil {
		return err
	}

	found.Tags = tags
	return s.saveAll(tasks)
}

// resolveOne returns the unique task whose ID equals prefix exactly, or – if
// no exact match exists – the unique task whose ID begins with prefix.
// Returns ErrNotFound when no task matches, ErrAmbiguous when multiple tasks
// share the same prefix.
func (s *Store) resolveOne(prefix string) (*task.Task, error) {
	// Exact match always wins over prefix matching.
	if t, ok := s.idMap[prefix]; ok {
		return t, nil
	}

	// Use binary search on sortedIDs to find tasks whose ID begins with prefix.
	i := sort.SearchStrings(s.sortedIDs, prefix)
	if i == len(s.sortedIDs) || !strings.HasPrefix(s.sortedIDs[i], prefix) {
		return nil, fmt.Errorf("%w: %s", ErrNotFound, prefix)
	}

	// Optimization: if there's only one match, return it without allocating a slice.
	if i+1 == len(s.sortedIDs) || !strings.HasPrefix(s.sortedIDs[i+1], prefix) {
		return s.idMap[s.sortedIDs[i]], nil
	}

	// Ambiguous match: collect IDs for a helpful error message.
	const maxErrorIDs = 5
	var matches []string
	for j := i; j < len(s.sortedIDs) && len(matches) < maxErrorIDs; j++ {
		id := s.sortedIDs[j]
		if !strings.HasPrefix(id, prefix) {
			break
		}
		matches = append(matches, id)
	}

	msg := strings.Join(matches, ", ")
	if i+maxErrorIDs < len(s.sortedIDs) && strings.HasPrefix(s.sortedIDs[i+maxErrorIDs], prefix) {
		msg += ", ..."
	}
	return nil, fmt.Errorf("%w: %q matches: %s", ErrAmbiguous, prefix, msg)
}

// generateID produces the next sequential task ID (1, 2, 3, …).
func generateID(tasks []*task.Task) string {
	return strconv.Itoa(len(tasks) + 1)
}
