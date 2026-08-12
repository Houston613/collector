package pool_test

import (
	"sync"
	"testing"

	"collector/pkg/pool"
)

type sampleStruct struct {
	Name  string
	Count int
	Slice []string
	Map   map[string]int
}

func (s *sampleStruct) Reset() {
	if s == nil {
		return
	}
	s.Name = ""
	s.Count = 0
	s.Slice = s.Slice[:0]
	clear(s.Map)
}

func TestPoolGetPutReset(t *testing.T) {
	p := pool.New(func() *sampleStruct {
		return &sampleStruct{
			Map: make(map[string]int),
		}
	})

	// Get new object
	obj := p.Get()
	if obj == nil {
		t.Fatalf("expected non-nil object from Get()")
	}

	// Modify fields
	obj.Name = "test"
	obj.Count = 42
	obj.Slice = append(obj.Slice, "a", "b")
	obj.Map["key"] = 100

	// Put back in pool
	p.Put(obj)

	// Retrieve object again
	obj2 := p.Get()
	if obj2.Name != "" {
		t.Errorf("expected Name to be reset to empty string, got %q", obj2.Name)
	}
	if obj2.Count != 0 {
		t.Errorf("expected Count to be reset to 0, got %d", obj2.Count)
	}
	if len(obj2.Slice) != 0 {
		t.Errorf("expected Slice to be reset to len 0, got %d", len(obj2.Slice))
	}
	if len(obj2.Map) != 0 {
		t.Errorf("expected Map to be cleared, got %d items", len(obj2.Map))
	}
}

func TestPoolConcurrency(t *testing.T) {
	p := pool.New(func() *sampleStruct {
		return &sampleStruct{
			Map: make(map[string]int),
		}
	})

	var wg sync.WaitGroup
	const goroutines = 50
	const iterations = 100

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				obj := p.Get()
				obj.Name = "concurrent"
				obj.Count = j
				obj.Slice = append(obj.Slice, "item")
				obj.Map["val"] = j
				p.Put(obj)
			}
		}()
	}

	wg.Wait()
}
