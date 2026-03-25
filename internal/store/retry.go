package store

import (
	"errors"
	"fmt"
	"math"
	"time"
)

// RetryConfig controls exponential-backoff retry behaviour.
type RetryConfig struct {
	// MaxAttempts is the total number of attempts (including the first). Must be >= 1.
	MaxAttempts int
	// InitialDelay is the wait time before the second attempt.
	InitialDelay time.Duration
	// MaxDelay caps the per-attempt wait time.
	MaxDelay time.Duration
	// Multiplier is the growth factor applied to the delay on each retry (>= 1.0).
	Multiplier float64
}

// DefaultRetryConfig returns a sensible default retry configuration.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     5 * time.Second,
		Multiplier:   2.0,
	}
}

// withRetry executes fn up to cfg.MaxAttempts times, waiting between retries
// using exponential backoff.  Errors wrapping ErrNotFound are not retried as
// they indicate a permanent condition.  The last error is returned if all
// attempts fail.
func withRetry(cfg RetryConfig, fn func() error) error {
	if cfg.MaxAttempts < 1 {
		cfg.MaxAttempts = 1
	}
	if cfg.Multiplier < 1.0 {
		cfg.Multiplier = 1.0
	}

	var lastErr error
	delay := cfg.InitialDelay

	for attempt := 0; attempt < cfg.MaxAttempts; attempt++ {
		if err := fn(); err != nil {
			// ErrNotFound is a permanent condition; retrying will not help.
			if errors.Is(err, ErrNotFound) {
				return err
			}
			lastErr = err
			if attempt < cfg.MaxAttempts-1 {
				time.Sleep(delay)
				delay = time.Duration(math.Min(
					float64(delay)*cfg.Multiplier,
					float64(cfg.MaxDelay),
				))
			}
			continue
		}
		return nil
	}
	return fmt.Errorf("after %d attempt(s): %w", cfg.MaxAttempts, lastErr)
}

// RetryBackend wraps a Backend and retries each operation on failure.
type RetryBackend struct {
	inner Backend
	cfg   RetryConfig
}

// NewRetryBackend wraps b with exponential-backoff retry logic using cfg.
func NewRetryBackend(b Backend, cfg RetryConfig) *RetryBackend {
	return &RetryBackend{inner: b, cfg: cfg}
}

func (r *RetryBackend) ReadTasksData() (result []byte, err error) {
	err = withRetry(r.cfg, func() error {
		result, err = r.inner.ReadTasksData()
		return err
	})
	return
}

func (r *RetryBackend) WriteTasksData(data []byte) error {
	return withRetry(r.cfg, func() error {
		return r.inner.WriteTasksData(data)
	})
}

func (r *RetryBackend) ReadDetail(docHash string) (result []byte, err error) {
	err = withRetry(r.cfg, func() error {
		result, err = r.inner.ReadDetail(docHash)
		return err
	})
	return
}

func (r *RetryBackend) WriteDetail(docHash string, data []byte) error {
	return withRetry(r.cfg, func() error {
		return r.inner.WriteDetail(docHash, data)
	})
}

func (r *RetryBackend) DeleteDetail(docHash string) error {
	return withRetry(r.cfg, func() error {
		return r.inner.DeleteDetail(docHash)
	})
}

func (r *RetryBackend) HealthCheck() error {
	return withRetry(r.cfg, func() error {
		return r.inner.HealthCheck()
	})
}
