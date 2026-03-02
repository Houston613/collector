package repository



type StructMem struct{
	gauges   map[string]float64
	counters map[string]int64
}

type MemRepository interface {
	UpdateGauge(name string, value float64)
	UpdateCounter(name string, value int64)
}

func NewStructMem() *StructMem {
	return &StructMem{
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
	}
}

func(m *StructMem) UpdateGauge (name string, value float64) {
	m.gauges[name] = value
}

func(m *StructMem) UpdateCounter (name string, value int64) {
	m.counters[name] = value
}
