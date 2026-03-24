# Task Creation Flow

## Purpose
This diagram illustrates the end-to-end flow when a user runs `tssk add --title "..." --detail "..." --deps T-1,T-2` to create a new task.

## Diagram

```mermaid
flowchart TD
    START([User runs: tssk add])
    ARGS[Parse --title, --detail, --deps flags]
    VALIDATE{--title provided\nand non-empty?}
    ERROR([Return error:\ntitle is required])
    LOAD[Load existing tasks\nfrom tasks.jsonl]
    BUILD[Build new Task struct\nID = T-N, Status = todo\nCreatedAt = now UTC]
    HASH[Compute DocHash\nSHA-256 of id+title+created_at]
    ENSURE[Ensure docs/ directory exists]
    WRITE_DOC[Write detail markdown file\ndocs/DocHash.md]
    APPEND[Append task to in-memory list]
    SAVE[Atomically save tasks.jsonl\nvia temp-file rename]
    CLEANUP{Save succeeded?}
    REMOVE[Remove orphaned\ndetail file]
    OUTPUT([Print: Added task T-N: title])

    START --> ARGS
    ARGS --> VALIDATE
    VALIDATE -->|No| ERROR
    VALIDATE -->|Yes| LOAD
    LOAD --> BUILD
    BUILD --> HASH
    HASH --> ENSURE
    ENSURE --> WRITE_DOC
    WRITE_DOC --> APPEND
    APPEND --> SAVE
    SAVE --> CLEANUP
    CLEANUP -->|Failed| REMOVE
    CLEANUP -->|Succeeded| OUTPUT
    REMOVE --> ERROR
```

## Key Components
- **Flag parsing**: Cobra validates that `--title` is present and non-empty before delegating to the Store.
- **DocHash**: SHA-256 of the JSON-encoded immutable metadata (`id`, `title`, `created_at`) – serves as a stable content address for the detail file.
- **Atomic save**: The JSONL file is replaced atomically using a temp-file rename to prevent partial writes.
- **Orphan cleanup**: If the JSONL save fails after the detail file is written, the orphaned detail file is removed as a best-effort cleanup.

## Notes
- Task IDs are sequential (`T-1`, `T-2`, …) based on the current count of tasks.
- Dependencies (`--deps`) are stored as a list of task ID strings; they are not validated against existing tasks at creation time.

## Related Diagrams
- [Task State Machine](task-states.md)
- [CLI Command Flow](../sequences/cli-command-flow.md)
- [Data Persistence Pipeline](data-pipeline.md)
