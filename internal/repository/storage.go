package repository

import (
	models "collector/internal/model"
	"context"
	"maps"
	"sync"
)

type StructMem struct {
	mu       sync.RWMutex
	gauges   map[string]float64
	counters map[string]int64
}

//интерфейс для проверки доступности хранилища
type Pinger interface {
	Ping(ctx context.Context) error
}

type MemRepository interface {
	UpdateGauge(ctx context.Context, name string, value float64) error
	UpdateCounter(ctx context.Context, name string, value int64) error
	UpdateMetrics(ctx context.Context, metrics []models.Metrics) error
	GetGauge(ctx context.Context, name string) (float64, bool, error)
	GetCounter(ctx context.Context, name string) (int64, bool, error)
	GetAllGauges(ctx context.Context) (map[string]float64, error)
	GetAllCounters(ctx context.Context) (map[string]int64, error)
}

func NewStructMem() *StructMem {
	return &StructMem{
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
	}
}

func (m *StructMem) UpdateGauge(ctx context.Context, name string, value float64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.gauges[name] = value
	return nil
}

func (m *StructMem) UpdateCounter(ctx context.Context, name string, value int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.counters[name] += value
	return nil
}

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

func (m *StructMem) GetGauge(ctx context.Context, name string) (float64, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.gauges[name]
	return v, ok, nil
}

func (m *StructMem) GetCounter(ctx context.Context, name string) (int64, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.counters[name]
	return v, ok, nil
}

func (m *StructMem) GetAllGauges(ctx context.Context) (map[string]float64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	mapCopy := make(map[string]float64, len(m.gauges))
	maps.Copy(mapCopy, m.gauges)
	return mapCopy, nil
}

func (m *StructMem) GetAllCounters(ctx context.Context) (map[string]int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	mapCopy := make(map[string]int64, len(m.counters))
	maps.Copy(mapCopy, m.counters)
	return mapCopy, nil
}
