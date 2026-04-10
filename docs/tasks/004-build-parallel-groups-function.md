# Task: Build parallel groups computation function

## Description
Create a function that finds maximal independent sets (parallel-safe groups) from the conflict graph.

## Implementation Details

Add to `internal/analysis` package:

```go
func FindParallelGroups(conflictGraph map[string][]string) [][]string
```

Returns a list of groups, where each group is a list of task IDs that can be worked on in parallel (no file conflicts within the group).

### Requirements
- Use greedy algorithm or maximum independent set approximation
- Groups should be sorted by size (largest first)
- Task IDs within each group should be sorted numerically
- Handle edge cases: empty input, single task, all conflicts

### Algorithm Options
- Greedy: Sort tasks by conflict count (ascending), iteratively add non-conflicting tasks
- Or use Bron-Kerbosch algorithm on the complement graph

## Files Created/Modified
- `internal/analysis/parallel_groups.go` - New file with parallel groups finder
- `internal/analysis/parallel_groups_test.go` - Unit tests

## Dependencies
- Task 65 (Extend task model with file tracking fields)
- Task 66 (Build inverted index computation function)
- Task 67 (Build conflict graph computation function)

## ADR Reference
- docs/ADR/001-task-file-dependency-tracking.md (parallel groups query)
