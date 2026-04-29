package task_test

import (
	"encoding/hex"
	"testing"

	"pgregory.net/rapid"

	"github.com/bmordue/tssk/internal/task"
)

// --- Status and Priority validation ---

func TestProperty_StatusIsValid_OnlyKnownStatuses(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		s := task.Status(rapid.String().Draw(t, "status"))
		valid := s.IsValid()
		isKnown := false
		for _, v := range task.ValidStatuses {
			if s == v {
				isKnown = true
				break
			}
		}
		if valid != isKnown {
			t.Fatalf("IsValid(%q) = %v, but isKnown = %v", s, valid, isKnown)
		}
	})
}

func TestProperty_PriorityIsValid_OnlyKnownPriorities(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		p := task.Priority(rapid.String().Draw(t, "priority"))
		valid := p.IsValid()
		isKnown := false
		for _, v := range task.ValidPriorities {
			if p == v {
				isKnown = true
				break
			}
		}
		if valid != isKnown {
			t.Fatalf("IsValid(%q) = %v, but isKnown = %v", p, valid, isKnown)
		}
	})
}

// --- ComputeDocHash properties ---

func TestProperty_ComputeDocHash_Deterministic(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		tk := &task.Task{
			ID:    rapid.String().Draw(t, "id"),
			Title: rapid.String().Draw(t, "title"),
		}
		if err := tk.ComputeDocHash(); err != nil {
			t.Fatalf("ComputeDocHash: %v", err)
		}
		h1 := tk.DocHash
		if err := tk.ComputeDocHash(); err != nil {
			t.Fatalf("ComputeDocHash second call: %v", err)
		}
		if tk.DocHash != h1 {
			t.Fatalf("hash not deterministic: %s vs %s", h1, tk.DocHash)
		}
	})
}

func TestProperty_ComputeDocHash_ValidHex64(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		tk := &task.Task{
			ID:    rapid.String().Draw(t, "id"),
			Title: rapid.String().Draw(t, "title"),
		}
		if err := tk.ComputeDocHash(); err != nil {
			t.Fatalf("ComputeDocHash: %v", err)
		}
		if len(tk.DocHash) != 64 {
			t.Fatalf("expected 64-char hash, got %d", len(tk.DocHash))
		}
		if _, err := hex.DecodeString(tk.DocHash); err != nil {
			t.Fatalf("hash is not valid hex: %s", tk.DocHash)
		}
	})
}

func TestProperty_ComputeDocHashN_PrefixOfFull(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		tk := &task.Task{
			ID:    rapid.String().Draw(t, "id"),
			Title: rapid.String().Draw(t, "title"),
		}
		n := rapid.IntRange(1, 64).Draw(t, "length")
		if err := tk.ComputeDocHashN(64); err != nil {
			t.Fatal(err)
		}
		full := tk.DocHash
		if err := tk.ComputeDocHashN(n); err != nil {
			t.Fatal(err)
		}
		if tk.DocHash != full[:n] {
			t.Fatalf("ComputeDocHashN(%d) = %s, expected prefix %s", n, tk.DocHash, full[:n])
		}
	})
}

func TestProperty_ComputeDocHashN_OutOfRange_FullHash(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		tk := &task.Task{
			ID:    rapid.String().Draw(t, "id"),
			Title: rapid.String().Draw(t, "title"),
		}
		// Generate out-of-range values: <= 0 or > 64
		bad := rapid.OneOf(
			rapid.IntRange(-100, 0),
			rapid.IntRange(65, 200),
		).Draw(t, "bad_length")
		if err := tk.ComputeDocHashN(bad); err != nil {
			t.Fatal(err)
		}
		if len(tk.DocHash) != 64 {
			t.Fatalf("out-of-range length %d produced %d-char hash", bad, len(tk.DocHash))
		}
	})
}

// --- Dependency set properties ---

func TestProperty_AddDependency_Idempotent(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		tk := &task.Task{ID: "T-1"}
		dep := rapid.String().Draw(t, "dep")
		tk.AddDependency(dep)
		countAfterFirst := len(tk.Dependencies)
		tk.AddDependency(dep)
		if len(tk.Dependencies) != countAfterFirst {
			t.Fatalf("AddDependency not idempotent: %d vs %d", countAfterFirst, len(tk.Dependencies))
		}
	})
}

func TestProperty_AddRemoveDependency_RoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		tk := &task.Task{ID: "T-1"}
		dep := rapid.String().Draw(t, "dep")
		tk.AddDependency(dep)
		if !tk.HasDependency(dep) {
			t.Fatal("HasDependency false after Add")
		}
		tk.RemoveDependency(dep)
		if tk.HasDependency(dep) {
			t.Fatal("HasDependency true after Remove")
		}
	})
}

func TestProperty_Dependencies_NoDuplicates(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		tk := &task.Task{ID: "T-1"}
		deps := rapid.SliceOf(rapid.String()).Draw(t, "deps")
		for _, d := range deps {
			tk.AddDependency(d)
		}
		seen := make(map[string]bool, len(tk.Dependencies))
		for _, d := range tk.Dependencies {
			if seen[d] {
				t.Fatalf("duplicate dependency: %q", d)
			}
			seen[d] = true
		}
	})
}

// --- Tag set properties ---

func TestProperty_AddTag_Idempotent(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		tk := &task.Task{ID: "T-1"}
		tag := rapid.String().Draw(t, "tag")
		tk.AddTag(tag)
		countAfterFirst := len(tk.Tags)
		tk.AddTag(tag)
		if len(tk.Tags) != countAfterFirst {
			t.Fatalf("AddTag not idempotent: %d vs %d", countAfterFirst, len(tk.Tags))
		}
	})
}

func TestProperty_AddRemoveTag_RoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		tk := &task.Task{ID: "T-1"}
		tag := rapid.String().Draw(t, "tag")
		tk.AddTag(tag)
		if !tk.HasTag(tag) {
			t.Fatal("HasTag false after Add")
		}
		tk.RemoveTag(tag)
		if tk.HasTag(tag) {
			t.Fatal("HasTag true after Remove")
		}
	})
}

func TestProperty_Tags_NoDuplicates(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		tk := &task.Task{ID: "T-1"}
		tags := rapid.SliceOf(rapid.String()).Draw(t, "tags")
		for _, tg := range tags {
			tk.AddTag(tg)
		}
		seen := make(map[string]bool, len(tk.Tags))
		for _, tg := range tk.Tags {
			if seen[tg] {
				t.Fatalf("duplicate tag: %q", tg)
			}
			seen[tg] = true
		}
	})
}

// --- MetaJSON determinism ---

func TestProperty_MetaJSON_Deterministic(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		tk := &task.Task{
			ID:    rapid.String().Draw(t, "id"),
			Title: rapid.String().Draw(t, "title"),
		}
		b1, err := tk.MetaJSON()
		if err != nil {
			t.Fatal(err)
		}
		b2, err := tk.MetaJSON()
		if err != nil {
			t.Fatal(err)
		}
		if string(b1) != string(b2) {
			t.Fatal("MetaJSON not deterministic")
		}
	})
}
