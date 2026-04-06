package store_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/bmordue/tssk/internal/store"
	"github.com/bmordue/tssk/internal/task"
)

// ---- helpers ----------------------------------------------------------------

// newNamedTempStore creates a fresh store rooted at a temp directory and
// pre-populates it with the given task titles.  It returns the store and its
// root directory.
func newNamedTempStore(t *testing.T, titles ...string) (*store.Store, string) {
	t.Helper()
	dir := t.TempDir()
	s := store.New(dir)
	for _, title := range titles {
		if _, err := s.Add(title, "", nil); err != nil {
			t.Fatalf("Add %q: %v", title, err)
		}
	}
	return s, dir
}

// writeTssk writes a .tssk.json file into root.
func writeTssk(t *testing.T, root string, content any) {
	t.Helper()
	b, err := json.MarshalIndent(content, "", "  ")
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, ".tssk.json"), b, 0o600); err != nil {
		t.Fatalf("write .tssk.json: %v", err)
	}
}

// ---- MultiStore tests -------------------------------------------------------

func TestMultiStoreLoadAll_PrimaryOnly(t *testing.T) {
	primary, _ := newNamedTempStore(t, "Alpha", "Beta")

	ms := store.NewMultiStoreWithCollections(primary, nil)
	got, err := ms.LoadAll()
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(got))
	}
	for _, ct := range got {
		if ct.Collection != "" {
			t.Errorf("primary task should have empty Collection, got %q", ct.Collection)
		}
	}
}

func TestMultiStoreLoadAll_WithCollections(t *testing.T) {
	primary, _ := newNamedTempStore(t, "P1")
	coll1, _ := newNamedTempStore(t, "C1a", "C1b")
	coll2, _ := newNamedTempStore(t, "C2a")

	ms := store.NewMultiStoreWithCollections(primary, []store.NamedStore{
		{Name: "proj1", Store: coll1},
		{Name: "proj2", Store: coll2},
	})

	got, err := ms.LoadAll()
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	if len(got) != 4 {
		t.Fatalf("expected 4 tasks, got %d", len(got))
	}

	// First task should be from primary.
	if got[0].Collection != "" {
		t.Errorf("first task should be from primary, got collection %q", got[0].Collection)
	}
	// Second and third from proj1.
	if got[1].Collection != "proj1" || got[2].Collection != "proj1" {
		t.Errorf("expected proj1, got %q/%q", got[1].Collection, got[2].Collection)
	}
	// Fourth from proj2.
	if got[3].Collection != "proj2" {
		t.Errorf("expected proj2, got %q", got[3].Collection)
	}
}

func TestMultiStoreGet_Primary(t *testing.T) {
	primary, _ := newNamedTempStore(t, "Main task")

	ms := store.NewMultiStoreWithCollections(primary, nil)
	ct, err := ms.Get("1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if ct.Task.Title != "Main task" {
		t.Errorf("unexpected title: %q", ct.Task.Title)
	}
	if ct.Collection != "" {
		t.Errorf("expected empty collection, got %q", ct.Collection)
	}
}

func TestMultiStoreGet_QualifiedID(t *testing.T) {
	primary, _ := newNamedTempStore(t, "P1")
	coll, _ := newNamedTempStore(t, "Coll task")

	ms := store.NewMultiStoreWithCollections(primary, []store.NamedStore{
		{Name: "other", Store: coll},
	})

	ct, err := ms.Get("other:1")
	if err != nil {
		t.Fatalf("Get(other:1): %v", err)
	}
	if ct.Task.Title != "Coll task" {
		t.Errorf("unexpected title: %q", ct.Task.Title)
	}
	if ct.Collection != "other" {
		t.Errorf("expected collection=other, got %q", ct.Collection)
	}
}

func TestMultiStoreGet_UnknownCollection(t *testing.T) {
	primary, _ := newNamedTempStore(t, "P1")
	ms := store.NewMultiStoreWithCollections(primary, nil)

	_, err := ms.Get("unknown:1")
	if err == nil {
		t.Fatal("expected error for unknown collection")
	}
}

