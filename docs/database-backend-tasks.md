# Database Backend Implementation - Task Structure

## Overview
This document outlines the atomic tasks for implementing a database backend as a replacement for the JSONL file storage in tssk.

## Task Dependency Graph

```
64 (Explore JSONL architecture)
  └─ 65 (Choose DB technology)
      └─ 66 (Design DB schema)
          └─ 67 (Implement DB connection)
              ├─ 68 (Implement DB migrations)
              │   ├─ 69 (Implement ReadTasksData)
              │   │   └─ 75 (JSONL to DB migration tool)
              │   └─ 70 (Implement WriteTasksData)
              │
              ├─ 71 (Implement ReadDetail)
              │   └─ 72 (Implement WriteDetail)
              │       └─ 73 (Implement DeleteDetail)
              │           └─ 76 (Write unit tests)
              │               └─ 77 (Write integration tests)
              │                   └─ 79 (Update documentation)
              │                       └─ 80 (Run full test suite)
              │
              ├─ 74 (Add DB config support)
              │   └─ 78 (Update CLI commands)
              │
              └─ 81 (Add health check CLI command)
```

## Detailed Task List

### Phase 1: Research & Design (Tasks 64-66)

| ID | Title | Dependencies | Files to Create/Modify |
|----|-------|-------------|------------------------|
| 64 | Explore and document current JSONL architecture for database backend replacement | - | Research only |
| 65 | Choose database technology and add dependency | 64 | `go.mod` |
| 66 | Design database schema for tasks storage | 65 | `internal/store/schema.sql` or `migrations/` |

### Phase 2: Core Implementation (Tasks 67-74)

| ID | Title | Dependencies | Files to Create/Modify |
|----|-------|-------------|------------------------|
| 67 | Implement database connection and initialization layer | 66 | `internal/store/db.go`, `internal/store/config.go` |
| 68 | Implement database schema migration logic | 67 | `internal/store/db.go`, `internal/store/schema.sql` |
| 69 | Implement ReadTasksData for database backend | 68 | `internal/store/db.go` |
| 70 | Implement WriteTasksData for database backend | 68 | `internal/store/db.go` |
| 71 | Implement ReadDetail for database backend | 67 | `internal/store/db.go` |
| 72 | Implement WriteDetail for database backend | 71 | `internal/store/db.go` |
| 73 | Implement DeleteDetail for database backend | 72 | `internal/store/db.go` |
| 74 | Add database backend configuration support | 67 | `internal/store/config.go` |

### Phase 3: Migration Tool (Task 75)

| ID | Title | Dependencies | Files to Create/Modify |
|----|-------|-------------|------------------------|
| 75 | Implement JSONL to database migration tool | 70 | `cmd/migrate_db.go` |

### Phase 4: Testing (Tasks 76-77)

| ID | Title | Dependencies | Files to Create/Modify |
|----|-------|-------------|------------------------|
| 76 | Write unit tests for DatabaseBackend | 73 | `internal/store/db_test.go` |
| 77 | Write integration tests for database backend with Store | 76 | `internal/store/store_db_test.go` |

### Phase 5: CLI & Documentation (Tasks 78-81)

| ID | Title | Dependencies | Files to Create/Modify |
|----|-------|-------------|------------------------|
| 78 | Update CLI commands to support database backend | 74 | `cmd/*.go` |
| 79 | Update documentation for database backend | 77 | `README.md`, `docs/` |
| 80 | Run full test suite and fix any failures | 79 | Various |
| 81 | Add database backend health check CLI command | 67 | `cmd/health.go` |

## Files Summary

### New Files to Create
- `internal/store/db.go` - DatabaseBackend implementation
- `internal/store/db_test.go` - Unit tests for DatabaseBackend
- `internal/store/schema.sql` - Database schema definition (or `migrations/` directory)
- `cmd/migrate_db.go` - Migration CLI command
- `cmd/health.go` - Health check CLI command (optional, could add to existing command)
- `internal/store/store_db_test.go` - Integration tests

### Existing Files to Modify
- `internal/store/config.go` - Add 'db' backend support, parse DB config
- `cmd/*.go` - All CLI commands may need minor updates for DB backend compatibility
- `README.md` - Document new 'db' backend option
- `docs/` - Update documentation
- `go.mod` - Add database driver dependency

## Key Design Decisions Required

1. **Database Technology**: Choose between SQLite (mattn/go-sqlite3), embedded key-value stores, or other options
2. **Detail File Storage**: Store in database (BLOB/TEXT column) or keep as filesystem files (reuse LocalBackend logic)
3. **Migration Strategy**: How to handle existing JSONL data migration (task 75)
4. **Schema Migration Tool**: Choose migration framework (golang-migrate, custom, etc.)

## Implementation Notes

- The `Backend` interface in `internal/store/backend.go` already defines the methods needed:
  - `ReadTasksData() ([]byte, error)`
  - `WriteTasksData(data []byte) error`
  - `ReadDetail(docHash string) ([]byte, error)`
  - `WriteDetail(docHash string, data []byte) error`
  - `DeleteDetail(docHash string) error`
  - `HealthCheck() error`

- The existing decorator pattern (RetryBackend, MeteredBackend) should work with DatabaseBackend

- MultiStore support should be considered - each collection can have its own backend type

- Maintain backward compatibility with JSONL format during ReadTasksData/WriteTasksData to preserve existing Store logic
