# Task: Add integration tests for CLI subcommands

## Description
End-to-end tests for the new CLI commands using a test store.

## Implementation Details

### Test Scenarios

#### conflict-check
- Two tasks with no file overlap → no conflict
- Two tasks with file overlap → conflict detected
- Invalid task IDs → error message
- One or both tasks don't exist → error message
- `--json` flag produces valid JSON output

#### impact
- File tracked by multiple tasks
- File tracked by single task
- File not tracked by any task
- File path with special characters
- `--json` flag produces valid JSON output

#### parallel-groups
- All todo tasks with no conflicts → single large group
- All todo tasks with conflicts → multiple groups
- `--status` flag filters correctly
- `--json` flag produces valid JSON output

### Test Infrastructure
- Create test fixtures with known task configurations
- Use temporary JSONL files for isolation
- Test CLI output parsing

## Files Created/Modified
- `cmd/conflict_check_test.go` - Integration tests for conflict-check
- `cmd/impact_test.go` - Integration tests for impact
- `cmd/parallel_groups_test.go` - Integration tests for parallel-groups
- `internal/testutil/test_store.go` - Test fixtures (if needed)

## Dependencies
- Task 69 (Add conflict-check CLI subcommand)
- Task 70 (Add impact CLI subcommand)
- Task 71 (Add parallel-groups CLI subcommand)
- Task 18 (Integration test suite - existing infrastructure)
