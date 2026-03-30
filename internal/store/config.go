package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

const (
	// EnvBackend selects the storage backend ("local" or "s3").
	EnvBackend = "TSSK_STORAGE_BACKEND"
	// EnvS3Bucket is the S3 bucket name.
	EnvS3Bucket = "TSSK_S3_BUCKET"
	// EnvS3Prefix is an optional key prefix inside the bucket.
	EnvS3Prefix = "TSSK_S3_PREFIX"
	// EnvS3Endpoint overrides the AWS endpoint (e.g. for MinIO).
	EnvS3Endpoint = "TSSK_S3_ENDPOINT"
	// EnvS3Region sets the AWS region.
	EnvS3Region = "TSSK_S3_REGION"
	// EnvS3TimeoutSec overrides the per-request timeout in seconds.
	EnvS3TimeoutSec = "TSSK_S3_TIMEOUT_SEC"
	// EnvTasksFile overrides the path/filename of the tasks JSONL file
	// (relative to root for the local backend, or an absolute path).
	EnvTasksFile = "TSSK_TASKS_FILE"
	// EnvDocsDir overrides the path to the directory containing task detail
	// files (relative to root for the local backend, or an absolute path).
	EnvDocsDir = "TSSK_DOCS_DIR"
	// EnvDisplayHashLength sets the number of hex characters from the SHA-256
	// hash to use when naming task detail files.  Must be between 1 and 64.
	// Defaults to 9.
	EnvDisplayHashLength = "TSSK_DISPLAY_HASH_LENGTH"

	// ConfigFile is the name of the optional JSON configuration file read
	// from the project root directory.  Environment variables always take
	// precedence over values in this file.
	ConfigFile = ".tssk.json"
)

// BackendType identifies a storage backend implementation.
type BackendType string

const (
	BackendLocal BackendType = "local"
	BackendS3    BackendType = "s3"
)

// Config holds the configuration required to build a Store.
type Config struct {
	// Backend selects the storage implementation.
	Backend BackendType
	// Root is the local filesystem root used by the local backend.
	Root string
	// TasksFile is the path/filename of the tasks JSONL file.  For the local
	// backend this may be relative to Root or an absolute path.  For the S3
	// backend it is used as the object key relative to Prefix.
	// Defaults to "tasks.jsonl" when empty.
	TasksFile string
	// DocsDir is the directory that contains task detail markdown files.  For
	// the local backend this may be relative to Root or an absolute path.
	// For the S3 backend it is used as a key prefix relative to Prefix.
	// Defaults to "docs" when empty.
	DocsDir string
	// DisplayHashLength is the number of hex characters from the SHA-256
	// hash used when naming task detail files.  Valid range: 1–64.  Defaults
	// to 9.  The full 64-character hash is always stored in DocHash in the
	// tasks file; this setting only controls the filename prefix length.
	DisplayHashLength int
	// S3 holds S3-specific configuration (used when Backend == BackendS3).
	S3 S3Config
}

// ConfigFromEnv builds a Config by reading environment variables, falling
// back to the provided root directory for the local backend.
func ConfigFromEnv(root string) (*Config, error) {
	backendStr := os.Getenv(EnvBackend)
	if backendStr == "" {
		backendStr = string(BackendLocal)
	}

	cfg := &Config{
		Backend: BackendType(backendStr),
		Root:    root,
	}

	if s := os.Getenv(EnvTasksFile); s != "" {
		cfg.TasksFile = s
	}
	if s := os.Getenv(EnvDocsDir); s != "" {
		cfg.DocsDir = s
	}
	if s := os.Getenv(EnvDisplayHashLength); s != "" {
		n, err := strconv.Atoi(s)
		if err != nil || n < 1 || n > 64 {
			return nil, fmt.Errorf("invalid %s value %q: must be an integer between 1 and 64", EnvDisplayHashLength, s)
		}
		cfg.DisplayHashLength = n
	}

	switch cfg.Backend {
	case BackendLocal:
		// Nothing extra to parse.
	case BackendS3:
		timeout := 30 * time.Second
		if s := os.Getenv(EnvS3TimeoutSec); s != "" {
			secs, err := strconv.Atoi(s)
			if err != nil || secs <= 0 {
				return nil, fmt.Errorf("invalid %s value %q: must be a positive integer", EnvS3TimeoutSec, s)
			}
			timeout = time.Duration(secs) * time.Second
		}
		cfg.S3 = S3Config{
			Bucket:         os.Getenv(EnvS3Bucket),
			Prefix:         os.Getenv(EnvS3Prefix),
			Endpoint:       os.Getenv(EnvS3Endpoint),
			Region:         os.Getenv(EnvS3Region),
			RequestTimeout: timeout,
		}
	default:
		return nil, fmt.Errorf("unknown storage backend %q (valid values: local, s3)", cfg.Backend)
	}

	return cfg, nil
}

