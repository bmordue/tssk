# Module Dependencies

## Purpose
This diagram shows the dependency relationships between the Go packages that make up `tssk`, including external library dependencies.

## Diagram

```mermaid
graph LR
    subgraph "Entry Point"
        MAIN[main.go]
    end

    subgraph "CLI Layer"
        CMD[cmd\n(root, add, list, show,\nstatus, deps, tags, init, ready)]
    end

    subgraph "Internal Packages"
        STORE[internal/store\n(store, backend, local,\ns3, config, retry,\nmetrics, multistore)]
        TASK[internal/task]
    end

    subgraph "External Libraries"
        COBRA[github.com/spf13/cobra]
        PFLAG[github.com/spf13/pflag]
        MOUSETRAP[github.com/inconshreveable/mousetrap]
        AWS_CFG[github.com/aws/aws-sdk-go-v2/config]
        AWS_S3[github.com/aws/aws-sdk-go-v2/service/s3]
        AWS_CORE[github.com/aws/aws-sdk-go-v2\n(core, credentials, etc.)]
    end

    subgraph "Go Standard Library"
        STDLIB[bufio / bytes / crypto/sha256\nencoding/json / errors\nfmt / io / os / path/filepath\nstrconv / strings / sync/atomic\ntext/tabwriter / time]
    end

    MAIN --> CMD
    CMD --> STORE
    CMD --> TASK
    CMD --> COBRA
    STORE --> TASK
    STORE --> AWS_CFG
    STORE --> AWS_S3
    STORE --> STDLIB
    TASK --> STDLIB
    COBRA --> PFLAG
    COBRA --> MOUSETRAP
    AWS_CFG --> AWS_CORE
    AWS_S3 --> AWS_CORE
```

## Key Components
- **main**: Minimal entry point – calls `cmd.Execute()`.
- **cmd**: All Cobra command definitions (`add`, `list`, `show`, `status`, `deps`, `tags`, `init`). Depends on `store` and `task` for business logic.
- **internal/store**: Persistence layer with multiple files – `store.go` (high-level ops), `backend.go` (interface), `local.go` (filesystem), `s3.go` (S3), `config.go` (configuration), `retry.go` (backoff decorator), `metrics.go` (metrics collector), `multistore.go` (cross-project aggregation).
- **internal/task**: Pure domain logic with no external dependencies beyond the Go standard library.
- **github.com/spf13/cobra**: CLI framework used for command routing and flag parsing.
- **github.com/aws/aws-sdk-go-v2**: AWS SDK v2 for S3-compatible object storage support.

## Notes
- `internal/task` has no dependency on `internal/store`, keeping domain logic cleanly separated.
- Both `pflag` and `mousetrap` are indirect dependencies pulled in by Cobra.
- AWS SDK dependencies are only needed when the S3 backend is used.
- The `internal/store` package uses the decorator pattern: base backends are wrapped with retry and metrics layers.

## Related Diagrams
- [System Overview](../architecture/system-overview.md)
- [Class Diagram](class-diagram.md)
