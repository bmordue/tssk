# CLI Command Flow

## Purpose
This diagram illustrates the sequence of interactions between the user, the Cobra CLI layer, and the internal `Store` when a typical `tssk` command is executed.

## Diagram

```mermaid
sequenceDiagram
    participant User
    participant Cobra as Cobra CLI (cmd/)
    participant Helpers as root_helpers
    participant Store as internal/store
    participant FS as File System

    User->>Cobra: tssk <command> [flags] [args]
    Cobra->>Cobra: Parse flags & validate args
    Cobra->>Helpers: projectRoot()
    Helpers-->>Cobra: root directory path
    Cobra->>Store: store.New(root)
    Store-->>Cobra: Store instance

    alt add command
        Cobra->>Store: s.Add(title, detail, deps)
        Store->>FS: os.MkdirAll(docs/)
        Store->>FS: os.WriteFile(docs/DocHash.md)
        Store->>FS: Atomic write tasks.jsonl
        Store-->>Cobra: *Task
        Cobra-->>User: "Added task T-N: title"
    else list command
        Cobra->>Store: st.LoadAll()
        Store->>FS: Read tasks.jsonl
        FS-->>Store: raw JSONL bytes
        Store-->>Cobra: []*Task
        Cobra-->>User: Tabular task list
    else show command
        Cobra->>Store: s.Get(id)
        Store->>FS: Read tasks.jsonl
        FS-->>Store: raw JSONL bytes
        Store-->>Cobra: *Task
        Cobra->>Store: s.ReadDetail(t)
        Store->>FS: Read docs/DocHash.md
        FS-->>Store: markdown text
        Store-->>Cobra: detail string
        Cobra-->>User: Task metadata + detail
    else status command
        Cobra->>Store: s.UpdateStatus(id, newStatus)
        Store->>FS: Read + rewrite tasks.jsonl
        Store-->>Cobra: *Task
        Cobra-->>User: "Updated T-N status to <status>"
    else deps command
        Cobra->>Store: s.AddDep / s.RemoveDep / s.LoadAll
        Store->>FS: Read + optionally rewrite tasks.jsonl
        Store-->>Cobra: result
        Cobra-->>User: Dependency summary
    end
```

## Key Components
- **User**: Developer or automation agent invoking `tssk` from the terminal.
- **Cobra CLI (`cmd/`)**: Parses flags, validates arguments, and routes to the correct `RunE` handler.
- **root_helpers**: Provides `projectRoot()`, which reads `TSSK_ROOT` or falls back to the current working directory.
- **Store (`internal/store`)**: Stateless persistence layer; every command creates a fresh `Store` instance.
- **File System**: The only storage backend – `tasks.jsonl` and `docs/*.md`.

## Notes
- Every command creates a new `Store` instance (no shared state between invocations).
- Errors at any step are printed to stderr and result in a non-zero exit code via Cobra's `RunE` mechanism.

## Related Diagrams
- [System Overview](../architecture/system-overview.md)
- [Task Creation Flow](../flows/task-creation.md)
- [Error Handling Flow](error-handling.md)
