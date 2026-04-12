package store

import (
	"bytes"
	"fmt"
	"testing"
)

func BenchmarkLoadAllUncached(b *testing.B) {
	const numTasks = 10000
	var buf bytes.Buffer
	for i := 1; i <= numTasks; i++ {
		fmt.Fprintf(&buf, "{\"id\":\"%d\",\"title\":\"Task %d\",\"status\":\"todo\",\"created_at\":\"2024-01-01T00:00:00Z\",\"doc_hash\":\"hash%d\"}\n", i, i, i)
	}
	data := buf.Bytes()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		backend := &mockLoadBackend{data: data}
		s := NewWithBackend(backend)
		tasks, err := s.LoadAll()
		if err != nil {
			b.Fatal(err)
		}
		if len(tasks) != numTasks {
			b.Fatalf("expected %d tasks, got %d", numTasks, len(tasks))
		}
	}
}
