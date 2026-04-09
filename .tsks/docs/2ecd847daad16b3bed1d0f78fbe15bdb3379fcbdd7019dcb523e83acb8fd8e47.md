Add a boolean `--json` flag to the list command. When set, output all matching tasks as a single valid JSON array instead of the default table format.

**Implementation details:**
- Add `--json` flag (bool, default false) to `cmd/list.go`
- When `--json` is true, serialize the filtered task list as a JSON array using `encoding/json`
- Output must be **only** valid JSON — no headers, footers, or status messages to stdout
- Any errors should still go to stderr
- Must work correctly when combined with `--status` filter
- Empty result should output `[]` (empty array)
- Output should be pipeable to `jq .` with no modifications

**JSON schema per task in array:**
```json
{
  "id": "10",
  "title": "Machine-readable JSON output (--json flag)",
  "status": "todo",
  "created_at": "2026-04-07T11:59:14.075393011Z",
  "doc_hash": "c6f3f7dd...",
  "dependencies": ["5", "6"]
}
```
- Use the existing `Task` struct from `internal/task`
- Use RFC 3339 timestamp format (already used internally)
- `dependencies` field should be an array (empty array if none, not null)
- `doc_hash` should be null if the task has no detail file

**Acceptance criteria:**
- `tssk list --json | jq .` produces valid parsed JSON
- `tssk list --json --status done | jq '.[] | select(.status == "done")'` works correctly
- No trailing whitespace or extra output on stdout
- Unit test verifies JSON validity and structure