# Task: Update documentation with task-file dependency tracking

## Description
Update README and add CLI help text for the new commands.

## Implementation Details

### README Updates
Add a section explaining the file dependency tracking feature:
- Overview of the three new commands
- Example usage scenarios
- How to interpret the output
- Link to the ADR for design decisions

### CLI Help Text
Ensure all three new commands have comprehensive `--help` output:
- `tssk conflict-check --help`
- `tssk impact --help`
- `tssk parallel-groups --help`

### ADR Updates
- Link back from ADR-001 to the implementation once complete
- Update ADR status to "Accepted" if the approach works well

### Files Created/Modified
- `README.md` - Add feature documentation section
- `cmd/conflict_check.go` - Help text
- `cmd/impact.go` - Help text
- `cmd/parallel_groups.go` - Help text
- `docs/README.md` - Update if separate docs site exists

## Dependencies
- Task 69 (Add conflict-check CLI subcommand)
- Task 70 (Add impact CLI subcommand)
- Task 71 (Add parallel-groups CLI subcommand)

## ADR Reference
- docs/ADR/001-task-file-dependency-tracking.md
