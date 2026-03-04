package repository

import "testing"

func TestUpdateGauge_StoresValue(t *testing.T) {
	s := NewStructMem()
	s.UpdateGauge("Alloc", 1024.5)

	got, ok := s.GetGauge("Alloc")
	if !ok {
		t.Fatal("gauge Alloc not found")
	}
	if got != 1024.5 {
		t.Errorf("expected 1024.5, got %v", got)
	}
}

func TestUpdateGauge_Overwrites(t *testing.T) {
	s := NewStructMem()
	s.UpdateGauge("Sys", 100)
	s.UpdateGauge("Sys", 200)

	got, _ := s.GetGauge("Sys")
	if got != 200 {
		t.Errorf("expected 200, got %v", got)
	}
}

func TestUpdateCounter_Accumulates(t *testing.T) {
	s := NewStructMem()
	s.UpdateCounter("PollCount", 1)
	s.UpdateCounter("PollCount", 1)
	s.UpdateCounter("PollCount", 3)

	got, ok := s.GetCounter("PollCount")
	if !ok {
		t.Fatal("counter PollCount not found")
	}
	if got != 5 {
		t.Errorf("expected 5, got %d", got)
	}
}

func TestUpdateCounter_StartsAtZero(t *testing.T) {
	s := NewStructMem()
	s.UpdateCounter("hits", 7)

	got, _ := s.GetCounter("hits")
	if got != 7 {
		t.Errorf("expected 7, got %d", got)
	}
}

func TestGetGauge_MissingReturnsNotFound(t *testing.T) {
	s := NewStructMem()
	_, ok := s.GetGauge("nonexistent")
	if ok {
		t.Error("expected ok=false for missing gauge")
	}
}

func TestGetCounter_MissingReturnsNotFound(t *testing.T) {
	s := NewStructMem()
	_, ok := s.GetCounter("nonexistent")
	if ok {
		t.Error("expected ok=false for missing counter")
	}
}
