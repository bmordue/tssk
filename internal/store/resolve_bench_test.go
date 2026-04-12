package store

import (
	"bytes"
	"fmt"
	"testing"
)

func BenchmarkResolveOneExact(b *testing.B) {
	const numTasks = 1000
	var buf bytes.Buffer
	for i := 1; i <= numTasks; i++ {
		fmt.Fprintf(&buf, "{\"id\":\"%d\",\"title\":\"Task %d\",\"status\":\"todo\",\"created_at\":\"2024-01-01T00:00:00Z\",\"doc_hash\":\"hash%d\"}\n", i, i, i)
	}
	data := buf.Bytes()

	backend := &mockLoadBackend{data: data}
	s := NewWithBackend(backend)

	// Warm the cache and idMap
	_, err := s.LoadAll()
	if err != nil {
		b.Fatal(err)
	}

	target := "500"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := s.Get(target)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkResolveOnePrefix(b *testing.B) {
	const numTasks = 1000
	var buf bytes.Buffer

	// Use IDs that have a unique prefix but the prefix itself is not an ID.
	// We'll use "T1", "T2", ..., "T1000"
	// Then "T1000" is an ID.
	// But "T1000" prefix will match it.
	// Wait, if I use "T1000" it's an exact match.
	// If I use "T100" it matches "T100", "T1000"...

	// Let's use "X-<number>" and "Y-<number>"
	// Task 1: "X-1"
	// Task 1000: "X-1000"
	// If we search for "X-1000" it's exact.

	// Let's use long IDs and search for a prefix.
	// ID: "abcdefghijklmnopqrstuvwxyz1"
	// Prefix: "abcdefghijklmnopqrstuvwxyz"

	for i := 1; i <= numTasks; i++ {
		id := fmt.Sprintf("task-id-prefix-%04d", i)
		fmt.Fprintf(&buf, "{\"id\":\"%s\",\"title\":\"Task %d\",\"status\":\"todo\",\"created_at\":\"2024-01-01T00:00:00Z\",\"doc_hash\":\"hash%d\"}\n", id, i, i)
	}
	data := buf.Bytes()

	backend := &mockLoadBackend{data: data}
	s := NewWithBackend(backend)

	// Warm the cache and idMap
	_, err := s.LoadAll()
	if err != nil {
		b.Fatal(err)
	}

	// Last ID is "task-id-prefix-1000"
	// Unique prefix: "task-id-prefix-1000" is the ID itself.
	// Let's search for "task-id-prefix-1000" -> exact match.

	// Let's make one ID special.
	// Task 1000 ID: "unique-id-9999"
	// Search for "unique" -> unique prefix match, NOT an exact match.

	buf.Reset()
	for i := 1; i < numTasks; i++ {
		id := fmt.Sprintf("task-%04d", i)
		fmt.Fprintf(&buf, "{\"id\":\"%s\",\"title\":\"Task %d\",\"status\":\"todo\",\"created_at\":\"2024-01-01T00:00:00Z\",\"doc_hash\":\"hash%d\"}\n", id, i, i)
	}
	fmt.Fprintf(&buf, "{\"id\":\"unique-task-id\",\"title\":\"Unique Task\",\"status\":\"todo\",\"created_at\":\"2024-01-01T00:00:00Z\",\"doc_hash\":\"hash1000\"}\n")

	data = buf.Bytes()
	backend.data = data
	s.cache = nil
	s.idMap = nil
	_, _ = s.LoadAll()

	prefix := "unique" // matches only "unique-task-id"

	b.Run("PrefixMatch", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := s.Get(prefix)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("NoMatch", func(b *testing.B) {
		noMatch := "zzzz"
		for i := 0; i < b.N; i++ {
			_, err := s.Get(noMatch)
			if err == nil {
				b.Fatal("expected error")
			}
		}
	})
}
