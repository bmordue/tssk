# CLI Command Flow

## Purpose
This diagram illustrates the sequence of interactions between the user, the Cobra CLI layer, and the internal `Store` when a typical `tssk` command is executed.

## Diagram

```mermaid
sequenceDiagram
    participant User
    participant Cobra as Cobra CLI (cmd/)
    participant Config as openStore()
    participant Store as internal/store
    participant Backend as Backend (local/S3)

    User->>Cobra: tssk <command> [flags] [args]
    Cobra->>Cobra: Parse flags & validate args
    Cobra->>Config: openStore()
    Config->>Config: Read .tssk.json + env vars
    Config->>Config: Build Backend chain
    Config-->>Cobra: Store instance

    alt add command
        Cobra->>Store: s.Add(title, detail, deps, tags)
        Store->>Store: LoadAll(), generate ID, compute DocHash
        Store->>Backend: WriteDetail(hash_prefix, detail)
        Store->>Backend: WriteTasksData(JSONL)
        Store-->>Cobra: *Task
        Cobra-->>User: "Added task N: title"
    else list command
        Cobra->>Store: st.LoadAll()
        Store->>Backend: ReadTasksData()
        Backend-->>Store: raw JSONL bytes
        Store-->>Cobra: []*Task
        Cobra-->>User: Tabular task list
    else show command
        Cobra->>Store: s.Get(id)
        Store->>Backend: ReadTasksData()
        Backend-->>Store: raw JSONL bytes
        Store-->>Cobra: *Task
        Cobra->>Store: s.ReadDetail(t)
        Store->>Backend: ReadDetail(hash_prefix)
        Backend-->>Store: markdown text
        Store-->>Cobra: detail string
        Cobra-->>User: Task metadata + detail
    else status command
        Cobra->>Store: s.UpdateStatus(id, newStatus)
        Store->>Backend: ReadTasksData() + WriteTasksData()
        Store-->>Cobra: *Task
        Cobra-->>User: "Updated task N status to <status>"
    else deps command
        Cobra->>Store: s.AddDep / s.RemoveDep / s.LoadAll
        Store->>Backend: ReadTasksData() + optionally WriteTasksData()
        Store-->>Cobra: result
        Cobra-->>User: Dependency summary
    else tags command
        Cobra->>Store: s.AddTags / s.RemoveTags / s.SetTags
        Store->>Backend: ReadTasksData() + WriteTasksData()
        Store-->>Cobra: *Task
        Cobra-->>User: Tags list
    else init command
        Cobra->>Cobra: Check if .tssk.json exists
        Cobra->>Cobra: Write default config
        Cobra-->>User: "Initialized tssk"
    end
```

## Key Components
- **User**: Developer or automation agent invoking `tssk` from the terminal.
- **Cobra CLI (`cmd/`)**: Parses flags, validates arguments, and routes to the correct `RunE` handler.
- **openStore()**: Reads `.tssk.json` + env vars, builds the Backend chain (metrics → retry → base), returns a Store.
- **Store (`internal/store`)**: High-level persistence manager; every command creates a fresh instance via `openStore()`.
- **Backend**: Pluggable storage interface. `LocalBackend` uses filesystem; `S3Backend` uses S3-compatible object storage. Both are decorated with `RetryBackend` and `MeteredBackend`.

## Notes
- Every command creates a new Store instance (no shared state between invocations).
- Errors at any step are printed to stderr and result in a non-zero exit code via Cobra's `RunE` mechanism.
- The `MultiStore` variant enables cross-project task management with qualified IDs (`{collection}:{id}`).

## Related Diagrams
- [System Overview](../architecture/system-overview.md)
- [Task Creation Flow](../flows/task-creation.md)
- [Error Handling Flow](error-handling.md)
