package repository

import (
	models "collector/internal/model"
	"context"
	"maps"
	"sync"
)

// StructMem implements MemRepository using in-memory maps protected by a mutex.
type StructMem struct {
	mu       sync.RWMutex
	gauges   map[string]float64
	counters map[string]int64
}

// Pinger defines an interface for checking the availability of a backing storage.
type Pinger interface {
	// Ping checks if the backing storage is reachable and functional.
	Ping(ctx context.Context) error
}

// Saver defines an interface for repositories that can persist their state to storage.
type Saver interface {
	Save() error
}


// MemRepository defines the set of methods required to read and write metrics.
type MemRepository interface {
	// UpdateGauge updates the value of a gauge metric by its name.
	UpdateGauge(ctx context.Context, name string, value float64) error
	// UpdateCounter updates the value of a counter metric by its name (accumulating the value).
	UpdateCounter(ctx context.Context, name string, value int64) error
	// UpdateMetrics performs a batch update of multiple metrics.
	UpdateMetrics(ctx context.Context, metrics []models.Metrics) error
	// GetGauge retrieves the value of a gauge metric by its name.
	// Returns the value, a boolean indicating if it was found, and any error encountered.
	GetGauge(ctx context.Context, name string) (float64, bool, error)
	// GetCounter retrieves the value of a counter metric by its name.
	// Returns the value, a boolean indicating if it was found, and any error encountered.
	GetCounter(ctx context.Context, name string) (int64, bool, error)
	// GetAllGauges returns a map copy of all currently stored gauge metrics.
	GetAllGauges(ctx context.Context) (map[string]float64, error)
	// GetAllCounters returns a map copy of all currently stored counter metrics.
	GetAllCounters(ctx context.Context) (map[string]int64, error)
}

// NewStructMem creates and initializes a new StructMem instance.
func NewStructMem() *StructMem {
	return &StructMem{
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
	}
}

// UpdateGauge sets the new value for a gauge metric.
func (m *StructMem) UpdateGauge(ctx context.Context, name string, value float64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.gauges[name] = value
	return nil
}

// UpdateCounter adds the value to a counter metric.
func (m *StructMem) UpdateCounter(ctx context.Context, name string, value int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.counters[name] += value
	return nil
}

// UpdateMetrics updates multiple gauge and counter metrics in a single batch operation.
func (m *StructMem) UpdateMetrics(ctx context.Context, metrics []models.Metrics) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, metric := range metrics {
		switch metric.MType {
		case models.Gauge:
			if metric.Value != nil {
				m.gauges[metric.ID] = *metric.Value
			}
		case models.Counter:
			if metric.Delta != nil {
				m.counters[metric.ID] += *metric.Delta
			}
		}
	}
	return nil
}

// GetGauge retrieves the current value of a gauge metric by name.
func (m *StructMem) GetGauge(ctx context.Context, name string) (float64, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.gauges[name]
	return v, ok, nil
}

// GetCounter retrieves the current value of a counter metric by name.
func (m *StructMem) GetCounter(ctx context.Context, name string) (int64, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.counters[name]
	return v, ok, nil
}

// GetAllGauges returns a thread-safe copy of all gauge metrics.
func (m *StructMem) GetAllGauges(ctx context.Context) (map[string]float64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	mapCopy := make(map[string]float64, len(m.gauges))
	maps.Copy(mapCopy, m.gauges)
	return mapCopy, nil
}

// GetAllCounters returns a thread-safe copy of all counter metrics.
func (m *StructMem) GetAllCounters(ctx context.Context) (map[string]int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	mapCopy := make(map[string]int64, len(m.counters))
	maps.Copy(mapCopy, m.counters)
	return mapCopy, nil
}
