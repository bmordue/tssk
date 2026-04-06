package store

import (
	"fmt"
	"strings"

	"github.com/bmordue/tssk/internal/task"
)

// CollectedTask pairs a task with the name of the collection it came from.
// When Collection is empty, the task belongs to the primary (unnamed) store.
type CollectedTask struct {
	*task.Task
	// Collection is the name of the source collection, or "" for the primary
	// store configured by the root .tssk.json.
	Collection string
}

// QualifiedID returns the fully-qualified task identifier.
// For tasks in a named collection the format is "{collection}:{id}" (e.g.
// "frontend:3").  For tasks in the primary collection the plain ID is
// returned unchanged.
func (ct CollectedTask) QualifiedID() string {
	if ct.Collection == "" {
		return ct.Task.ID
	}
	return ct.Collection + ":" + ct.Task.ID
}

// NamedStore pairs a Store with its collection name for use in a MultiStore.
type NamedStore struct {
	Name  string
	Store *Store
}

// namedStore is the internal representation stored in MultiStore.
type namedStore struct {
	name  string
	store *Store
}

// MultiStore aggregates a primary Store with zero or more named collection
// Stores.  Read operations (LoadAll, Get, CheckDeps) span all stores.
type MultiStore struct {
	primary     *Store
	primaryName string // optional name for the primary store
	collections []namedStore
}

// NewMultiStoreWithCollections creates a MultiStore from a primary Store and a
// slice of NamedStore values.  This is the preferred public constructor when
// building a MultiStore from named collection stores.
func NewMultiStoreWithCollections(primary *Store, collections []NamedStore) *MultiStore {
	internal := make([]namedStore, len(collections))
	for i, ns := range collections {
		internal[i] = namedStore{name: ns.Name, store: ns.Store}
	}
	return &MultiStore{primary: primary, collections: internal}
}

// newMultiStore creates a MultiStore with a pre-converted internal slice.
// Used internally by MultiStoreFromConfig.
func newMultiStore(primary *Store, collections []namedStore) *MultiStore {
	return &MultiStore{primary: primary, collections: collections}
}

// LoadAll returns tasks from every store, in order: primary first, then each
// named collection.  Each CollectedTask carries the collection name so callers
// can qualify or display it as needed.
func (m *MultiStore) LoadAll() ([]CollectedTask, error) {
	var all []CollectedTask

	if m.primary != nil {
		tasks, err := m.primary.LoadAll()
		if err != nil {
			return nil, fmt.Errorf("loading primary collection: %w", err)
		}
		for _, t := range tasks {
			all = append(all, CollectedTask{Task: t, Collection: m.primaryName})
		}
	}

	for _, ns := range m.collections {
		tasks, err := ns.store.LoadAll()
		if err != nil {
			return nil, fmt.Errorf("loading collection %q: %w", ns.name, err)
		}
		for _, t := range tasks {
			all = append(all, CollectedTask{Task: t, Collection: ns.name})
		}
	}

	return all, nil
}

// Get resolves a qualified or unqualified task ID.
//
//   - A qualified ID has the form "{collection}:{id}" and is resolved against
//     the named collection only.  If the primary store has a name (set via
//     MultiStoreFromConfig when the top-level config has a "name" field), a
//     qualified ID using that name is resolved against the primary store.
//   - An unqualified ID is resolved against the primary store only.
func (m *MultiStore) Get(qualifiedID string) (CollectedTask, error) {
	collection, id, err := splitQualifiedID(qualifiedID)
	if err != nil {
		return CollectedTask{}, err
	}

	if collection == "" {
		// Unqualified: look in primary.
		if m.primary == nil {
			return CollectedTask{}, fmt.Errorf("%w: %s (no primary collection)", ErrNotFound, qualifiedID)
		}
		t, err := m.primary.Get(id)
		if err != nil {
			return CollectedTask{}, err
		}
		return CollectedTask{Task: t, Collection: m.primaryName}, nil
	}

	// Check if the collection name matches the primary's name.
	if m.primaryName != "" && collection == m.primaryName && m.primary != nil {
		t, err := m.primary.Get(id)
		if err != nil {
			return CollectedTask{}, err
		}
		return CollectedTask{Task: t, Collection: m.primaryName}, nil
	}

	// Qualified: find the named collection.
	for _, ns := range m.collections {
		if ns.name == collection {
			t, err := ns.store.Get(id)
			if err != nil {
				return CollectedTask{}, err
			}
			return CollectedTask{Task: t, Collection: ns.name}, nil
		}
	}
	return CollectedTask{}, fmt.Errorf("%w: collection %q not found", ErrNotFound, collection)
}