// fileConfig is the on-disk JSON representation of tssk configuration.
// All fields are optional; absent fields fall back to built-in defaults,
// which may be further overridden by environment variables.
type fileConfig struct {
	Backend           string        `json:"backend,omitempty"`
	TasksFile         string        `json:"tasks_file,omitempty"`
	DocsDir           string        `json:"docs_dir,omitempty"`
	DisplayHashLength int           `json:"display_hash_length,omitempty"`
	S3                *fileS3Config `json:"s3,omitempty"`
}

// fileS3Config is the S3 sub-section of the JSON config file.
type fileS3Config struct {
	Bucket     string `json:"bucket,omitempty"`
	Prefix     string `json:"prefix,omitempty"`
	Endpoint   string `json:"endpoint,omitempty"`
	Region     string `json:"region,omitempty"`
	TimeoutSec int    `json:"timeout_sec,omitempty"`
}

// DefaultConfigFileContent returns the default JSON content for .tssk.json.
func DefaultConfigFileContent() ([]byte, error) {
	fc := fileConfig{
		Backend:           string(BackendLocal),
		TasksFile:         defaultTasksFile,
		DocsDir:           defaultDocsDir,
		DisplayHashLength: DefaultDisplayHashLength,
	}
	b, err := json.MarshalIndent(fc, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshalling default config: %w", err)
	}
	b = append(b, '\n')
	return b, nil
}

// loadFileConfig reads and parses the JSON config file at path.
// Returns (nil, nil) when the file does not exist.
func loadFileConfig(path string) (*fileConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading config file %q: %w", path, err)
	}
	var fc fileConfig
	if err := json.Unmarshal(data, &fc); err != nil {
		return nil, fmt.Errorf("parsing config file %q (check for invalid JSON syntax or unexpected field types): %w", path, err)
	}
	return &fc, nil
}

