package hashmap

import "sync"

type MapEmpty[T comparable] struct {
	sync.RWMutex
	m map[T]struct{}
}

// NewMapEmpty returns an empty T set initialized with specific size
func NewMapEmpty[T comparable](size int) *MapEmpty[T] {
	return &MapEmpty[T]{
		RWMutex: sync.RWMutex{},
		m:       make(map[T]struct{}, size),
	}
}

// Add adds the specified element to this set
// Always returns true due to the build-in map doesn't indicate caller whether the given element already exists
// Reserves the return type for future extension
func (hs *MapEmpty[T]) Add(value T) bool {
	hs.Lock()
	hs.m[value] = struct{}{}
	hs.Unlock()
	return true
}

// Contains returns true if this set contains the specified element
func (hs *MapEmpty[T]) Contains(value T) (ok bool) {
	hs.RLock()
	_, ok = hs.m[value]
	hs.RUnlock()
	return ok
}

// Remove removes the specified element from this set
// Always returns true due to the build-in map doesn't indicate caller whether the given element already exists
// Reserves the return type for future extension
func (hs *MapEmpty[T]) Remove(value T) bool {
	hs.Lock()
	delete(hs.m, value)
	hs.Unlock()
	return true
}

func (hs *MapEmpty[T]) Flush() {
	hs.Lock()
	hs.m = make(map[T]struct{}, 16)
	hs.Unlock()
	return
}

// Range calls f sequentially for each value present in the hashset.
// If f returns false, range stops the iteration.
func (hs *MapEmpty[T]) Range(f func(value T) bool) {
	hs.Lock()
	for k := range hs.m {
		if !f(k) {
			break
		}
	}
	hs.Unlock()
}

// Len returns the number of elements of this set
func (hs *MapEmpty[T]) Len() (n int) {
	hs.RLock()
	n = len(hs.m)
	hs.RUnlock()
	return
}
