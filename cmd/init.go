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

		if _, err := os.Stat(configPath); err == nil {
			fmt.Fprintf(os.Stdout, "Config file already exists at %s\n", configPath)
			return nil
		} else if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("checking config file %q: %w", configPath, err)
		}

		content, err := store.DefaultConfigFileContent()
		if err != nil {
			return err
		}
		if err := os.WriteFile(configPath, content, 0o644); err != nil {
			return fmt.Errorf("writing config file %q: %w", configPath, err)
		}

		fmt.Fprintf(os.Stdout, "Wrote default config to %s\n", configPath)
		return nil
	},
}
