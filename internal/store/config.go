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
	// Name is an optional label for the primary store used when resolving
	// cross-collection dependency IDs.  When non-empty, qualified IDs of the
	// form "{Name}:{id}" are resolved against this (primary) store.
	Name string
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
	// Collections lists additional named task collections to include when
	// operating across multiple projects.  Each entry is opened as an
	// independent Store that participates in a MultiStore.
	Collections []CollectionConfig
}

func applyEnvOverrides(cfg *Config) error {
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
			return fmt.Errorf("invalid %s value %q: must be an integer between 1 and 64", EnvDisplayHashLength, s)
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
				return fmt.Errorf("invalid %s value %q: must be a positive integer", EnvS3TimeoutSec, s)
			}
			cfg.S3.RequestTimeout = time.Duration(secs) * time.Second
		}
		if cfg.S3.RequestTimeout <= 0 {
			cfg.S3.RequestTimeout = 30 * time.Second
		}
	default:
		return fmt.Errorf("unknown storage backend %q (valid values: local, s3)", cfg.Backend)
	}
	return nil
}

// ConfigFromEnv builds a Config by reading environment variables, falling
// back to the provided root directory for the local backend.
func ConfigFromEnv(root string) (*Config, error) {
	cfg := &Config{Root: root}
	if err := applyEnvOverrides(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// CollectionConfig describes a named external task collection to include when
// working across multiple projects.  Each collection is opened as an
// independent Store and its tasks are surfaced via a MultiStore.
type CollectionConfig struct {
	// Name is the display label used to qualify task IDs from this collection
	// (e.g. "frontend" turns task "3" into "frontend:3").  Required.
	Name string
	// Backend selects the storage implementation for this collection.
	// Defaults to the local filesystem backend.
	Backend BackendType
	// Root is the local filesystem root for this collection.
	// May be absolute or relative to the directory containing .tssk.json.
	Root string
	// TasksFile overrides the path to the tasks JSONL file within this
	// collection (relative to Root for local, or as an S3 key).
	TasksFile string
	// DocsDir overrides the directory that holds task detail markdown files.
	DocsDir string
	// DisplayHashLength overrides the hash prefix length for this collection.
	DisplayHashLength int
	// S3 holds S3-specific settings (used when Backend == BackendS3).
	S3 S3Config
}

// fileConfig is the on-disk JSON representation of tssk configuration.
// All fields are optional; absent fields fall back to built-in defaults,
// which may be further overridden by environment variables.
type fileConfig struct {
	Name              string                 `json:"name,omitempty"`
	Backend           string                 `json:"backend,omitempty"`
	TasksFile         string                 `json:"tasks_file,omitempty"`
	DocsDir           string                 `json:"docs_dir,omitempty"`
	DisplayHashLength int                    `json:"display_hash_length,omitempty"`
	S3                *fileS3Config          `json:"s3,omitempty"`
	Collections       []fileCollectionConfig `json:"collections,omitempty"`
}

// fileCollectionConfig is the JSON representation of a single collection entry.
type fileCollectionConfig struct {
	Name              string        `json:"name"`
	Backend           string        `json:"backend,omitempty"`
	Root              string        `json:"root,omitempty"`
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
		DisplayHashLength: defaultDisplayHashLength,
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
		cfg.Name = fileCfg.Name
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
		// Parse collections.
		for i, fc := range fileCfg.Collections {
			if fc.Name == "" {
				return nil, fmt.Errorf("collection at index %d has no name", i)
			}
			collBackend := BackendType(fc.Backend)
			if collBackend == "" {
				collBackend = BackendLocal
			}
			collRoot := fc.Root
			if collRoot != "" && !filepath.IsAbs(collRoot) {
				collRoot = filepath.Join(root, collRoot)
			}
			// For the local backend, root is required so tasks are not
			// inadvertently read/written relative to the process working directory.
			if collBackend == BackendLocal && collRoot == "" {
				return nil, fmt.Errorf("collection %q: \"root\" is required for the local backend", fc.Name)
			}
			cc := CollectionConfig{
				Name:      fc.Name,
				Backend:   collBackend,
				Root:      collRoot,
				TasksFile: fc.TasksFile,
				DocsDir:   fc.DocsDir,
			}
			if fc.DisplayHashLength != 0 {
				if fc.DisplayHashLength < 1 || fc.DisplayHashLength > 64 {
					return nil, fmt.Errorf("collection %q: invalid display_hash_length %d: must be between 1 and 64", fc.Name, fc.DisplayHashLength)
				}
				cc.DisplayHashLength = fc.DisplayHashLength
			}
			if fc.S3 != nil {
				cc.S3.Bucket = fc.S3.Bucket
				cc.S3.Prefix = fc.S3.Prefix
				cc.S3.Endpoint = fc.S3.Endpoint
				cc.S3.Region = fc.S3.Region
				if fc.S3.TimeoutSec > 0 {
					cc.S3.RequestTimeout = time.Duration(fc.S3.TimeoutSec) * time.Second
				}
			}
			cfg.Collections = append(cfg.Collections, cc)
		}
	}

	// Apply env var overrides – env vars always win.
	if err := applyEnvOverrides(cfg); err != nil {
		return nil, err
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
