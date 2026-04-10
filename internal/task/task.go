package task

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Status represents the lifecycle state of a task.
type Status string

const (
	StatusTodo       Status = "todo"
	StatusInProgress Status = "in-progress"
	StatusInReview   Status = "in-review"
	StatusDone       Status = "done"
	StatusBlocked    Status = "blocked"
)

// ValidStatuses lists all accepted status values.
var ValidStatuses = []Status{StatusTodo, StatusInProgress, StatusInReview, StatusDone, StatusBlocked}

// PhaseGateStatuses defines the valid progression path for phase gates.
// Tasks must pass through in-review before being marked done.
var PhaseGateStatuses = []Status{StatusInReview}

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

// HasTag reports whether tag is present in the task's tag list.
func (t *Task) HasTag(tag string) bool {
	for _, tg := range t.Tags {
		if tg == tag {
			return true
		}
	}
	return false
}

// AddTag appends tag to the task's tag list if not already present.
// Returns false if tag was already present.
func (t *Task) AddTag(tag string) bool {
	if t.HasTag(tag) {
		return false
	}
	t.Tags = append(t.Tags, tag)
	return true
}

// RemoveTag removes tag from the task's tag list.  Returns false if tag was
// not in the list.
func (t *Task) RemoveTag(tag string) bool {
	for i, tg := range t.Tags {
		if tg == tag {
			t.Tags = append(t.Tags[:i], t.Tags[i+1:]...)
			return true
		}
	}
	return false
}

// ValidatePhaseGate checks if the status transition is valid according to
// phase gate rules. Returns an error if the transition violates the rules.
//
// Phase gate rules:
// - Tasks must be in-review before they can be marked done
// - Tasks cannot move from done back to in-progress or todo without explicit override
func (t *Task) ValidatePhaseGate(newStatus Status) error {
	// No-op if status isn't changing
	if newStatus == t.Status {
		return nil
	}

	// Allow transitions to blocked from any state
	if newStatus == StatusBlocked {
		return nil
	}

	// Allow transitions to in-progress from todo, blocked, or in-review (rework)
	if newStatus == StatusInProgress && (t.Status == StatusTodo || t.Status == StatusBlocked || t.Status == StatusInReview) {
		return nil
	}

	// Allow transitions to in-review from in-progress
	if newStatus == StatusInReview && t.Status == StatusInProgress {
		return nil
	}

	// Allow transitions to done only from in-review
	if newStatus == StatusDone {
		if t.Status != StatusInReview {
			return fmt.Errorf("phase gate violation: task %s must be in-review before marking as done (current: %s)", t.ID, t.Status)
		}
		return nil
	}

	// Allow transitions to todo from any state (reset)
	if newStatus == StatusTodo {
		return nil
	}

	return fmt.Errorf("invalid status transition: %s -> %s", t.Status, newStatus)
}

// HasPhaseTag reports whether the task has a phase tag (phase-N).
func (t *Task) HasPhaseTag() bool {
	for _, tag := range t.Tags {
		if strings.HasPrefix(tag, "phase-") && len(tag) > 6 {
			return true
		}
	}
	return false
}

// GetPhase returns the phase number from the task's phase tag, or 0 if none.
func (t *Task) GetPhase() int {
	for _, tag := range t.Tags {
		if strings.HasPrefix(tag, "phase-") {
			var phase int
			if _, err := fmt.Sscanf(tag, "phase-%d", &phase); err == nil {
				return phase
			}
		}
	}
	return 0
}
