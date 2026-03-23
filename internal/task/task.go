package task

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"
)

// Status represents the lifecycle state of a task.
type Status string

const (
	StatusTodo       Status = "todo"
	StatusInProgress Status = "in-progress"
	StatusDone       Status = "done"
	StatusBlocked    Status = "blocked"
)

// ValidStatuses lists all accepted status values.
var ValidStatuses = []Status{StatusTodo, StatusInProgress, StatusDone, StatusBlocked}

// IsValid reports whether s is a recognised status string.
func (s Status) IsValid() bool {
	for _, v := range ValidStatuses {
		if s == v {
			return true
		}
	}
	return false
}

// Task holds the metadata stored in the JSONL file.
type Task struct {
	ID           string    `json:"id"`
	Title        string    `json:"title"`
	Status       Status    `json:"status"`
	Dependencies []string  `json:"dependencies,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	// DocHash is the SHA-256 hash of the metadata JSON blob; it is also the
	// filename (without extension) of the corresponding markdown detail file.
	DocHash string `json:"doc_hash"`
}

// MetaJSON returns the canonical JSON representation of the task metadata
// fields that are used as input to the content-address hash.  DocHash itself
// is intentionally excluded so the hash is deterministic at creation time.
func (t *Task) MetaJSON() ([]byte, error) {
	type hashable struct {
		ID           string    `json:"id"`
		Title        string    `json:"title"`
		Status       Status    `json:"status"`
		Dependencies []string  `json:"dependencies,omitempty"`
		CreatedAt    time.Time `json:"created_at"`
	}
	return json.Marshal(hashable{
		ID:           t.ID,
		Title:        t.Title,
		Status:       t.Status,
		Dependencies: t.Dependencies,
		CreatedAt:    t.CreatedAt,
	})
}

// ComputeDocHash calculates the SHA-256 hex digest of the task's metadata
// JSON and stores it in DocHash.
func (t *Task) ComputeDocHash() error {
	b, err := t.MetaJSON()
	if err != nil {
		return err
	}
	t.DocHash = fmt.Sprintf("%x", sha256.Sum256(b))
	return nil
}

// HasDependency reports whether id is listed as a dependency of t.
func (t *Task) HasDependency(id string) bool {
	for _, d := range t.Dependencies {
		if d == id {
			return true
		}
	}
	return false
}

// AddDependency appends id to the task's dependency list if not already
// present.  Returns false if id was already a dependency.
func (t *Task) AddDependency(id string) bool {
	if t.HasDependency(id) {
		return false
	}
	t.Dependencies = append(t.Dependencies, id)
	return true
}

// RemoveDependency removes id from the task's dependency list.  Returns false
// if id was not in the list.
func (t *Task) RemoveDependency(id string) bool {
	for i, d := range t.Dependencies {
		if d == id {
			t.Dependencies = append(t.Dependencies[:i], t.Dependencies[i+1:]...)
			return true
		}
	}
	return false
}
