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
		backend = NewLocalBackend(cfg.Root)
	case BackendS3:
		backend, err = NewS3Backend(cfg.S3)
		if err != nil {
			return nil, fmt.Errorf("initialising s3 backend: %w", err)
		}
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
	return s, nil
}
