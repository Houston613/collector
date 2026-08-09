package pool

import "sync"

// Resetter is an interface constraint for types that possess a Reset() method.
type Resetter interface {
	Reset()
}

// Pool represents a type-safe generic pool for objects that implement Resetter.
type Pool[T Resetter] struct {
	pool sync.Pool
}

// New creates and returns a pointer to a new generic Pool.
func New[T Resetter](newFn func() T) *Pool[T] {
	return &Pool[T]{
		pool: sync.Pool{
			New: func() any {
				return newFn()
			},
		},
	}
}

// Get retrieves an item of type T from the pool.
func (p *Pool[T]) Get() T {
	return p.pool.Get().(T)
}

// Put resets the item's state via Reset() and places it back into the pool.
func (p *Pool[T]) Put(item T) {
	var anyVal any = item
	if anyVal != nil {
		item.Reset()
	}
	p.pool.Put(item)
}
