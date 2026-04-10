package task_test

import (
	"testing"

	"github.com/bmordue/tssk/internal/task"
)

func TestStatusIsValid(t *testing.T) {
	valid := []task.Status{
		task.StatusTodo,
		task.StatusInProgress,
		task.StatusInReview,
		task.StatusDone,
		task.StatusBlocked,
	}
	for _, s := range valid {
		if !s.IsValid() {
			t.Errorf("expected %q to be valid", s)
		}
	}

	invalid := task.Status("unknown")
	if invalid.IsValid() {
		t.Errorf("expected %q to be invalid", invalid)
	}
}

func TestComputeDocHash(t *testing.T) {
	tk := &task.Task{
		ID:     "T-1",
		Title:  "Hello",
		Status: task.StatusTodo,
	}
	if err := tk.ComputeDocHash(); err != nil {
		t.Fatalf("ComputeDocHash error: %v", err)
	}
	if len(tk.DocHash) != 64 {
		t.Errorf("expected 64-character hex hash, got %d chars: %s", len(tk.DocHash), tk.DocHash)
	}

	// Hash should be deterministic.
	hash1 := tk.DocHash
	if err := tk.ComputeDocHash(); err != nil {
		t.Fatalf("ComputeDocHash error on second call: %v", err)
	}
	if tk.DocHash != hash1 {
		t.Errorf("hash is not deterministic: %s vs %s", hash1, tk.DocHash)
	}
}

func TestComputeDocHashN(t *testing.T) {
	tk := &task.Task{
		ID:    "T-2",
		Title: "Hash length test",
	}

	for _, length := range []int{8, 16, 32, 64} {
		if err := tk.ComputeDocHashN(length); err != nil {
			t.Fatalf("ComputeDocHashN(%d) error: %v", length, err)
		}
		if len(tk.DocHash) != length {
			t.Errorf("ComputeDocHashN(%d): expected %d chars, got %d: %s", length, length, len(tk.DocHash), tk.DocHash)
		}
	}

	// Values out of range should fall back to the full 64-char hash.
	for _, bad := range []int{0, -1, 65, 100} {
		if err := tk.ComputeDocHashN(bad); err != nil {
			t.Fatalf("ComputeDocHashN(%d) error: %v", bad, err)
		}
		if len(tk.DocHash) != 64 {
			t.Errorf("ComputeDocHashN(%d): expected fallback to 64 chars, got %d", bad, len(tk.DocHash))
		}
	}

	// A length-N hash must be a prefix of the full hash.
	if err := tk.ComputeDocHashN(64); err != nil {
		t.Fatalf("ComputeDocHashN(64) error: %v", err)
	}
	full := tk.DocHash
	if err := tk.ComputeDocHashN(16); err != nil {
		t.Fatalf("ComputeDocHashN(16) error: %v", err)
	}
	if tk.DocHash != full[:16] {
		t.Errorf("short hash is not a prefix of full hash: %s vs %s", tk.DocHash, full[:16])
	}
}

func TestAddRemoveDependency(t *testing.T) {
	tk := &task.Task{ID: "T-2"}

	added := tk.AddDependency("T-1")
	if !added {
		t.Error("expected AddDependency to return true for new dep")
	}
	if !tk.HasDependency("T-1") {
		t.Error("expected HasDependency to return true after adding")
	}

	// Adding the same dep twice should return false.
	added = tk.AddDependency("T-1")
	if added {
		t.Error("expected AddDependency to return false for duplicate dep")
	}
	if len(tk.Dependencies) != 1 {
		t.Errorf("expected 1 dependency, got %d", len(tk.Dependencies))
	}

	removed := tk.RemoveDependency("T-1")
	if !removed {
		t.Error("expected RemoveDependency to return true")
	}
	if tk.HasDependency("T-1") {
		t.Error("expected HasDependency to return false after removal")
	}

	// Removing non-existent dep should return false.
	removed = tk.RemoveDependency("T-99")
	if removed {
		t.Error("expected RemoveDependency to return false for missing dep")
	}
}

