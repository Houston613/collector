package repository

import (
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
