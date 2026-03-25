package store

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// LocalBackend implements Backend using the local filesystem.
// Files are stored under a root directory with the same layout as the
// original Store: tasks.jsonl at the root and docs/{hash}.md for details.
type LocalBackend struct {
	root string
}

// NewLocalBackend creates a LocalBackend rooted at the given directory.
func NewLocalBackend(root string) *LocalBackend {
	return &LocalBackend{root: root}
}

func (b *LocalBackend) tasksPath() string {
	return filepath.Join(b.root, tasksFile)
}

func (b *LocalBackend) docPath(docHash string) string {
	return filepath.Join(b.root, docsDir, docHash+".md")
}

func (b *LocalBackend) ensureDocsDir() error {
	return os.MkdirAll(filepath.Join(b.root, docsDir), 0o755)
}

// ReadTasksData returns the raw content of tasks.jsonl, or (nil, nil) when
// the file does not yet exist.
func (b *LocalBackend) ReadTasksData() ([]byte, error) {
	data, err := os.ReadFile(b.tasksPath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading tasks file: %w", err)
	}
	return data, nil
}

// WriteTasksData atomically replaces tasks.jsonl using a temp-file rename.
func (b *LocalBackend) WriteTasksData(data []byte) error {
	tasksPath := b.tasksPath()
	dir := filepath.Dir(tasksPath)

	tmpFile, err := os.CreateTemp(dir, "tasks-*.jsonl")
	if err != nil {
		return fmt.Errorf("creating temp tasks file: %w", err)
	}
	tmpName := tmpFile.Name()

	if _, err := tmpFile.Write(data); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpName)
		return fmt.Errorf("writing temp tasks file: %w", err)
	}

	if err := tmpFile.Sync(); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpName)
		return fmt.Errorf("syncing temp tasks file: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("closing temp tasks file: %w", err)
	}

	if err := os.Rename(tmpName, tasksPath); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("replacing tasks file: %w", err)
	}
	return nil
}

// ReadDetail returns the markdown content for the given docHash.
func (b *LocalBackend) ReadDetail(docHash string) ([]byte, error) {
	data, err := os.ReadFile(b.docPath(docHash))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("%w: %s", ErrNotFound, docHash)
		}
		return nil, fmt.Errorf("reading detail file: %w", err)
	}
	return data, nil
}

// WriteDetail creates or replaces the markdown detail file for the given docHash.
func (b *LocalBackend) WriteDetail(docHash string, data []byte) error {
	if err := b.ensureDocsDir(); err != nil {
		return fmt.Errorf("creating docs directory: %w", err)
	}
	if err := os.WriteFile(b.docPath(docHash), data, 0o644); err != nil {
		return fmt.Errorf("writing detail file: %w", err)
	}
	return nil
}

// DeleteDetail removes the markdown detail file for the given docHash.
// A missing file is treated as a no-op.
func (b *LocalBackend) DeleteDetail(docHash string) error {
	err := os.Remove(b.docPath(docHash))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("deleting detail file: %w", err)
	}
	return nil
}

// HealthCheck verifies the root directory is accessible.
func (b *LocalBackend) HealthCheck() error {
	if _, err := os.Stat(b.root); err != nil {
		return fmt.Errorf("local backend root %q: %w", b.root, err)
	}
	return nil
}
