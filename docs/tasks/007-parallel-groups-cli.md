# Task: Add parallel-groups CLI subcommand

## Description
Implement `tssk parallel-groups` CLI command.

## Implementation Details

Add a new Cobra subcommand under `cmd/`:

```bash
tssk parallel-groups
# Output:
# "Parallel Group 1 (15 tasks): 1, 2, 3, 4, 10, 11, 19, 27, 28, 29, 30, 37, 42, 43, 45"
# "Parallel Group 2 (8 tasks): 5, 7, 8, 9, 14, 15, 16, 17"
# ...
```

### Requirements
- Load all tasks from store
- Compute parallel groups using the conflict graph
- Display groups sorted by size (largest first)
- Show task count and IDs for each group
- Optionally filter by status (e.g., `--status todo`)
- Return exit code 0 always
- Support `--json` flag for machine-readable output

### JSON Output Format
```json
{
  "groups": [
    ["1", "2", "3", ...],
    ["5", "7", "8", ...],
    ...
  ]
}
```

### Options
- `--status todo` - Only consider todo-status tasks
- `--max-groups N` - Limit output to top N groups
- `--min-size N` - Only show groups with at least N tasks

## Files Created/Modified
- `cmd/parallel_groups.go` - New CLI subcommand
- `internal/analysis/` - Uses parallel groups computation

## Dependencies
- Task 65 (Extend task model with file tracking fields)
- Task 66 (Build inverted index computation function)
- Task 67 (Build conflict graph computation function)
- Task 68 (Build parallel groups computation function)

## ADR Reference
- docs/ADR/001-task-file-dependency-tracking.md (parallel groups query)
