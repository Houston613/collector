package repository

type StructMem struct {
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
	m.gauges[name] = value
}

func (m *StructMem) UpdateCounter(name string, value int64) {
	m.counters[name] += value
}

func (m *StructMem) GetGauge(name string) (float64, bool) {
	v, ok := m.gauges[name]
	return v, ok
}

func (m *StructMem) GetCounter(name string) (int64, bool) {
	v, ok := m.counters[name]
	return v, ok
}

func (m *StructMem) GetAllGauges() map[string]float64 {
	return m.gauges
}

func (m *StructMem) GetAllCounters() map[string]int64 {
	return m.counters
}
