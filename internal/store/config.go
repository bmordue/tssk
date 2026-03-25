package store

import (
	"fmt"
	"os"
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
	// EnvHashLength sets the number of hex characters to use from the
	// SHA-256 hash when naming task detail files.  Must be between 1 and 64.
	// Defaults to 64 (full hash).
	EnvHashLength = "TSSK_HASH_LENGTH"
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
	// HashLength is the number of hex characters taken from the SHA-256 hash
	// when naming task detail files.  Valid range: 1–64.  Defaults to 64.
	HashLength int
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
	if s := os.Getenv(EnvHashLength); s != "" {
		n, err := strconv.Atoi(s)
		if err != nil || n < 1 || n > 64 {
			return nil, fmt.Errorf("invalid %s value %q: must be an integer between 1 and 64", EnvHashLength, s)
		}
		cfg.HashLength = n
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
	if cfg.HashLength > 0 {
		s.hashLength = cfg.HashLength
	}
	return s, nil
}
