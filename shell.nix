{ pkgs ? import <nixpkgs> {} }:

pkgs.mkShell {
  buildInputs = [
    # Go toolchain
    pkgs.go
    pkgs.gopls
    pkgs.golangci-lint
    pkgs.gotools

    # Node.js (required)
    pkgs.nodejs_22

    # General development utilities
    pkgs.git
    pkgs.gnumake
    pkgs.curl
    pkgs.jq
  ] ++ (if pkgs.config.allowUnfree or false then [
    pkgs.gemini-cli
    pkgs.claude-code
  ] else []);

  shellHook = ''
    echo "tssk development environment"
    echo "Go version: $(go version)"
    echo "Node.js version: $(node --version)"
    export GOPATH="$HOME/go"
    export PATH="$GOPATH/bin:$PATH"
  '';
}
