package task

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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

// Priority represents the urgency level of a task.
type Priority string

const (
	PriorityNone     Priority = ""
	PriorityLow      Priority = "low"
	PriorityMedium   Priority = "medium"
	PriorityHigh     Priority = "high"
	PriorityCritical Priority = "critical"
)

// ValidPriorities lists all accepted priority values.
var ValidPriorities = []Priority{PriorityNone, PriorityLow, PriorityMedium, PriorityHigh, PriorityCritical}

// IsValid reports whether p is a recognised priority string.
func (p Priority) IsValid() bool {
	for _, v := range ValidPriorities {
		if p == v {
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
	Priority     Priority  `json:"priority,omitempty"`
	Dependencies []string  `json:"dependencies,omitempty"`
	Tags         []string  `json:"tags,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	// DocHash is the SHA-256 hash of the canonical JSON representation of the
	// immutable task metadata fields (id, title, created_at); it is also the
	// filename (without extension) of the corresponding markdown detail file.
	DocHash string `json:"doc_hash"`
}

// MetaJSON returns the canonical JSON representation of the immutable task
// metadata fields that are used as input to the content-address hash.
// DocHash itself is intentionally excluded so the hash is deterministic at
// creation time.
func (t *Task) MetaJSON() ([]byte, error) {
	type hashable struct {
		ID        string    `json:"id"`
		Title     string    `json:"title"`
		CreatedAt time.Time `json:"created_at"`
	}
	return json.Marshal(hashable{
		ID:        t.ID,
		Title:     t.Title,
		CreatedAt: t.CreatedAt,
	})
}

// ComputeDocHash calculates the full 64-character SHA-256 hex digest of the
// task's metadata JSON and stores it in DocHash.
func (t *Task) ComputeDocHash() error {
	return t.ComputeDocHashN(0)
}

// ComputeDocHashN calculates the SHA-256 hex digest of the task's metadata
// JSON and stores the first length characters in DocHash.  length must be
// between 1 and 64; any value outside that range (including 0) uses the full
// 64-character digest.
func (t *Task) ComputeDocHashN(length int) error {
	b, err := t.MetaJSON()
	if err != nil {
		return err
	}
	sum := sha256.Sum256(b)
	full := hex.EncodeToString(sum[:])
	if length < 1 || length > len(full) {
		length = len(full)
	}
	t.DocHash = full[:length]
	return nil
}

// hasString reports whether val is present in slice.
func hasString(slice []string, val string) bool {
	for _, s := range slice {
		if s == val {
			return true
		}
	}
	return false
}

// addString appends val to slice if not already present.
// Returns the updated slice and true when val was added, or the original slice
// and false when val was already present.
func addString(slice []string, val string) ([]string, bool) {
	if hasString(slice, val) {
		return slice, false
	}
	return append(slice, val), true
}

// removeString removes the first occurrence of val from slice.
// Returns the updated slice and true when val was found, or the original slice
// and false when val was not present.
func removeString(slice []string, val string) ([]string, bool) {
	for i, s := range slice {
		if s == val {
			return append(slice[:i], slice[i+1:]...), true
		}
	}
	return slice, false
}

// HasDependency reports whether id is listed as a dependency of t.
func (t *Task) HasDependency(id string) bool {
	return hasString(t.Dependencies, id)
}

// AddDependency appends id to the task's dependency list if not already
// present.  Returns false if id was already a dependency.
func (t *Task) AddDependency(id string) bool {
	deps, added := addString(t.Dependencies, id)
	if added {
		t.Dependencies = deps
	}
	return added
}

// RemoveDependency removes id from the task's dependency list.  Returns false
// if id was not in the list.
func (t *Task) RemoveDependency(id string) bool {
	deps, removed := removeString(t.Dependencies, id)
	if removed {
		t.Dependencies = deps
	}
	return removed
}

// HasTag reports whether tag is present in the task's tag list.
func (t *Task) HasTag(tag string) bool {
	return hasString(t.Tags, tag)
}

// AddTag appends tag to the task's tag list if not already present.
// Returns false if tag was already present.
func (t *Task) AddTag(tag string) bool {
	tags, added := addString(t.Tags, tag)
	if added {
		t.Tags = tags
	}
	return added
}

// RemoveTag removes tag from the task's tag list.  Returns false if tag was
// not in the list.
func (t *Task) RemoveTag(tag string) bool {
	tags, removed := removeString(t.Tags, tag)
	if removed {
		t.Tags = tags
	}
	return removed
}
