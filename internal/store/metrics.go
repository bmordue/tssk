package store

import (
	"sync/atomic"
	"time"
)

// OperationMetrics holds counters and timing for a single operation type.
type OperationMetrics struct {
	Calls   atomic.Int64
	Errors  atomic.Int64
	TotalNs atomic.Int64 // cumulative duration in nanoseconds
}

// Avg returns the average duration per call, or 0 if there have been no calls.
func (m *OperationMetrics) Avg() time.Duration {
	calls := m.Calls.Load()
	if calls == 0 {
		return 0
	}
	return time.Duration(m.TotalNs.Load() / calls)
}

// Metrics aggregates storage operation statistics.
type Metrics struct {
	ReadTasksData  OperationMetrics
	WriteTasksData OperationMetrics
	ReadDetail     OperationMetrics
	WriteDetail    OperationMetrics
	DeleteDetail   OperationMetrics
	HealthCheck    OperationMetrics
}

// MeteredBackend wraps a Backend and records operation metrics.
type MeteredBackend struct {
	inner   Backend
	metrics *Metrics
}

// NewMeteredBackend wraps b and records metrics into m.
func NewMeteredBackend(b Backend, m *Metrics) *MeteredBackend {
	return &MeteredBackend{inner: b, metrics: m}
}

// Metrics returns the metrics collector used by this backend.
func (mb *MeteredBackend) Metrics() *Metrics {
	return mb.metrics
}

func record(m *OperationMetrics, fn func() error) error {
	start := time.Now()
	m.Calls.Add(1)
	err := fn()
	m.TotalNs.Add(time.Since(start).Nanoseconds())
	if err != nil {
		m.Errors.Add(1)
	}
	return err
}

func (mb *MeteredBackend) ReadTasksData() (result []byte, err error) {
	err = record(&mb.metrics.ReadTasksData, func() error {
		result, err = mb.inner.ReadTasksData()
		return err
	})
	return
}

func (mb *MeteredBackend) WriteTasksData(data []byte) error {
	return record(&mb.metrics.WriteTasksData, func() error {
		return mb.inner.WriteTasksData(data)
	})
}

func (mb *MeteredBackend) ReadDetail(docHash string) (result []byte, err error) {
	err = record(&mb.metrics.ReadDetail, func() error {
		result, err = mb.inner.ReadDetail(docHash)
		return err
	})
	return
}

func (mb *MeteredBackend) WriteDetail(docHash string, data []byte) error {
	return record(&mb.metrics.WriteDetail, func() error {
		return mb.inner.WriteDetail(docHash, data)
	})
}

func (mb *MeteredBackend) DeleteDetail(docHash string) error {
	return record(&mb.metrics.DeleteDetail, func() error {
		return mb.inner.DeleteDetail(docHash)
	})
}

func (mb *MeteredBackend) HealthCheck() error {
	return record(&mb.metrics.HealthCheck, func() error {
		return mb.inner.HealthCheck()
	})
}