// ConfigFromFileAndEnv builds a Config by first reading the optional JSON
// config file at {root}/.tssk.json and then overlaying environment-variable
// overrides.  Environment variables always take precedence over file values.
func ConfigFromFileAndEnv(root string) (*Config, error) {
	fileCfg, err := loadFileConfig(filepath.Join(root, ConfigFile))
	if err != nil {
		return nil, err
	}

	// Seed defaults from the config file (if present).
	cfg := &Config{Root: root}
	if fileCfg != nil {
		cfg.Backend = BackendType(fileCfg.Backend)
		cfg.TasksFile = fileCfg.TasksFile
		cfg.DocsDir = fileCfg.DocsDir
		if fileCfg.DisplayHashLength != 0 {
			if fileCfg.DisplayHashLength < 1 || fileCfg.DisplayHashLength > 64 {
				return nil, fmt.Errorf("invalid display_hash_length %d in config file: must be between 1 and 64", fileCfg.DisplayHashLength)
			}
			cfg.DisplayHashLength = fileCfg.DisplayHashLength
		}
		if fileCfg.S3 != nil {
			cfg.S3.Bucket = fileCfg.S3.Bucket
			cfg.S3.Prefix = fileCfg.S3.Prefix
			cfg.S3.Endpoint = fileCfg.S3.Endpoint
			cfg.S3.Region = fileCfg.S3.Region
			if fileCfg.S3.TimeoutSec > 0 {
				cfg.S3.RequestTimeout = time.Duration(fileCfg.S3.TimeoutSec) * time.Second
			}
		}
	}

	// Apply env var overrides – env vars always win.
	if s := os.Getenv(EnvBackend); s != "" {
		cfg.Backend = BackendType(s)
	}
	if cfg.Backend == "" {
		cfg.Backend = BackendLocal
	}

	if s := os.Getenv(EnvTasksFile); s != "" {
		cfg.TasksFile = s
	}
	if s := os.Getenv(EnvDocsDir); s != "" {
		cfg.DocsDir = s
	}
	if s := os.Getenv(EnvDisplayHashLength); s != "" {
		n, err := strconv.Atoi(s)
		if err != nil || n < 1 || n > 64 {
			return nil, fmt.Errorf("invalid %s value %q: must be an integer between 1 and 64", EnvDisplayHashLength, s)
		}
		cfg.DisplayHashLength = n
	}

	switch cfg.Backend {
	case BackendLocal:
		// Nothing extra to parse.
	case BackendS3:
		if s := os.Getenv(EnvS3Bucket); s != "" {
			cfg.S3.Bucket = s
		}
		if s := os.Getenv(EnvS3Prefix); s != "" {
			cfg.S3.Prefix = s
		}
		if s := os.Getenv(EnvS3Endpoint); s != "" {
			cfg.S3.Endpoint = s
		}
		if s := os.Getenv(EnvS3Region); s != "" {
			cfg.S3.Region = s
		}
		if s := os.Getenv(EnvS3TimeoutSec); s != "" {
			secs, err := strconv.Atoi(s)
			if err != nil || secs <= 0 {
				return nil, fmt.Errorf("invalid %s value %q: must be a positive integer", EnvS3TimeoutSec, s)
			}
			cfg.S3.RequestTimeout = time.Duration(secs) * time.Second
		}
		if cfg.S3.RequestTimeout <= 0 {
			cfg.S3.RequestTimeout = 30 * time.Second
		}
	default:
		return nil, fmt.Errorf("unknown storage backend %q (valid values: local, s3)", cfg.Backend)
	}

	return cfg, nil
}

// NewFromConfig creates a Store wired to the backend described by cfg.
// The backend is wrapped with connection-level retry (3 attempts, exponential
// backoff) and a metrics collector.
func NewFromConfig(cfg *Config) (*Store, error) {
	var backend Backend
	var err error

	switch cfg.Backend {
	case BackendLocal, "":
		lb := NewLocalBackend(cfg.Root)
		if cfg.TasksFile != "" {
			lb.tasksFile = cfg.TasksFile
		}
		if cfg.DocsDir != "" {
			lb.docsDir = cfg.DocsDir
		}
		backend = lb
	case BackendS3:
		var sb *S3Backend
		sb, err = NewS3Backend(cfg.S3)
		if err != nil {
			return nil, fmt.Errorf("initialising s3 backend: %w", err)
		}
		if cfg.TasksFile != "" {
			sb.tasksFile = cfg.TasksFile
		}
		if cfg.DocsDir != "" {
			sb.docsDir = cfg.DocsDir
		}
		backend = sb
	default:
		return nil, fmt.Errorf("unknown storage backend %q", cfg.Backend)
	}

	// Wrap with retry and metrics.
	retryCfg := DefaultRetryConfig()
	backend = NewRetryBackend(backend, retryCfg)
	m := &Metrics{}
	backend = NewMeteredBackend(backend, m)

	s := NewWithBackend(backend)
	s.metrics = m
	if cfg.DisplayHashLength > 0 {
		s.displayHashLength = cfg.DisplayHashLength
	}
	return s, nil
}
