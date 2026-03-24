# Module Dependencies

## Purpose
This diagram shows the dependency relationships between the Go packages that make up `tssk`, including the external library dependency on Cobra.

## Diagram

```mermaid
graph LR
    subgraph "Entry Point"
        MAIN[main]
    end

    subgraph "CLI Layer"
        CMD[cmd]
    end

    subgraph "Internal Packages"
        STORE[internal/store]
        TASK[internal/task]
    end

    subgraph "External Libraries"
        COBRA[github.com/spf13/cobra]
        PFLAG[github.com/spf13/pflag]
        MOUSETRAP[github.com/inconshreveable/mousetrap]
    end

    subgraph "Go Standard Library"
        STDLIB[bufio / crypto/sha256\nencoding/json / errors\nfmt / os / path/filepath\nstrings / text/tabwriter / time]
    end

    MAIN --> CMD
    CMD --> STORE
    CMD --> TASK
    CMD --> COBRA
    STORE --> TASK
    STORE --> STDLIB
    TASK --> STDLIB
    COBRA --> PFLAG
    COBRA --> MOUSETRAP
```

## Key Components
- **main**: Minimal entry point – calls `cmd.Execute()`.
- **cmd**: All Cobra command definitions (`add`, `list`, `show`, `status`, `deps`). Depends on `store` and `task` for business logic.
- **internal/store**: Persistence layer; depends on `internal/task` for the `Task` type.
- **internal/task**: Pure domain logic with no external dependencies beyond the Go standard library.
- **github.com/spf13/cobra**: CLI framework used for command routing and flag parsing.

## Notes
- `internal/task` has no dependency on `internal/store`, keeping domain logic cleanly separated.
- Both `pflag` and `mousetrap` are indirect dependencies pulled in by Cobra.

## Related Diagrams
- [System Overview](../architecture/system-overview.md)
- [Class Diagram](class-diagram.md)
