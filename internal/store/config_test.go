package store_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/bmordue/tssk/internal/store"
)

func TestDefaultConfigFileContent(t *testing.T) {
	b, err := store.DefaultConfigFileContent()
	if err != nil {
		t.Fatalf("DefaultConfigFileContent: %v", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		t.Fatalf("json unmarshal: %v", err)
	}

	if got, ok := raw["backend"].(string); !ok || got != "local" {
		t.Errorf("expected backend=local, got %#v", raw["backend"])
	}
	if got, ok := raw["tasks_file"].(string); !ok || got != ".tsks/tasks.jsonl" {
		t.Errorf("expected tasks_file=.tsks/tasks.jsonl, got %#v", raw["tasks_file"])
	}
	if got, ok := raw["docs_dir"].(string); !ok || got != ".tsks/docs" {
		t.Errorf("expected docs_dir=.tsks/docs, got %#v", raw["docs_dir"])
	}
	if got, ok := raw["display_hash_length"].(float64); !ok || int(got) != 9 {
		t.Errorf("expected display_hash_length=9, got %#v", raw["display_hash_length"])
	}
}

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

func TestConfigFromEnv_TasksFile(t *testing.T) {
	t.Setenv(store.EnvBackend, "")
	t.Setenv(store.EnvTasksFile, "custom/my-tasks.jsonl")
	cfg, err := store.ConfigFromEnv("/root")
	if err != nil {
		t.Fatalf("ConfigFromEnv: %v", err)
	}
	if cfg.TasksFile != "custom/my-tasks.jsonl" {
		t.Errorf("unexpected TasksFile: %q", cfg.TasksFile)
	}
}

func TestConfigFromEnv_DocsDir(t *testing.T) {
	t.Setenv(store.EnvBackend, "")
	t.Setenv(store.EnvDocsDir, "task-details")
	cfg, err := store.ConfigFromEnv("/root")
	if err != nil {
		t.Fatalf("ConfigFromEnv: %v", err)
	}
	if cfg.DocsDir != "task-details" {
		t.Errorf("unexpected DocsDir: %q", cfg.DocsDir)
	}
}

func TestConfigFromEnv_HashLength(t *testing.T) {
	t.Setenv(store.EnvBackend, "")
	t.Setenv(store.EnvDisplayHashLength, "16")
	cfg, err := store.ConfigFromEnv("/root")
	if err != nil {
		t.Fatalf("ConfigFromEnv: %v", err)
	}
	if cfg.DisplayHashLength != 16 {
		t.Errorf("expected DisplayHashLength 16, got %d", cfg.DisplayHashLength)
	}
}

func TestConfigFromEnv_HashLengthInvalid(t *testing.T) {
	t.Setenv(store.EnvBackend, "")
	for _, bad := range []string{"0", "65", "abc", "-1"} {
		t.Setenv(store.EnvDisplayHashLength, bad)
		_, err := store.ConfigFromEnv("/root")
		if err == nil {
			t.Errorf("expected error for TSSK_DISPLAY_HASH_LENGTH=%q", bad)
		}
	}
}

func TestNewFromConfig_CustomTasksFileAndDocsDir(t *testing.T) {
	dir := t.TempDir()
	cfg := &store.Config{
		Backend:   store.BackendLocal,
		Root:      dir,
		TasksFile: "custom-tasks.jsonl",
		DocsDir:   "custom-docs",
	}
	s, err := store.NewFromConfig(cfg)
	if err != nil {
		t.Fatalf("NewFromConfig: %v", err)
	}
	// Add a task to exercise the custom paths.
	task, err := s.Add("Test task", "detail text", nil)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	// Verify the custom tasks file was created.
	if _, err := os.Stat(filepath.Join(dir, "custom-tasks.jsonl")); err != nil {
		t.Errorf("expected custom tasks file: %v", err)
	}
	// Verify the custom docs directory was created.
	// DocHash is the full 64-char SHA-256 hash; the filename uses the full hash.
	docPath := filepath.Join(dir, "custom-docs", task.DocHash+".md")
	if _, err := os.Stat(docPath); err != nil {
		t.Errorf("expected detail file at %s: %v", docPath, err)
	}
}

func TestNewFromConfig_CustomHashLength(t *testing.T) {
	dir := t.TempDir()
	cfg := &store.Config{
		Backend:           store.BackendLocal,
		Root:              dir,
		DisplayHashLength: 12,
	}
	s, err := store.NewFromConfig(cfg)
	if err != nil {
		t.Fatalf("NewFromConfig: %v", err)
	}
	task, err := s.Add("Hash length test", "some detail", nil)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	// DocHash is always the full 64-char hash.
	if len(task.DocHash) != 64 {
		t.Errorf("expected DocHash length 64, got %d: %s", len(task.DocHash), task.DocHash)
	}
	// Detail file is named with the full hash; DisplayHashLength only controls display output.
	docPath := filepath.Join(dir, ".tsks", "docs", task.DocHash+".md")
	if _, err := os.Stat(docPath); err != nil {
		t.Errorf("expected detail file at %s: %v", docPath, err)
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

// writeConfigFile writes content to {dir}/.tssk.json.
func writeConfigFile(t *testing.T, dir, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, store.ConfigFile), []byte(content), 0o644); err != nil {
		t.Fatalf("writeConfigFile: %v", err)
	}
}

func TestConfigFromFileAndEnv_NoFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(store.EnvBackend, "")
	cfg, err := store.ConfigFromFileAndEnv(dir)
	if err != nil {
		t.Fatalf("ConfigFromFileAndEnv: %v", err)
	}
	if cfg.Backend != store.BackendLocal {
		t.Errorf("expected local backend, got %q", cfg.Backend)
	}
	if cfg.Root != dir {
		t.Errorf("unexpected root: %q", cfg.Root)
	}
}

