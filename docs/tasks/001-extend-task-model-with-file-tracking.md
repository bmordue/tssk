# Task: Extend task model with file tracking fields

## Description
Add structured file fields to the task JSONL records to support file-level dependency tracking.

## Implementation Details

Add two new fields to the task JSON structure:

```json
{
  "id": "5",
  "title": "...",
  "status": "todo",
  "files_modify": [".github/workflows/ci.yml", ".golangci.yml"],
  "files_create": []
}
```

### Requirements
- Add `files_modify` and `files_create` fields to the internal task struct
- Update JSONL serialization/deserialization to handle these fields
- Ensure backward compatibility with existing task records that don't have these fields
- Add validation to ensure file paths are relative to project root
- Both fields should be optional and default to empty arrays

## Files Created/Modified
- `internal/task/task.go` - Add file tracking fields to Task struct
- `internal/store/store.go` - Update serialization logic
- `internal/task/task_test.go` - Add tests for new fields

## Dependencies
None - this is foundational work that other tasks depend on

## ADR Reference
- docs/ADR/001-task-file-dependency-tracking.md (Option 3 decision)
