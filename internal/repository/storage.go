package repository

import "maps"
import "sync"
import "context"

type StructMem struct {
	mu       sync.RWMutex
	gauges   map[string]float64
	counters map[string]int64
}

type MemRepository interface {
	UpdateGauge(name string, value float64) error
	UpdateCounter(name string, value int64) error
	GetGauge(name string) (float64, bool, error)
	GetCounter(name string) (int64, bool, error)
	GetAllGauges() (map[string]float64, error)
	GetAllCounters() (map[string]int64, error)
	Ping(ctx context.Context) error
}

func NewStructMem() *StructMem {
	return &StructMem{
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
	}
}

func (m *StructMem) UpdateGauge(name string, value float64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.gauges[name] = value
	return nil
}

func (m *StructMem) UpdateCounter(name string, value int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.counters[name] += value
	return nil
}

func (m *StructMem) GetGauge(name string) (float64, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.gauges[name]
	return v, ok, nil
}

func (m *StructMem) GetCounter(name string) (int64, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.counters[name]
	return v, ok, nil
}

func (m *StructMem) GetAllGauges() (map[string]float64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	mapCopy := make(map[string]float64, len(m.gauges))
	maps.Copy(mapCopy, m.gauges)
	return mapCopy, nil
}

func (m *StructMem) GetAllCounters() (map[string]int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	mapCopy := make(map[string]int64, len(m.counters))
	maps.Copy(mapCopy, m.counters)
	return mapCopy, nil
}

func (m *StructMem) Ping(ctx context.Context) error {
	return nil // Memory storage is always "up"
}
