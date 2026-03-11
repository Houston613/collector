package repository

import "maps"

import "sync"

type StructMem struct {
	mu       sync.RWMutex
	gauges   map[string]float64
	counters map[string]int64
}

type MemRepository interface {
	UpdateGauge(name string, value float64)
	UpdateCounter(name string, value int64)
	GetGauge(name string) (float64, bool)
	GetCounter(name string) (int64, bool)
	GetAllGauges() map[string]float64
	GetAllCounters() map[string]int64
}

func NewStructMem() *StructMem {
	return &StructMem{
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
	}
}

func (m *StructMem) UpdateGauge(name string, value float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.gauges[name] = value
}

//лок на чтение для консистентности 
func (m *StructMem) UpdateCounter(name string, value int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.counters[name] += value
}

func (m *StructMem) GetGauge(name string) (float64, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.gauges[name]
	return v, ok
}

func (m *StructMem) GetCounter(name string) (int64, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.counters[name]
	return v, ok
}
//теперь перед тем как отдать - делаем копию
func (m *StructMem) GetAllGauges() map[string]float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	mapCopy := make(map[string]float64, len(m.gauges))
	maps.Copy(mapCopy, m.gauges)
	return mapCopy
}

//теперь перед тем как отдать - делаем копию
func (m *StructMem) GetAllCounters() map[string]int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	mapCopy := make(map[string]int64, len(m.counters))
	maps.Copy(mapCopy, m.counters)
	return mapCopy
}