func TestValidatePhaseGate(t *testing.T) {
	tests := []struct {
		name        string
		current     task.Status
		next        task.Status
		shouldError bool
	}{
		// Same status (no-op)
		{"todo -> todo", task.StatusTodo, task.StatusTodo, false},
		{"in-progress -> in-progress", task.StatusInProgress, task.StatusInProgress, false},
		{"in-review -> in-review", task.StatusInReview, task.StatusInReview, false},
		{"done -> done", task.StatusDone, task.StatusDone, false},

		// Valid transitions
		{"todo -> in-progress", task.StatusTodo, task.StatusInProgress, false},
		{"in-progress -> in-review", task.StatusInProgress, task.StatusInReview, false},
		{"in-review -> done", task.StatusInReview, task.StatusDone, false},
		{"in-review -> in-progress (rework)", task.StatusInReview, task.StatusInProgress, false},
		{"todo -> blocked", task.StatusTodo, task.StatusBlocked, false},
		{"in-progress -> blocked", task.StatusInProgress, task.StatusBlocked, false},
		{"in-review -> blocked", task.StatusInReview, task.StatusBlocked, false},
		{"done -> blocked", task.StatusDone, task.StatusBlocked, false},
		{"blocked -> in-progress", task.StatusBlocked, task.StatusInProgress, false},
		{"blocked -> todo (reset)", task.StatusBlocked, task.StatusTodo, false},
		{"in-progress -> todo (reset)", task.StatusInProgress, task.StatusTodo, false},
		{"in-review -> todo (reset)", task.StatusInReview, task.StatusTodo, false},
		{"done -> todo (reset)", task.StatusDone, task.StatusTodo, false},

		// Invalid transitions
		{"todo -> done (skip review)", task.StatusTodo, task.StatusDone, true},
		{"in-progress -> done (skip review)", task.StatusInProgress, task.StatusDone, true},
		{"blocked -> done (skip review)", task.StatusBlocked, task.StatusDone, true},
		{"todo -> in-review (skip in-progress)", task.StatusTodo, task.StatusInReview, true},
		{"done -> in-progress", task.StatusDone, task.StatusInProgress, true},
		{"done -> in-review", task.StatusDone, task.StatusInReview, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tk := &task.Task{ID: "1", Status: tt.current}
			err := tk.ValidatePhaseGate(tt.next)

			if tt.shouldError && err == nil {
				t.Errorf("expected error for transition %s -> %s", tt.current, tt.next)
			}
			if !tt.shouldError && err != nil {
				t.Errorf("unexpected error for transition %s -> %s: %v", tt.current, tt.next, err)
			}
		})
	}
}

func TestHasPhaseTag(t *testing.T) {
	tests := []struct {
		name     string
		tags     []string
		hasPhase bool
	}{
		{"no tags", nil, false},
		{"empty tags", []string{}, false},
		{"phase tag present", []string{"phase-1"}, true},
		{"phase tag with others", []string{"bug", "phase-2", "urgent"}, true},
		{"no phase tag", []string{"bug", "feature"}, false},
		{"similar but not phase", []string{"phased", "phase"}, false},
		{"empty phase tag", []string{"phase-"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tk := &task.Task{ID: "1", Tags: tt.tags}
			if got := tk.HasPhaseTag(); got != tt.hasPhase {
				t.Errorf("HasPhaseTag() = %v, want %v", got, tt.hasPhase)
			}
		})
	}
}

func TestGetPhase(t *testing.T) {
	tests := []struct {
		name      string
		tags      []string
		wantPhase int
	}{
		{"no tags", nil, 0},
		{"phase-1", []string{"phase-1"}, 1},
		{"phase-3", []string{"phase-3"}, 3},
		{"phase-10", []string{"phase-10"}, 10},
		{"multiple tags", []string{"bug", "phase-2", "urgent"}, 2},
		{"no phase tag", []string{"bug", "feature"}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tk := &task.Task{ID: "1", Tags: tt.tags}
			if got := tk.GetPhase(); got != tt.wantPhase {
				t.Errorf("GetPhase() = %v, want %v", got, tt.wantPhase)
			}
		})
	}
}
