# Task: Build conflict graph computation function

## Description
Create a function that computes the pairwise conflict graph from the inverted index.

## Implementation Details

Add to `internal/analysis` package:

```go
func BuildConflictGraph(invertedIndex map[string][]string) map[string][]string
```

Returns a map where each key is a task ID and the value is a list of task IDs that conflict with it (share at least one file).

### Requirements
- Efficiently compute pairwise conflicts
- Each task should only list each conflict once (no duplicates)
- Task ID should not appear in its own conflict list
- Sort conflict lists numerically for consistency

### Algorithm
1. Iterate through each file in the inverted index
2. For each file with multiple tasks, mark all pairs as conflicting
3. Aggregate all conflicts per task

## Files Created/Modified
- `internal/analysis/conflict_graph.go` - New file with conflict graph builder
- `internal/analysis/conflict_graph_test.go` - Unit tests

## Dependencies
- Task 65 (Extend task model with file tracking fields)
- Task 66 (Build inverted index computation function)

## ADR Reference
- docs/ADR/001-task-file-dependency-tracking.md (conflict graph as derived artifact)
