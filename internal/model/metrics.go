package models

const (
	// Counter represents a counter metric type that accumulates values.
	Counter = "counter"
	// Gauge represents a gauge metric type that takes a single numerical value.
	Gauge = "gauge"
)

// Metrics represents a data structure for sending and receiving metric payloads.
// Delta and Value are pointers to distinguish between zero values and unset values.
type Metrics struct {
	ID    string   `json:"id"`              // unique metric identifier
	MType string   `json:"type"`            // metric type (gauge or counter)
	Delta *int64   `json:"delta,omitempty"` // value if it's a counter
	Value *float64 `json:"value,omitempty"` // value if it's a gauge
	Hash  string   `json:"hash,omitempty"`  // hash for integrity checks
}
