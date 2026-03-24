package cmd

import (
	"fmt"
	"os"

	"github.com/bmordue/tssk/internal/store"
)

// projectRoot returns the directory that tssk uses as its working root.
// By default this is the current working directory.  Set the TSSK_ROOT
// environment variable to override (useful for testing).
func projectRoot() string {
	if r := os.Getenv("TSSK_ROOT"); r != "" {
		return r
	}
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return wd
}

// openStore creates a Store configured from environment variables.
// The storage backend is selected via TSSK_STORAGE_BACKEND (default: "local").
func openStore() (*store.Store, error) {
	cfg, err := store.ConfigFromEnv(projectRoot())
	if err != nil {
		return nil, fmt.Errorf("storage configuration: %w", err)
	}
	s, err := store.NewFromConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("opening store: %w", err)
	}
	return s, nil
}
