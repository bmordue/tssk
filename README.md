# tssk

A command-line tool for managing repository tasks for humans and AI agents.

## Overview

tssk provides a lightweight task tracking system that lives alongside your code. Tasks are stored as JSONL metadata and markdown detail files directly in the repository, making them accessible to both humans and automation tools.

## Tech Stack

- Language: Go 1.24
- CLI framework: [Cobra](https://github.com/spf13/cobra)

## Installation

Build the binary from source:

```bash
go build -o build/tssk .
```

Then move the binary to a directory on your PATH, for example:

```bash
mv tssk /usr/local/bin/
```

## Usage

### Initialize config

```bash
tssk init
```

Writes a default `.tssk.json` in the project root when it does not exist.
If the file already exists, `tssk` prints a message and leaves it unchanged.

Default `.tssk.json` content:

```json
{
  "backend": "local",
  "tasks_file": "tasks.jsonl",
  "docs_dir": "docs",
  "hash_length": 64
}
```

### Add a task

```bash
tssk add --title "Implement feature X"
tssk add --title "Implement feature X" --detail "Detailed markdown description"
tssk add --title "Implement feature X" --detail "..." --deps T-1,T-2
```

Flags:

- `-t, --title` (required) - Task title
- `-d, --detail` - Detail text, written to a markdown file
- `-D, --deps` - Comma-separated list of dependency task IDs

### List tasks

```bash
tssk list
tssk list --status todo
tssk list --status in-progress
tssk list --status done
tssk list --status blocked
```

Flags:

- `-s, --status` - Filter by status (`todo`, `in-progress`, `done`, `blocked`)

### Show a task

```bash
tssk show T-1
```

Displays the full task metadata and the content of its detail file.

### Update task status

```bash
tssk status T-1 in-progress
tssk status T-1 done
```

Valid status values: `todo`, `in-progress`, `done`, `blocked`

### Manage dependencies

```bash
# Add a dependency (T-2 depends on T-1)
tssk deps add T-2 T-1

# Remove a dependency
tssk deps remove T-2 T-1

# Check whether all dependencies of a task are done
tssk deps check T-2
```

### Visualize tasks in a web browser

```bash
tssk serve
tssk serve --port 3000
tssk serve --open
```

Flags:

- `-p, --port` - Port to listen on (default: 8080)
- `--host` - Host to listen on (default: localhost)
- `-o, --open` - Open browser automatically

The web UI provides:
- Interactive task list with status badges and tags
- Filter by status, tag, or search by title
- Sortable columns (ID, status, title, created date)
- Expandable task cards with full markdown detail
- Status updates directly from the UI
- Dependency visualization
- Dark/light theme toggle

## Storage

Tasks are stored relative to the project root:

- `.tsks/tasks.jsonl` - Task metadata in JSONL format, one record per line
- `.tsks/docs/` - Markdown detail files, one per task that has detail text

> **Note:** tssk does not migrate existing data when storage settings change. If you alter the tasks file path, docs directory, or backend after creating tasks, tssk will treat the new location as empty and your existing tasks will remain at the old location — invisible to tssk until you manually move them. Detail files are named using the first N characters of the SHA-256 hash of the task metadata, where N is controlled by `display_hash_length` (defaults to 9). This also has upgrade and migration implications: repositories created with earlier versions may have detail files named with the full 64-character SHA-256 hash, and changing `display_hash_length` after tasks already exist will change the expected detail filenames, making existing detail files appear missing until you manually rename or copy them. For existing repositories, keep `display_hash_length=64` unless you also migrate the existing detail filenames to the new length.

## Configuration

By default, tssk uses the current working directory as the project root. Set the `TSSK_ROOT` environment variable to override this:

```bash
TSSK_ROOT=/path/to/project tssk list
```

You can create a default config file with:

```bash
tssk init
```

## Development

### Managing Tasks with tssk

This project uses tssk for task tracking and development workflow. All development tasks are tracked in the repository using tssk itself.

### Running Tests

To run all tests in the project:

```bash
go test ./...
```

### Formatting Code

To format the code according to Go standards:

```bash
go fmt ./...
```

## Development Environment

### Using Nix

A reproducible development environment is available via [Nix](https://nixos.org/).

**Enter the development shell:**

```sh
nix-shell
```

This provides:
- Go toolchain (`go`, `gopls`, `golangci-lint`, `gotools`)
- Node.js 22 (`node`, `npm`)
- General utilities (`git`, `make`, `curl`, `jq`)

**Unfree packages** (requires `allowUnfree = true` in your Nix config):
- `gemini-cli`
- `claude-code`

To enable unfree packages, add the following to your `~/.config/nixpkgs/config.nix`:

```nix
{ allowUnfree = true; }
```

### Using direnv

If you have [direnv](https://direnv.net/) installed, the environment is activated automatically when you enter the project directory:

```sh
direnv allow
```
