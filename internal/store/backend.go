package store

// Backend defines the low-level storage operations required by the Store.
// Implementations must be safe for concurrent use by a single goroutine
// (the Store serialises access).
type Backend interface {
	// ReadTasksData returns the raw JSONL content of the tasks metadata store.
	// It must return (nil, nil) when the store is empty or not yet initialised.
	ReadTasksData() ([]byte, error)

	// WriteTasksData atomically replaces the tasks metadata store with the
	// provided JSONL content.
	WriteTasksData(data []byte) error

	// ReadDetail returns the content of the markdown detail file identified
	// by docHash.  Returns an error wrapping ErrNotFound when not present.
	ReadDetail(docHash string) ([]byte, error)

	// WriteDetail stores the markdown detail content for the given docHash.
	WriteDetail(docHash string, data []byte) error

	// HealthCheck verifies that the backend is reachable and operational.
	HealthCheck() error
}
