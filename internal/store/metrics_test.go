package store_test

import (
	"testing"

	"github.com/bmordue/tssk/internal/store"
)

func TestMeteredBackend_RecordsCallCounts(t *testing.T) {
	inner := store.NewLocalBackend(t.TempDir())
	m := &store.Metrics{}
	metered := store.NewMeteredBackend(inner, m)

	// Two ReadTasksData calls.
	if _, err := metered.ReadTasksData(); err != nil {
		t.Fatalf("ReadTasksData: %v", err)
	}
	if _, err := metered.ReadTasksData(); err != nil {
		t.Fatalf("ReadTasksData: %v", err)
	}

	if got := m.ReadTasksData.Calls.Load(); got != 2 {
		t.Errorf("expected 2 ReadTasksData calls, got %d", got)
	}
	if got := m.ReadTasksData.Errors.Load(); got != 0 {
		t.Errorf("expected 0 errors, got %d", got)
	}
}

func TestMeteredBackend_RecordsErrors(t *testing.T) {
	inner := store.NewLocalBackend("/this/does/not/exist")
	m := &store.Metrics{}
	metered := store.NewMeteredBackend(inner, m)

	if _, err := metered.ReadDetail("hash"); err == nil {
		t.Fatal("expected error reading from nonexistent path")
	}

	if got := m.ReadDetail.Errors.Load(); got != 1 {
		t.Errorf("expected 1 error, got %d", got)
	}
	if got := m.ReadDetail.Calls.Load(); got != 1 {
		t.Errorf("expected 1 call, got %d", got)
	}
}

func TestMeteredBackend_RecordsWriteDetail(t *testing.T) {
	inner := store.NewLocalBackend(t.TempDir())
	m := &store.Metrics{}
	metered := store.NewMeteredBackend(inner, m)

	if err := metered.WriteDetail("abc", []byte("content")); err != nil {
		t.Fatalf("WriteDetail: %v", err)
	}

	if got := m.WriteDetail.Calls.Load(); got != 1 {
		t.Errorf("expected 1 WriteDetail call, got %d", got)
	}
}

func TestMetrics_Avg(t *testing.T) {
	m := &store.Metrics{}
	// No calls: avg should be zero.
	if avg := m.ReadTasksData.Avg(); avg != 0 {
		t.Errorf("expected 0 avg with no calls, got %v", avg)
	}
}
