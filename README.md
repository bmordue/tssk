# tssk
command line tool for managing repository tasks for humans and agents

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
