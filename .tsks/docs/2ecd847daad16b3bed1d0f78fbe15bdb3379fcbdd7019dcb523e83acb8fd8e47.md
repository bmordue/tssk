Add --json flag to tssk list command

## Files to Modify/Create

### Modify
- `cmd/list.go` — Add `--json` flag; when present, output tasks as JSON array instead of formatted text
- `cmd/json.go` — Already has `printJSON()` helper; reuse for list output
- `README.md` — Document the `--json` flag for `tssk list`

### Create
- Unit tests for JSON output in `cmd/list_test.go` (if new file) or extend existing tests
