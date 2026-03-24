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
go build -o tssk .
```

Then move the binary to a directory on your PATH, for example:

```bash
mv tssk /usr/local/bin/
```

## Usage

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

## Storage

Tasks are stored relative to the project root:

- `tasks.jsonl` - Task metadata in JSONL format, one record per line
- `docs/` - Markdown detail files, one per task that has detail text

## Configuration

By default, tssk uses the current working directory as the project root. Set the `TSSK_ROOT` environment variable to override this:

```bash
TSSK_ROOT=/path/to/project tssk list
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