func TestConfigFromFileAndEnv_FileValues(t *testing.T) {
	dir := t.TempDir()
	writeConfigFile(t, dir, `{
		"tasks_file": "my-tasks.jsonl",
		"docs_dir": "my-docs",
		"display_hash_length": 16
	}`)
	t.Setenv(store.EnvBackend, "")
	t.Setenv(store.EnvTasksFile, "")
	t.Setenv(store.EnvDocsDir, "")
	t.Setenv(store.EnvDisplayHashLength, "")

	cfg, err := store.ConfigFromFileAndEnv(dir)
	if err != nil {
		t.Fatalf("ConfigFromFileAndEnv: %v", err)
	}
	if cfg.TasksFile != "my-tasks.jsonl" {
		t.Errorf("unexpected TasksFile: %q", cfg.TasksFile)
	}
	if cfg.DocsDir != "my-docs" {
		t.Errorf("unexpected DocsDir: %q", cfg.DocsDir)
	}
	if cfg.DisplayHashLength != 16 {
		t.Errorf("expected DisplayHashLength 16, got %d", cfg.DisplayHashLength)
	}
}

func TestConfigFromFileAndEnv_EnvOverridesFile(t *testing.T) {
	dir := t.TempDir()
	writeConfigFile(t, dir, `{
		"tasks_file": "file-tasks.jsonl",
		"docs_dir": "file-docs",
		"display_hash_length": 16
	}`)
	t.Setenv(store.EnvBackend, "")
	t.Setenv(store.EnvTasksFile, "env-tasks.jsonl")
	t.Setenv(store.EnvDocsDir, "")
	t.Setenv(store.EnvDisplayHashLength, "32")

	cfg, err := store.ConfigFromFileAndEnv(dir)
	if err != nil {
		t.Fatalf("ConfigFromFileAndEnv: %v", err)
	}
	// Env var overrides file value.
	if cfg.TasksFile != "env-tasks.jsonl" {
		t.Errorf("expected env TasksFile to win, got %q", cfg.TasksFile)
	}
	// File value used when no env var set.
	if cfg.DocsDir != "file-docs" {
		t.Errorf("expected file DocsDir, got %q", cfg.DocsDir)
	}
	// Env var overrides file display hash length.
	if cfg.DisplayHashLength != 32 {
		t.Errorf("expected env DisplayHashLength 32, got %d", cfg.DisplayHashLength)
	}
}

