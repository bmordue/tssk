# Task: Add conflict-check CLI subcommand

## Description
Implement `tssk conflict-check <task-a> <task-b>` CLI command.

## Implementation Details

Add a new Cobra subcommand under `cmd/`:

```bash
tssk conflict-check 5 7
# Output: "Tasks 5 and 7 CONFLICT - shared files: README.md"
# or
# Output: "Tasks 5 and 7 can run in parallel (no file conflicts)"
```

### Requirements
- Accept two task IDs as arguments
- Load all tasks from store
- Build conflict graph (or check file lists directly)
- Print clear conflict status with shared files listed
- Return exit code 0 if no conflict, 1 if conflict exists
- Support `--json` flag for machine-readable output

### JSON Output Format
```json
{
  "task_a": "5",
  "task_b": "7",
  "conflicts": true,
  "shared_files": ["README.md", ".golangci.yml"]
}
```

## Files Created/Modified
- `cmd/conflict_check.go` - New CLI subcommand
- `internal/analysis/` - Uses conflict graph or direct file comparison

## Dependencies
- Task 65 (Extend task model with file tracking fields)
- Task 66 (Build inverted index computation function)

## ADR Reference
- docs/ADR/001-task-file-dependency-tracking.md (conflict-check query)