func TestCollectedTaskQualifiedID(t *testing.T) {
	tk := &task.Task{ID: "3"}

	primary := store.CollectedTask{Task: tk, Collection: ""}
	if primary.QualifiedID() != "3" {
		t.Errorf("primary qualified ID = %q, want %q", primary.QualifiedID(), "3")
	}

	named := store.CollectedTask{Task: tk, Collection: "frontend"}
	if named.QualifiedID() != "frontend:3" {
		t.Errorf("named qualified ID = %q, want %q", named.QualifiedID(), "frontend:3")
	}
}

func TestMultiStoreCheckDeps_NoDeps(t *testing.T) {
	primary, _ := newNamedTempStore(t, "Standalone")
	ms := store.NewMultiStoreWithCollections(primary, nil)

	blocking, allDone, err := ms.CheckDeps("1")
	if err != nil {
		t.Fatalf("CheckDeps: %v", err)
	}
	if !allDone {
		t.Error("expected allDone=true for task with no deps")
	}
	if len(blocking) != 0 {
		t.Errorf("expected no blocking, got %v", blocking)
	}
}

func TestMultiStoreCheckDeps_SameCollection(t *testing.T) {
	_, dir := newNamedTempStore(t, "Dep", "Dependent")

	// Mark task 1 as todo (not done) and make task 2 depend on task 1.
	s := store.New(dir)
	if err := s.AddDep("2", "1"); err != nil {
		t.Fatalf("AddDep: %v", err)
	}

	ms := store.NewMultiStoreWithCollections(s, nil)
	blocking, allDone, err := ms.CheckDeps("2")
	if err != nil {
		t.Fatalf("CheckDeps: %v", err)
	}
	if allDone {
		t.Error("expected allDone=false; dep is not done")
	}
	if len(blocking) != 1 {
		t.Fatalf("expected 1 blocking task, got %d", len(blocking))
	}
	if blocking[0].Task.ID != "1" {
		t.Errorf("unexpected blocking task ID: %q", blocking[0].Task.ID)
	}
}

func TestMultiStoreCheckDeps_CrossCollection(t *testing.T) {
	// Set up two named collections: "projectA" and "projectB".
	// projectB:1 has a cross-collection dependency on projectA:1 (todo → not done).
	storeA, _ := newNamedTempStore(t, "ProjectA dep")
	storeB, dirB := newNamedTempStore(t, "Cross dep task")

	// Rewrite storeB's tasks.jsonl so that task 1 has dep "projectA:1".
	ot1, err := storeB.Get("1")
	if err != nil {
		t.Fatalf("get storeB:1: %v", err)
	}
	type rawTask struct {
		ID           string   `json:"id"`
		Title        string   `json:"title"`
		Status       string   `json:"status"`
		Dependencies []string `json:"dependencies"`
		CreatedAt    string   `json:"created_at"`
		DocHash      string   `json:"doc_hash"`
	}
	rt := rawTask{
		ID:           ot1.ID,
		Title:        ot1.Title,
		Status:       string(ot1.Status),
		Dependencies: []string{"projectA:1"}, // Cross-collection dep.
		CreatedAt:    ot1.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		DocHash:      ot1.DocHash,
	}
	b, _ := json.Marshal(rt)
	if err := os.MkdirAll(filepath.Join(dirB, ".tsks"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dirB, ".tsks", "tasks.jsonl"), append(b, '\n'), 0o600); err != nil {
		t.Fatalf("write tasks: %v", err)
	}

	storeBReloaded := store.New(dirB)

	ms := store.NewMultiStoreWithCollections(nil, []store.NamedStore{
		{Name: "projectA", Store: storeA},
		{Name: "projectB", Store: storeBReloaded},
	})

	// projectB:1 depends on projectA:1 which is todo (not done).
	blocking, allDone, err := ms.CheckDeps("projectB:1")
	if err != nil {
		t.Fatalf("CheckDeps: %v", err)
	}
	if allDone {
		t.Error("expected allDone=false; cross-collection dep is not done")
	}
	if len(blocking) != 1 {
		t.Fatalf("expected 1 blocking, got %d: %v", len(blocking), blocking)
	}
	// The blocking task is projectA:1.
	if blocking[0].QualifiedID() != "projectA:1" {
		t.Errorf("unexpected blocking qualified ID: %q, want %q", blocking[0].QualifiedID(), "projectA:1")
	}
}

// ---- Config collections tests -----------------------------------------------

