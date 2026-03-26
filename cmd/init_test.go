package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteDefaultConfigIfMissing_CreatesFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tssk", "config.env")

	created, err := writeDefaultConfigIfMissing(path)
	if err != nil {
		t.Fatalf("writeDefaultConfigIfMissing: %v", err)
	}
	if !created {
		t.Fatal("expected config file to be created")
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != defaultConfig {
		t.Fatalf("unexpected config contents\nwant:\n%s\n\ngot:\n%s", defaultConfig, string(got))
	}
}

func TestWriteDefaultConfigIfMissing_FileExists(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.env")
	const existing = "TSSK_STORAGE_BACKEND=s3\n"
	if err := os.WriteFile(path, []byte(existing), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	created, err := writeDefaultConfigIfMissing(path)
	if err != nil {
		t.Fatalf("writeDefaultConfigIfMissing: %v", err)
	}
	if created {
		t.Fatal("expected config file not to be created when it already exists")
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != existing {
		t.Fatalf("existing config file was modified\nwant:\n%s\n\ngot:\n%s", existing, string(got))
	}
}