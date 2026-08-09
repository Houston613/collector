package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMetricsReset(t *testing.T) {
	delta := int64(42)
	val := float64(3.14)
	m := &Metrics{
		ID:    "cpu",
		MType: Gauge,
		Delta: &delta,
		Value: &val,
		Hash:  "abc123hash",
	}

	m.Reset()

	assert.Equal(t, "", m.ID)
	assert.Equal(t, "", m.MType)
	assert.Equal(t, int64(0), *m.Delta)
	assert.Equal(t, float64(0), *m.Value)
	assert.Equal(t, "", m.Hash)
}

func TestNilMetricsReset(t *testing.T) {
	var m *Metrics
	assert.NotPanics(t, func() {
		m.Reset()
	})
}
