# Task: Add impact CLI subcommand

## Description
Implement `tssk impact <file-path>` CLI command.

## Implementation Details

Add a new Cobra subcommand under `cmd/`:

```bash
tssk impact README.md
# Output:
# "File README.md is touched by tasks: 7, 8, 9, 14, 15, 16..."
```

### Requirements
- Accept a file path as argument
- Build inverted index from current tasks
- Look up the file in the inverted index
- Print list of tasks that touch the file
- Show whether each task will modify or create the file
- Return exit code 0 always (informational command)
- Support `--json` flag for machine-readable output

### JSON Output Format
```json
{
  "file": "README.md",
  "tasks": [
    {"id": "7", "action": "create"},
    {"id": "8", "action": "modify"},
    ...
  ]
}
```

### Edge Cases
- File not found in any task: print "File not tracked by any tasks"
- Support glob patterns? (e.g., `tssk impact "*.go"`) - Phase 2 feature

## Files Created/Modified
- `cmd/impact.go` - New CLI subcommand
- `internal/analysis/` - Uses inverted index

## Dependencies
- Task 65 (Extend task model with file tracking fields)
- Task 66 (Build inverted index computation function)

## ADR Reference
- docs/ADR/001-task-file-dependency-tracking.md (impact analysis query)
