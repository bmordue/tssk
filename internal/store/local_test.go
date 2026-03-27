package store_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bmordue/tssk/internal/store"
)

func newTempLocalBackend(t *testing.T) *store.LocalBackend {
	t.Helper()
	return store.NewLocalBackend(t.TempDir())
}

func TestLocalBackend_ReadTasksDataEmpty(t *testing.T) {
	b := newTempLocalBackend(t)
	data, err := b.ReadTasksData()
	if err != nil {
		t.Fatalf("ReadTasksData on empty dir: %v", err)
	}
	if data != nil {
		t.Errorf("expected nil data for empty store, got %q", data)
	}
}

func TestLocalBackend_WriteAndReadTasksData(t *testing.T) {
	b := newTempLocalBackend(t)
	payload := []byte(`{"id":"T-1"}` + "\n")

	if err := b.WriteTasksData(payload); err != nil {
		t.Fatalf("WriteTasksData: %v", err)
	}

	got, err := b.ReadTasksData()
	if err != nil {
		t.Fatalf("ReadTasksData: %v", err)
	}
	if string(got) != string(payload) {
		t.Errorf("round-trip mismatch: got %q, want %q", got, payload)
	}
}

func TestLocalBackend_WriteTasksDataIsAtomic(t *testing.T) {
	// Write twice; only the second write should be visible.
	b := newTempLocalBackend(t)
	_ = b.WriteTasksData([]byte("first\n"))
	if err := b.WriteTasksData([]byte("second\n")); err != nil {
		t.Fatalf("WriteTasksData: %v", err)
	}
	got, _ := b.ReadTasksData()
	if string(got) != "second\n" {
		t.Errorf("expected second write to win, got %q", got)
	}
}

func TestLocalBackend_WriteAndReadDetail(t *testing.T) {
	b := newTempLocalBackend(t)
	content := []byte("# Detail\n\nContent here.")

	if err := b.WriteDetail("abc123", content); err != nil {
		t.Fatalf("WriteDetail: %v", err)
	}

	got, err := b.ReadDetail("abc123")
	if err != nil {
		t.Fatalf("ReadDetail: %v", err)
	}
	if string(got) != string(content) {
		t.Errorf("detail round-trip mismatch: got %q, want %q", got, content)
	}
}

func TestLocalBackend_ReadDetailNotFound(t *testing.T) {
	b := newTempLocalBackend(t)
	_, err := b.ReadDetail("nonexistent")
	if err == nil {
		t.Fatal("expected error for missing detail, got nil")
	}
}

func TestLocalBackend_WriteDetailCreatesDocsDir(t *testing.T) {
	dir := t.TempDir()
	b := store.NewLocalBackend(dir)

	if err := b.WriteDetail("myhash", []byte("content")); err != nil {
		t.Fatalf("WriteDetail: %v", err)
	}

	docPath := filepath.Join(dir, ".tsks", "docs", "myhash.md")
	if _, err := os.Stat(docPath); err != nil {
		t.Errorf("expected detail file at %s: %v", docPath, err)
	}
}

func TestLocalBackend_HealthCheck(t *testing.T) {
	b := newTempLocalBackend(t)
	if err := b.HealthCheck(); err != nil {
		t.Errorf("HealthCheck on valid dir: %v", err)
	}
}

func TestLocalBackend_HealthCheckMissingDir(t *testing.T) {
	b := store.NewLocalBackend("/this/path/does/not/exist/tssk-test")
	if err := b.HealthCheck(); err == nil {
		t.Error("expected HealthCheck to fail for missing directory")
	}
}
