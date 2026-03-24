package store_test

import (
	"testing"

	"github.com/bmordue/tssk/internal/store"
)

func TestConfigFromEnv_DefaultsToLocal(t *testing.T) {
	t.Setenv(store.EnvBackend, "")
	cfg, err := store.ConfigFromEnv("/some/root")
	if err != nil {
		t.Fatalf("ConfigFromEnv: %v", err)
	}
	if cfg.Backend != store.BackendLocal {
		t.Errorf("expected local backend, got %q", cfg.Backend)
	}
	if cfg.Root != "/some/root" {
		t.Errorf("unexpected root: %q", cfg.Root)
	}
}

func TestConfigFromEnv_S3Backend(t *testing.T) {
	t.Setenv(store.EnvBackend, "s3")
	t.Setenv(store.EnvS3Bucket, "my-bucket")
	t.Setenv(store.EnvS3Prefix, "tssk")
	t.Setenv(store.EnvS3Region, "eu-west-1")
	t.Setenv(store.EnvS3Endpoint, "http://localhost:9000")

	cfg, err := store.ConfigFromEnv("/root")
	if err != nil {
		t.Fatalf("ConfigFromEnv: %v", err)
	}
	if cfg.Backend != store.BackendS3 {
		t.Errorf("expected s3 backend, got %q", cfg.Backend)
	}
	if cfg.S3.Bucket != "my-bucket" {
		t.Errorf("unexpected bucket: %q", cfg.S3.Bucket)
	}
	if cfg.S3.Prefix != "tssk" {
		t.Errorf("unexpected prefix: %q", cfg.S3.Prefix)
	}
	if cfg.S3.Region != "eu-west-1" {
		t.Errorf("unexpected region: %q", cfg.S3.Region)
	}
	if cfg.S3.Endpoint != "http://localhost:9000" {
		t.Errorf("unexpected endpoint: %q", cfg.S3.Endpoint)
	}
}

func TestConfigFromEnv_UnknownBackend(t *testing.T) {
	t.Setenv(store.EnvBackend, "redis")
	_, err := store.ConfigFromEnv("/root")
	if err == nil {
		t.Fatal("expected error for unknown backend")
	}
}

func TestConfigFromEnv_InvalidTimeout(t *testing.T) {
	t.Setenv(store.EnvBackend, "s3")
	t.Setenv(store.EnvS3TimeoutSec, "not-a-number")
	_, err := store.ConfigFromEnv("/root")
	if err == nil {
		t.Fatal("expected error for invalid timeout")
	}
}

func TestConfigFromEnv_S3TimeoutSec(t *testing.T) {
	t.Setenv(store.EnvBackend, "s3")
	t.Setenv(store.EnvS3Bucket, "bucket")
	t.Setenv(store.EnvS3TimeoutSec, "60")
	cfg, err := store.ConfigFromEnv("/root")
	if err != nil {
		t.Fatalf("ConfigFromEnv: %v", err)
	}
	if cfg.S3.RequestTimeout.Seconds() != 60 {
		t.Errorf("expected 60s timeout, got %v", cfg.S3.RequestTimeout)
	}
}

func TestNewFromConfig_LocalBackend(t *testing.T) {
	cfg := &store.Config{
		Backend: store.BackendLocal,
		Root:    t.TempDir(),
	}
	s, err := store.NewFromConfig(cfg)
	if err != nil {
		t.Fatalf("NewFromConfig: %v", err)
	}
	// Verify a basic operation works.
	tasks, err := s.LoadAll()
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	if len(tasks) != 0 {
		t.Errorf("expected empty store, got %d tasks", len(tasks))
	}
}

func TestNewFromConfig_HealthCheck(t *testing.T) {
	cfg := &store.Config{
		Backend: store.BackendLocal,
		Root:    t.TempDir(),
	}
	s, err := store.NewFromConfig(cfg)
	if err != nil {
		t.Fatalf("NewFromConfig: %v", err)
	}
	if err := s.HealthCheck(); err != nil {
		t.Errorf("HealthCheck: %v", err)
	}
}
