# Project: tssk

## Agent Instructions

### Task Management

**Always use the `tssk` CLI tool for any task-related queries.** Do not search the codebase or test files for task status.

Commands:
- `tssk list` - List all tasks
- `tssk show <id>` - Show details of a specific task
- `tssk add <title>` - Add a new task
- `tssk done <id>` - Mark a task as done
- `tssk edit <id>` - Edit a task

For JSON output (scripting/agent consumption), use `--json` flag:
- `tssk list --json`
- `tssk show <id> --json`
