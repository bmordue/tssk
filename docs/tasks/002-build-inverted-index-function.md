# Task: Build inverted index computation function

## Description
Create a helper function that reads all tasks and builds the inverted index (file → [task_ids]).

## Implementation Details

Create a function in a new package `internal/analysis` that:
- Reads all tasks from the JSONL store
- Builds the inverted index: `map[string][]string` where key is file path, value is list of task IDs
- Returns the inverted index as a map

### Function Signature
```go
func BuildInvertedIndex(tasks []task.Task) map[string][]string
```

### Requirements
- Efficiently process all tasks
- Handle both `files_modify` and `files_create` entries
- Sort task IDs numerically for consistent output
- Should be pure function (no side effects, easily testable)

## Files Created/Modified
- `internal/analysis/inverted_index.go` - New file with inverted index builder
- `internal/analysis/inverted_index_test.go` - Unit tests

## Dependencies
- Task 65 (Extend task model with file tracking fields)

## ADR Reference
- docs/ADR/001-task-file-dependency-tracking.md (Option 3: derive inverted index on demand)
