package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

const defaultConfig = `# tssk configuration
# Storage backend: local or s3
TSSK_STORAGE_BACKEND=local

# Optional project root override for local backend.
# TSSK_ROOT=/path/to/project

# S3 backend settings (used when TSSK_STORAGE_BACKEND=s3).
# TSSK_S3_BUCKET=my-bucket
# TSSK_S3_PREFIX=tssk
# TSSK_S3_REGION=eu-west-1
# TSSK_S3_ENDPOINT=http://localhost:9000
# TSSK_S3_TIMEOUT_SEC=30
`

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Write default configuration file",
	Long:  `Write the default tssk configuration to the config file location if it does not already exist.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		configPath, err := configFilePath()
		if err != nil {
			return fmt.Errorf("resolving config file path: %w", err)
		}

		created, err := writeDefaultConfigIfMissing(configPath)
		if err != nil {
			return err
		}

		if created {
			fmt.Printf("Wrote default config to %s\n", configPath)
			return nil
		}

		fmt.Printf("Config file already exists at %s\n", configPath)
		return nil
	},
}

func configFilePath() (string, error) {
	if p := os.Getenv("TSSK_CONFIG_FILE"); p != "" {
		return p, nil
	}

	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(configDir, "tssk", "config.env"), nil
}

func writeDefaultConfigIfMissing(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return false, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return false, fmt.Errorf("checking config file %s: %w", path, err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return false, fmt.Errorf("creating config directory: %w", err)
	}

	if err := os.WriteFile(path, []byte(defaultConfig), 0o644); err != nil {
		return false, fmt.Errorf("writing config file %s: %w", path, err)
	}

	return true, nil
}