func TestConfigFromFileAndEnv_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	writeConfigFile(t, dir, `not valid json`)
	_, err := store.ConfigFromFileAndEnv(dir)
	if err == nil {
		t.Fatal("expected error for invalid JSON in config file")
	}
}

func TestConfigFromFileAndEnv_InvalidHashLengthInFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(store.EnvDisplayHashLength, "")
	// 65 is out of range and should be rejected.
	writeConfigFile(t, dir, `{"display_hash_length": 65}`)
	_, err := store.ConfigFromFileAndEnv(dir)
	if err == nil {
		t.Fatal("expected error for display_hash_length 65 in config file")
	}
	// -1 is also out of range (JSON allows negative integers).
	writeConfigFile(t, dir, `{"display_hash_length": -1}`)
	_, err = store.ConfigFromFileAndEnv(dir)
	if err == nil {
		t.Fatal("expected error for display_hash_length -1 in config file")
	}
}

func TestConfigFromFileAndEnv_BackendFromFile(t *testing.T) {
	dir := t.TempDir()
	writeConfigFile(t, dir, `{"backend": "local"}`)
	t.Setenv(store.EnvBackend, "")
	cfg, err := store.ConfigFromFileAndEnv(dir)
	if err != nil {
		t.Fatalf("ConfigFromFileAndEnv: %v", err)
	}
	if cfg.Backend != store.BackendLocal {
		t.Errorf("expected local backend from file, got %q", cfg.Backend)
	}
}

func TestConfigFromFileAndEnv_EnvOverridesFileBackend(t *testing.T) {
	dir := t.TempDir()
	// File says local, env says... (s3 would fail without bucket; just use local as override too
	// to verify the env var is actually consulted; use an unknown value to expect the right error).
	writeConfigFile(t, dir, `{"backend": "local"}`)
	t.Setenv(store.EnvBackend, "redis")
	_, err := store.ConfigFromFileAndEnv(dir)
	if err == nil {
		t.Fatal("expected error for unknown backend from env override")
	}
}

func TestConfigFromFileAndEnv_FileUsedForStore(t *testing.T) {
	dir := t.TempDir()
	writeConfigFile(t, dir, `{
		"tasks_file": "custom-tasks.jsonl",
		"docs_dir": "custom-docs",
		"display_hash_length": 8
	}`)
	t.Setenv(store.EnvBackend, "")
	t.Setenv(store.EnvTasksFile, "")
	t.Setenv(store.EnvDocsDir, "")
	t.Setenv(store.EnvDisplayHashLength, "")

	cfg, err := store.ConfigFromFileAndEnv(dir)
	if err != nil {
		t.Fatalf("ConfigFromFileAndEnv: %v", err)
	}
	s, err := store.NewFromConfig(cfg)
	if err != nil {
		t.Fatalf("NewFromConfig: %v", err)
	}
	task, err := s.Add("File config task", "some detail", nil)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	// DocHash is always the full 64-char hash.
	if len(task.DocHash) != 64 {
		t.Errorf("expected DocHash length 64, got %d: %s", len(task.DocHash), task.DocHash)
	}
	// Custom tasks file should have been created.
	if _, err := os.Stat(filepath.Join(dir, "custom-tasks.jsonl")); err != nil {
		t.Errorf("expected custom-tasks.jsonl: %v", err)
	}
	// Custom docs dir: file named with the full hash (DisplayHashLength only controls display).
	if _, err := os.Stat(filepath.Join(dir, "custom-docs", task.DocHash+".md")); err != nil {
		t.Errorf("expected detail file in custom-docs: %v", err)
	}
}
