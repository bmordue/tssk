# Class / Type Diagram

## Purpose
This diagram shows the Go types defined in the `internal/task` and `internal/store` packages, their fields, and the relationships between them.

## Diagram

```mermaid
classDiagram
    class Task {
        +String ID
        +String Title
        +Status Status
        +[]String Dependencies
        +[]String Tags
        +Time CreatedAt
        +String DocHash
        +MetaJSON() ([]byte, error)
        +ComputeDocHash() error
        +ComputeDocHashN(length int) error
        +HasDependency(id string) bool
        +AddDependency(id string) bool
        +RemoveDependency(id string) bool
        +HasTag(tag string) bool
        +AddTag(tag string) bool
        +RemoveTag(tag string) bool
    }

    class Status {
        <<enumeration>>
        todo
        in-progress
        done
        blocked
        +IsValid() bool
    }

    class Backend {
        <<interface>>
        +ReadTasksData() ([]byte, error)
        +WriteTasksData(data []byte) error
        +ReadDetail(docHash string) ([]byte, error)
        +WriteDetail(docHash string, data []byte) error
        +DeleteDetail(docHash string) error
        +HealthCheck() error
    }

    class LocalBackend {
        -String root
        -String tasksFile
        -String docsDir
        +NewLocalBackend(root string) *LocalBackend
    }

    class S3Backend {
        -S3Config config
        -s3API client
        -String tasksFile
        -String docsDir
        +NewS3Backend(cfg S3Config) (*S3Backend, error)
    }

    class RetryBackend {
        -Backend wrapped
        -RetryConfig retryCfg
        +NewRetryBackend(b Backend, cfg RetryConfig) *RetryBackend
    }

    class MeteredBackend {
        -Backend wrapped
        -Metrics metrics
        +NewMeteredBackend(b Backend, m *Metrics) *MeteredBackend
    }

    class Store {
        -Backend backend
        -Metrics metrics
        -int displayHashLength
        -[]*Task cache
        +New(root string) *Store
        +NewWithBackend(b Backend) *Store
        +HealthCheck() error
        +Metrics() *Metrics
        +LoadAll() ([]*Task, error)
        +Get(id string) (*Task, error)
        +Add(title, detail string, deps, tags []string) (*Task, error)
        +UpdateStatus(id string, status Status) (*Task, error)
        +AddDep(id, depID string) error
        +RemoveDep(id, depID string) error
        +AddTags(id string, tags []string) error
        +RemoveTags(id string, tags []string) error
        +SetTags(id string, tags []string) error
        +ReadDetail(t *Task) (string, error)
    }

    class MultiStore {
        -Store primary
        -map[string]*Store collections
        +NewMultiStoreWithCollections(primary *Store, collections []NamedStore) *MultiStore
        +LoadAll() ([]CollectedTask, error)
        +Get(qualifiedID string) (CollectedTask, error)
        +CheckDeps(qualifiedID string) (blocking []CollectedTask, allDone bool, err error)
    }

    class Config {
        +String Name
        +Backend BackendType
        +String Root
        +String TasksFile
        +String DocsDir
        +int DisplayHashLength
        +S3Config S3
        +[]CollectionConfig Collections
    }

    Task --> Status : has
    Backend <|.. LocalBackend : implements
    Backend <|.. S3Backend : implements
    Backend <|.. RetryBackend : decorates
    Backend <|.. MeteredBackend : decorates
    RetryBackend --> Backend : wraps
    MeteredBackend --> Backend : wraps
    Store --> Backend : uses
    Store --> Task : persists
    MultiStore --> Store : aggregates
    Config --> S3Config : contains
    Config --> CollectionConfig : contains
```

## Key Components
- **Task**: Central data model representing a single work item. Holds metadata including tags and the SHA-256 `DocHash` that links to its markdown detail file.
- **Status**: Enumeration type constraining the lifecycle of a task (`todo`, `in-progress`, `done`, `blocked`).
- **Backend**: Interface defining low-level storage operations. Implemented by `LocalBackend` (filesystem) and `S3Backend` (S3-compatible object storage).
- **RetryBackend**: Decorator that wraps any Backend with exponential backoff retry logic (3 attempts, configurable).
- **MeteredBackend**: Decorator that collects metrics (operation counts, errors, timing) on all backend calls.
- **Store**: High-level persistence manager. Created via `New(root)` or `NewWithBackend(backend)`. Manages task CRUD, dependencies, tags, and caching.
- **MultiStore**: Aggregates a primary Store with named collection Stores for cross-project task management using qualified IDs (`{collection}:{id}`).
- **Config**: Configuration loaded from `.tssk.json` with environment variable overrides.

## Notes
- `DocHash` is computed from the immutable fields (`ID`, `Title`, `CreatedAt`) using SHA-256, making it a stable content address.
- The Store caches loaded tasks in memory after the first `LoadAll()` call.
- Task IDs are sequential integers as strings (`"1"`, `"2"`, …) generated at creation time.
- The backend decorator chain is: `MeteredBackend` → `RetryBackend` → `LocalBackend/S3Backend`.
- `displayHashLength` (default 9) controls the filename prefix length for detail markdown files; the full 64-char hash is stored in `DocHash`.

## Related Diagrams
- [Module Dependencies](dependencies.md)
- [Task State Machine](../flows/task-states.md)