func TestConfigFromFileAndEnv_Collections(t *testing.T) {
	root := t.TempDir()
	collDir := t.TempDir()

	writeTssk(t, root, map[string]any{
		"backend":    "local",
		"tasks_file": ".tsks/tasks.jsonl",
		"collections": []map[string]any{
			{
				"name": "sub",
				"root": collDir,
			},
		},
	})

	cfg, err := store.ConfigFromFileAndEnv(root)
	if err != nil {
		t.Fatalf("ConfigFromFileAndEnv: %v", err)
	}
	if len(cfg.Collections) != 1 {
		t.Fatalf("expected 1 collection, got %d", len(cfg.Collections))
	}
	if cfg.Collections[0].Name != "sub" {
		t.Errorf("collection name = %q, want sub", cfg.Collections[0].Name)
	}
	if cfg.Collections[0].Root != collDir {
		t.Errorf("collection root = %q, want %q", cfg.Collections[0].Root, collDir)
	}
}

func TestConfigFromFileAndEnv_CollectionRelativeRoot(t *testing.T) {
	root := t.TempDir()

	writeTssk(t, root, map[string]any{
		"collections": []map[string]any{
			{"name": "rel", "root": "subproject"},
		},
	})

	cfg, err := store.ConfigFromFileAndEnv(root)
	if err != nil {
		t.Fatalf("ConfigFromFileAndEnv: %v", err)
	}
	expected := filepath.Join(root, "subproject")
	if cfg.Collections[0].Root != expected {
		t.Errorf("expected absolute root %q, got %q", expected, cfg.Collections[0].Root)
	}
}

func TestConfigFromFileAndEnv_CollectionMissingName(t *testing.T) {
	root := t.TempDir()
	writeTssk(t, root, map[string]any{
		"collections": []map[string]any{
			{"root": "/some/path"},
		},
	})

	_, err := store.ConfigFromFileAndEnv(root)
	if err == nil {
		t.Fatal("expected error for collection without name")
	}
}

func TestConfigFromFileAndEnv_CollectionInvalidHashLength(t *testing.T) {
	root := t.TempDir()
	writeTssk(t, root, map[string]any{
		"collections": []map[string]any{
			{"name": "x", "display_hash_length": 65},
		},
	})

	_, err := store.ConfigFromFileAndEnv(root)
	if err == nil {
		t.Fatal("expected error for invalid display_hash_length in collection")
	}
}

func TestMultiStoreFromConfig_NoCollections(t *testing.T) {
	root := t.TempDir()
	cfg, err := store.ConfigFromFileAndEnv(root)
	if err != nil {
		t.Fatalf("ConfigFromFileAndEnv: %v", err)
	}
	ms, err := store.MultiStoreFromConfig(cfg)
	if err != nil {
		t.Fatalf("MultiStoreFromConfig: %v", err)
	}

	// Add a task and verify it is visible via the MultiStore.
	s, err := store.NewFromConfig(cfg)
	if err != nil {
		t.Fatalf("NewFromConfig: %v", err)
	}
	if _, err := s.Add("Hello", "", nil); err != nil {
		t.Fatalf("Add: %v", err)
	}

	all, err := ms.LoadAll()
	if err != nil {
		t.Fatalf("ms.LoadAll: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("expected 1 task, got %d", len(all))
	}
}

func TestMultiStoreFromConfig_NamedPrimary(t *testing.T) {
	root := t.TempDir()
	collRoot := t.TempDir()

	writeTssk(t, root, map[string]any{
		"name": "main",
		"collections": []map[string]any{
			{"name": "other", "root": collRoot},
		},
	})

	cfg, err := store.ConfigFromFileAndEnv(root)
	if err != nil {
		t.Fatalf("ConfigFromFileAndEnv: %v", err)
	}
	if cfg.Name != "main" {
		t.Fatalf("expected Name=main, got %q", cfg.Name)
	}

	// Add a task to the primary and one to the named collection.
	primaryS := store.New(root)
	if _, err := primaryS.Add("Primary task", "", nil); err != nil {
		t.Fatalf("Add primary: %v", err)
	}
	collS := store.New(collRoot)
	if _, err := collS.Add("Coll task", "", nil); err != nil {
		t.Fatalf("Add coll: %v", err)
	}

	ms, err := store.MultiStoreFromConfig(cfg)
	if err != nil {
		t.Fatalf("MultiStoreFromConfig: %v", err)
	}

	// The primary task should be reachable as "main:1".
	ct, err := ms.Get("main:1")
	if err != nil {
		t.Fatalf("Get(main:1): %v", err)
	}
	if ct.Task.Title != "Primary task" {
		t.Errorf("unexpected title: %q", ct.Task.Title)
	}
	if ct.Collection != "main" {
		t.Errorf("expected collection=main, got %q", ct.Collection)
	}

	// And tasks should show the "main" collection in LoadAll.
	all, err := ms.LoadAll()
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(all))
	}
	if all[0].Collection != "main" {
		t.Errorf("primary task collection = %q, want main", all[0].Collection)
	}
}

