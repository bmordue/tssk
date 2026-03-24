package store_test

import (
	"errors"
	"testing"
	"time"

	"github.com/bmordue/tssk/internal/store"
)

// faultyBackend returns errors for the first N calls to a given operation,
// then succeeds.
type faultyBackend struct {
	store.Backend
	readFails int
	calls     int
}

func (f *faultyBackend) ReadTasksData() ([]byte, error) {
	f.calls++
	if f.calls <= f.readFails {
		return nil, errors.New("transient error")
	}
	return f.Backend.ReadTasksData()
}

func TestRetryBackend_SucceedsAfterTransientFailures(t *testing.T) {
	inner := store.NewLocalBackend(t.TempDir())
	faulty := &faultyBackend{Backend: inner, readFails: 2}

	cfg := store.RetryConfig{
		MaxAttempts:  3,
		InitialDelay: time.Millisecond, // fast for tests
		MaxDelay:     10 * time.Millisecond,
		Multiplier:   2,
	}
	retry := store.NewRetryBackend(faulty, cfg)

	data, err := retry.ReadTasksData()
	if err != nil {
		t.Fatalf("expected success after retries, got: %v", err)
	}
	if data != nil {
		t.Errorf("expected nil data from empty store, got %q", data)
	}
	if faulty.calls != 3 {
		t.Errorf("expected 3 calls (2 failures + 1 success), got %d", faulty.calls)
	}
}

func TestRetryBackend_FailsAfterMaxAttempts(t *testing.T) {
	inner := store.NewLocalBackend(t.TempDir())
	faulty := &faultyBackend{Backend: inner, readFails: 5}

	cfg := store.RetryConfig{
		MaxAttempts:  3,
		InitialDelay: time.Millisecond,
		MaxDelay:     10 * time.Millisecond,
		Multiplier:   2,
	}
	retry := store.NewRetryBackend(faulty, cfg)

	_, err := retry.ReadTasksData()
	if err == nil {
		t.Fatal("expected error after max attempts, got nil")
	}
	if faulty.calls != 3 {
		t.Errorf("expected exactly 3 calls, got %d", faulty.calls)
	}
}
