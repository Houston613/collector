package repository

import (
	models "collector/internal/model"
	"context"
	"testing"
)

func BenchmarkUpdateGauge(b *testing.B) {
	ctx := context.Background()
	s := NewStructMem()

	for i := 0; b.Loop(); i++ {
		s.UpdateGauge(ctx, "Alloc", float64(i))
	}
}

func BenchmarkUpdateCounter(b *testing.B) {
	ctx := context.Background()
	s := NewStructMem()
	
	for i := 0; b.Loop(); i++ {
		s.UpdateCounter(ctx, "PollCount", int64(i))
	}
}

func BenchmarkUpdateMetrics(b *testing.B) {
	ctx := context.Background()
	s := NewStructMem()
	val := 10.5
	delta := int64(5)
	metrics := []models.Metrics{
		{ID: "g1", MType: models.Gauge, Value: &val},
		{ID: "c1", MType: models.Counter, Delta: &delta},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.UpdateMetrics(ctx, metrics)
	}
}