func TestMultiStoreGet_MalformedID(t *testing.T) {
primary, _ := newNamedTempStore(t, "Task")
ms := store.NewMultiStoreWithCollections(primary, nil)

for _, bad := range []string{":1", "frontend:"} {
_, err := ms.Get(bad)
if err == nil {
t.Errorf("Get(%q): expected error for malformed ID", bad)
}
}
}

func TestMultiStoreCheckDeps_MissingDepHasBlockedStatus(t *testing.T) {
primary, primaryDir := newNamedTempStore(t, "Task with missing dep")

// Manually inject a non-existent dep.
type rawTask struct {
ID           string   `json:"id"`
Title        string   `json:"title"`
Status       string   `json:"status"`
Dependencies []string `json:"dependencies"`
CreatedAt    string   `json:"created_at"`
DocHash      string   `json:"doc_hash"`
}
t1, err := primary.Get("1")
if err != nil {
t.Fatalf("Get 1: %v", err)
}
rt := rawTask{
ID:           t1.ID,
Title:        t1.Title,
Status:       string(t1.Status),
Dependencies: []string{"99"}, // does not exist
CreatedAt:    t1.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
DocHash:      t1.DocHash,
}
b, _ := json.Marshal(rt)
if err := os.WriteFile(filepath.Join(primaryDir, ".tsks", "tasks.jsonl"), append(b, '\n'), 0o600); err != nil {
t.Fatalf("write tasks: %v", err)
}

s := store.New(primaryDir)
ms := store.NewMultiStoreWithCollections(s, nil)

blocking, allDone, err := ms.CheckDeps("1")
if err != nil {
t.Fatalf("CheckDeps: %v", err)
}
if allDone {
t.Error("expected allDone=false for missing dep")
}
if len(blocking) != 1 {
t.Fatalf("expected 1 blocking, got %d", len(blocking))
}
if blocking[0].Status != task.StatusBlocked {
t.Errorf("missing dep placeholder should have status=blocked, got %q", blocking[0].Status)
}
}

func TestMultiStoreFromConfig_DuplicateCollectionName(t *testing.T) {
dir1 := t.TempDir()
dir2 := t.TempDir()
root := t.TempDir()

writeTssk(t, root, map[string]any{
"collections": []map[string]any{
{"name": "dup", "root": dir1},
{"name": "dup", "root": dir2},
},
})

cfg, err := store.ConfigFromFileAndEnv(root)
if err != nil {
t.Fatalf("ConfigFromFileAndEnv: %v", err)
}
_, err = store.MultiStoreFromConfig(cfg)
if err == nil {
t.Fatal("expected error for duplicate collection names")
}
}

func TestMultiStoreFromConfig_CollectionNameCollidesPrimaryName(t *testing.T) {
collRoot := t.TempDir()
root := t.TempDir()

writeTssk(t, root, map[string]any{
"name": "main",
"collections": []map[string]any{
{"name": "main", "root": collRoot},
},
})

cfg, err := store.ConfigFromFileAndEnv(root)
if err != nil {
t.Fatalf("ConfigFromFileAndEnv: %v", err)
}
_, err = store.MultiStoreFromConfig(cfg)
if err == nil {
t.Fatal("expected error for collection name colliding with primary name")
}
}

func TestConfigFromFileAndEnv_CollectionLocalRequiresRoot(t *testing.T) {
root := t.TempDir()
writeTssk(t, root, map[string]any{
"collections": []map[string]any{
{"name": "missing-root"},
},
})

_, err := store.ConfigFromFileAndEnv(root)
if err == nil {
t.Fatal("expected error for local collection without root")
}
}
