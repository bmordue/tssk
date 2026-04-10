# System Architecture Overview

## Purpose
This diagram shows the high-level architecture of `tssk`, a command-line tool for managing repository tasks. It illustrates the relationships between the CLI layer, the internal business logic, and the pluggable storage backends.

## Diagram

```mermaid
graph TB
    subgraph "User Interface"
        USER([Developer / Agent])
    end

    subgraph "CLI Layer (cmd/)"
        ROOT[root command]
        ADD[add command]
        LIST[list command]
        SHOW[show command]
        STATUS[status command]
        DEPS[deps command]
        TAGS[tags command]
        INIT[init command]
        READY[ready command]
    end

    subgraph "Configuration"
        CONFIG[".tssk.json\n+ env vars"]
    end

    subgraph "Business Logic (internal/)"
        STORE[Store]
        MULTISTORE[MultiStore]
        TASK[Task]
    end

    subgraph "Backend Layer"
        RETRY[RetryBackend\n(exponential backoff)]
        METERED[MeteredBackend\n(metrics collection)]
        LOCAL[LocalBackend\n(filesystem)]
        S3[S3Backend\n(object storage)]
    end

    subgraph "Storage"
        JSONL[(.tsks/tasks.jsonl\nMetadata)]
        DOCS[(.tsks/docs\nMarkdown Detail Files)]
        S3STORE[(S3 Bucket\nobjects/)]
    end

    USER --> ROOT
    ROOT --> ADD
    ROOT --> LIST
    ROOT --> SHOW
    ROOT --> STATUS
    ROOT --> DEPS
    ROOT --> TAGS
    ROOT --> INIT
    ROOT --> READY

    ADD --> STORE
    LIST --> STORE
    LIST --> MULTISTORE
    SHOW --> STORE
    STATUS --> STORE
    DEPS --> STORE
    DEPS --> MULTISTORE
    TAGS --> STORE

    STORE --> CONFIG
    MULTISTORE --> STORE

    STORE --> METERED
    METERED --> RETRY
    RETRY --> LOCAL
    RETRY --> S3

    LOCAL --> JSONL
    LOCAL --> DOCS
    S3 --> S3STORE

    STORE --> TASK
```

## Key Components
- **CLI Layer (`cmd/`)**: Cobra-based commands that parse user input and delegate to the Store. Includes `add`, `list`, `show`, `status`, `deps`, `tags`, `init`, and `ready` commands.
- **Configuration**: `.tssk.json` config file with environment variable overrides (`TSSK_STORAGE_BACKEND`, `TSSK_ROOT`, etc.).
- **Store (`internal/store`)**: High-level persistence manager – handles task CRUD, dependencies, tags, and caching.
- **MultiStore**: Aggregates multiple Stores for cross-project task management with qualified IDs (`{collection}:{id}`).
- **Task (`internal/task`)**: Defines the `Task` struct, `Status` type, and helper methods for dependency/tag management and content-address hashing.
- **Backend Decorators**: `RetryBackend` adds exponential backoff retry; `MeteredBackend` collects operation metrics.
- **LocalBackend**: Filesystem-based storage using atomic temp-file rename for consistency.
- **S3Backend**: S3-compatible object storage for remote/shared task data.
- **tasks.jsonl**: Newline-delimited JSON file containing task metadata (one task per line).
- **docs/**: Directory holding content-addressed markdown files (named by SHA-256 hash prefix).

## Notes
- The tool supports both local filesystem and S3-compatible storage backends.
- `TSSK_ROOT` environment variable overrides the working directory used for storage.
- Backend decorator chain: `MeteredBackend` → `RetryBackend` → `LocalBackend/S3Backend`.
- Each CLI invocation creates a fresh Store instance (no shared state between runs).

## Related Diagrams
- [Module Dependencies](../components/dependencies.md)
- [CLI Command Flow](../sequences/cli-command-flow.md)
- [Task State Machine](../flows/task-states.md)
