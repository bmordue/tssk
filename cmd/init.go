package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/bmordue/tssk/internal/store"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Write default config file",
	Long:  `Write a default .tssk.json configuration file in the project root when one does not already exist.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		configPath := filepath.Join(projectRoot(), store.ConfigFile)

		content, err := store.DefaultConfigFileContent()
		if err != nil {
			return err
		}

		f, err := os.OpenFile(configPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
		if err != nil {
			if errors.Is(err, os.ErrExist) {
				fmt.Fprintf(os.Stdout, "Config file already exists at %s\n", configPath)
				return nil
			}
			if fi, statErr := os.Stat(configPath); statErr == nil && fi.Mode().IsDir() {
				return fmt.Errorf("config path %q is a directory, not a regular file", configPath)
			}
			return fmt.Errorf("creating config file %q: %w", configPath, err)
		}
		defer func() { _ = f.Close() }()

		if _, err := f.Write(content); err != nil {
			return fmt.Errorf("writing config file %q: %w", configPath, err)
		}

		if err := f.Close(); err != nil {
			return fmt.Errorf("closing config file %q: %w", configPath, err)
		}

		fmt.Fprintf(os.Stderr, "Wrote default config to %s\n", configPath)
		return nil
	},
}
