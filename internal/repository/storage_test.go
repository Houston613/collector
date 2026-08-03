package repository

import (
	models "collector/internal/model"
	"context"
	"testing"
)

func TestUpdateGauge_StoresValue(t *testing.T) {
	ctx := context.Background()
	s := NewStructMem()
	s.UpdateGauge(ctx, "Alloc", 1024.5)

	got, ok, err := s.GetGauge(ctx, "Alloc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("gauge Alloc not found")
	}
	if got != 1024.5 {
		t.Errorf("expected 1024.5, got %v", got)
	}
}

func TestUpdateGauge_Overwrites(t *testing.T) {
	ctx := context.Background()
	s := NewStructMem()
	s.UpdateGauge(ctx, "Sys", 100)
	s.UpdateGauge(ctx, "Sys", 200)

	got, _, _ := s.GetGauge(ctx, "Sys")
	if got != 200 {
		t.Errorf("expected 200, got %v", got)
	}
}

func TestUpdateCounter_Accumulates(t *testing.T) {
	ctx := context.Background()
	s := NewStructMem()
	s.UpdateCounter(ctx, "PollCount", 1)
	s.UpdateCounter(ctx, "PollCount", 1)
	s.UpdateCounter(ctx, "PollCount", 3)

	got, ok, err := s.GetCounter(ctx, "PollCount")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("counter PollCount not found")
	}
	if got != 5 {
		t.Errorf("expected 5, got %d", got)
	}
}

func TestUpdateCounter_StartsAtZero(t *testing.T) {
	ctx := context.Background()
	s := NewStructMem()
	s.UpdateCounter(ctx, "hits", 7)

	got, _, _ := s.GetCounter(ctx, "hits")
	if got != 7 {
		t.Errorf("expected 7, got %d", got)
	}
}

func TestGetGauge_MissingReturnsNotFound(t *testing.T) {
	ctx := context.Background()
	s := NewStructMem()
	_, ok, _ := s.GetGauge(ctx, "nonexistent")
	if ok {
		t.Error("expected ok=false for missing gauge")
	}
}

func TestGetCounter_MissingReturnsNotFound(t *testing.T) {
	ctx := context.Background()
	s := NewStructMem()
	_, ok, _ := s.GetCounter(ctx, "nonexistent")
	if ok {
		t.Error("expected ok=false for missing counter")
	}
}

func TestUpdateMetrics(t *testing.T) {
	ctx := context.Background()
	s := NewStructMem()

	val1 := 10.5
	val2 := 20.7
	var delta1 int64 = 5
	var delta2 int64 = 15

	metrics := []models.Metrics{
		{ID: "g1", MType: models.Gauge, Value: &val1},
		{ID: "g2", MType: models.Gauge, Value: &val2},
		{ID: "g3", MType: models.Gauge, Value: nil}, // nil check
		{ID: "c1", MType: models.Counter, Delta: &delta1},
		{ID: "c2", MType: models.Counter, Delta: &delta2},
		{ID: "c3", MType: models.Counter, Delta: nil},     // nil check
		{ID: "c1", MType: models.Counter, Delta: &delta1}, // Accumulates c1 to 10
		{ID: "unknown", MType: "unknown_type"},            // unknown type check
	}

	err := s.UpdateMetrics(ctx, metrics)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	g1, ok, _ := s.GetGauge(ctx, "g1")
	if !ok || g1 != 10.5 {
		t.Errorf("expected g1 to be 10.5, got %v (ok=%v)", g1, ok)
	}

	g2, ok, _ := s.GetGauge(ctx, "g2")
	if !ok || g2 != 20.7 {
		t.Errorf("expected g2 to be 20.7, got %v (ok=%v)", g2, ok)
	}

	_, ok, _ = s.GetGauge(ctx, "g3")
	if ok {
		t.Error("expected g3 to not exist since value was nil")
	}

	c1, ok, _ := s.GetCounter(ctx, "c1")
	if !ok || c1 != 10 {
		t.Errorf("expected c1 to be 10, got %v (ok=%v)", c1, ok)
	}

	c2, ok, _ := s.GetCounter(ctx, "c2")
	if !ok || c2 != 15 {
		t.Errorf("expected c2 to be 15, got %v (ok=%v)", c2, ok)
	}

	_, ok, _ = s.GetCounter(ctx, "c3")
	if ok {
		t.Error("expected c3 to not exist since delta was nil")
	}
}

func TestGetAllGaugesAndCounters(t *testing.T) {
	ctx := context.Background()
	s := NewStructMem()

	s.UpdateGauge(ctx, "g1", 1.2)
	s.UpdateGauge(ctx, "g2", 3.4)
	s.UpdateCounter(ctx, "c1", 10)
	s.UpdateCounter(ctx, "c2", 20)

	gauges, err := s.GetAllGauges(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(gauges) != 2 || gauges["g1"] != 1.2 || gauges["g2"] != 3.4 {
		t.Errorf("unexpected gauges: %v", gauges)
	}

	counters, err := s.GetAllCounters(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(counters) != 2 || counters["c1"] != 10 || counters["c2"] != 20 {
		t.Errorf("unexpected counters: %v", counters)
	}
}
