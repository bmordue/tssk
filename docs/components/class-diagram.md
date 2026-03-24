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
        +Time CreatedAt
        +String DocHash
        +MetaJSON() []byte
        +ComputeDocHash() error
        +HasDependency(id string) bool
        +AddDependency(id string) bool
        +RemoveDependency(id string) bool
    }

    class Status {
        <<enumeration>>
        todo
        in-progress
        done
        blocked
        +IsValid() bool
    }

    class Store {
        -String root
        +New(root string) Store
        +LoadAll() []Task
        +Get(id string) Task
        +Add(title, detail string, deps []string) Task
        +UpdateStatus(id string, status Status) Task
        +AddDep(id, depID string) error
        +RemoveDep(id, depID string) error
        +ReadDetail(t Task) string
    }

    Task --> Status : has
    Store "1" --> "0..*" Task : persists
```

## Key Components
- **Task**: Central data model representing a single work item. Holds metadata and the SHA-256 `DocHash` that links to its markdown detail file.
- **Status**: Enumeration type constraining the lifecycle of a task (`todo`, `in-progress`, `done`, `blocked`).
- **Store**: Persistence layer responsible for reading and writing task metadata (JSONL) and detail files (Markdown). Created via `store.New(root)`.

## Notes
- `DocHash` is computed from the immutable fields (`ID`, `Title`, `CreatedAt`) using SHA-256, making it a stable content address.
- The `Store` does not cache in memory; every operation re-reads the JSONL file to avoid stale state.
- Task IDs are sequential strings (`T-1`, `T-2`, …) generated at creation time.

## Related Diagrams
- [Module Dependencies](dependencies.md)
- [Task State Machine](../flows/task-states.md)