// CheckDeps examines the dependencies of the task identified by qualifiedID
// and returns the subset that are not yet done, drawing on all collections.
//
// Dependencies that already use the "{collection}:{id}" format are resolved
// against the named collection; bare IDs are resolved against the same
// collection as the parent task.
//
// Returns (blocking, allDone, err).  When allDone is true blocking is empty.
func (m *MultiStore) CheckDeps(qualifiedID string) (blocking []CollectedTask, allDone bool, err error) {
	parentCollection, _, parseErr := splitQualifiedID(qualifiedID)
	if parseErr != nil {
		return nil, false, parseErr
	}
	parent, err := m.Get(qualifiedID)
	if err != nil {
		return nil, false, err
	}

	if len(parent.Dependencies) == 0 {
		return nil, true, nil
	}

	for _, depID := range parent.Dependencies {
		depCollection, rawID, parseErr := splitQualifiedID(depID)
		if parseErr != nil {
			// Malformed dep ID — treat as blocking.
			blocking = append(blocking, CollectedTask{
				Task:       &task.Task{ID: depID, Title: "(malformed dependency ID)", Status: task.StatusBlocked},
				Collection: "",
			})
			continue
		}
		if depCollection == "" {
			// Inherit the parent's collection for unqualified dep IDs.
			depCollection = parentCollection
		}

		var qualDep string
		if depCollection == "" {
			qualDep = rawID
		} else {
			qualDep = depCollection + ":" + rawID
		}

		dep, lookupErr := m.Get(qualDep)
		if lookupErr != nil {
			// Treat a missing dep as blocking (not done).
			blocking = append(blocking, CollectedTask{
				Task:       &task.Task{ID: rawID, Title: "(dependency not found)", Status: task.StatusBlocked},
				Collection: depCollection,
			})
			continue
		}
		if dep.Status != task.StatusDone {
			blocking = append(blocking, dep)
		}
	}

	return blocking, len(blocking) == 0, nil
}

// splitQualifiedID splits a possibly-qualified ID into (collection, id).
// If the ID has no ":" separator, collection is "".
// Returns an error for malformed inputs where either the collection or the id
// part would be empty (e.g. ":1" or "frontend:").
func splitQualifiedID(qualifiedID string) (collection, id string, err error) {
	i := strings.Index(qualifiedID, ":")
	if i < 0 {
		return "", qualifiedID, nil
	}
	collection = qualifiedID[:i]
	id = qualifiedID[i+1:]
	if collection == "" {
		return "", "", fmt.Errorf("malformed qualified ID %q: collection name must not be empty", qualifiedID)
	}
	if id == "" {
		return "", "", fmt.Errorf("malformed qualified ID %q: task ID must not be empty after ':'", qualifiedID)
	}
	return collection, id, nil
}

// CollectionStoreFromConfig creates a Store from a CollectionConfig.  The
// resulting store is wrapped with retry and metrics exactly like a primary
// store created via NewFromConfig.
func CollectionStoreFromConfig(cc CollectionConfig) (*Store, error) {
	cfg := &Config{
		Backend:           cc.Backend,
		Root:              cc.Root,
		TasksFile:         cc.TasksFile,
		DocsDir:           cc.DocsDir,
		DisplayHashLength: cc.DisplayHashLength,
		S3:                cc.S3,
	}
	if cfg.Backend == "" {
		cfg.Backend = BackendLocal
	}
	return NewFromConfig(cfg)
}

// MultiStoreFromConfig constructs a MultiStore from a Config.  The primary
// store is built from the top-level Config fields; any Collections are opened
// as additional named stores.  If cfg.Name is set, the primary store is
// addressable via that name in qualified dependency IDs.
//
// Returns an error if any collection names are duplicated or collide with the
// primary store's name.
func MultiStoreFromConfig(cfg *Config) (*MultiStore, error) {
	// Validate collection name uniqueness before opening any stores.
	seen := make(map[string]bool, len(cfg.Collections)+1)
	if cfg.Name != "" {
		seen[cfg.Name] = true
	}
	for _, cc := range cfg.Collections {
		if seen[cc.Name] {
			if cfg.Name != "" && cc.Name == cfg.Name {
				return nil, fmt.Errorf("collection name %q collides with the primary store name", cc.Name)
			}
			return nil, fmt.Errorf("duplicate collection name %q", cc.Name)
		}
		seen[cc.Name] = true
	}

	primary, err := NewFromConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("opening primary store: %w", err)
	}

	var named []namedStore
	for _, cc := range cfg.Collections {
		s, err := CollectionStoreFromConfig(cc)
		if err != nil {
			return nil, fmt.Errorf("opening collection %q: %w", cc.Name, err)
		}
		named = append(named, namedStore{name: cc.Name, store: s})
	}

	ms := newMultiStore(primary, named)
	ms.primaryName = cfg.Name
	return ms, nil
}
