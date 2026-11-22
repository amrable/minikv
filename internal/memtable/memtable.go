package memtable

import (
	"slices"
	"sync"
)

const MAX_MEMTABLE_SIZE = (1 << 28) // 256MB

type MemtableManager struct {
	mu      sync.RWMutex
	current *List

	immutable []*List
}

func New() *MemtableManager {
	return &MemtableManager{
		current: NewList(),
	}
}

func (m *MemtableManager) Get(k string) (Tuple, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if t, ok := m.current.getIterator(k); ok && t.n.tuple.Key == k {
		return cloneTuple(t.n.tuple), ok
	}
	for i := len(m.immutable) - 1; i >= 0; i-- {
		if t, ok := m.immutable[i].getIterator(k); ok && t.n.tuple.Key == k {
			return cloneTuple(t.n.tuple), ok
		}
	}
	return Tuple{}, false
}

func (m *MemtableManager) Put(t Tuple) (Tuple, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.current.put(t)
	if m.current.size() > MAX_MEMTABLE_SIZE {
		m.rotateLocked()
	}
	return Tuple{}, false
}

func (m *MemtableManager) Delete(k string) (Tuple, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.current.put(Tuple{Key: k, Deleted: true})
	if m.current.size() > MAX_MEMTABLE_SIZE {
		m.rotateLocked()
	}
	return Tuple{}, false
}

func (m *MemtableManager) Scan(from, to string) []Tuple {
	m.mu.RLock()
	defer m.mu.RUnlock()

	itrs := make([]Iterator, 0)

	itr, ok := m.current.getIterator(from)
	if ok {
		itrs = append(itrs, itr)
	}
	for _, v := range slices.Backward(m.immutable) {
		itr, ok := v.getIterator(from)
		if ok {
			itrs = append(itrs, itr)
		}
	}
	resp := make([]Tuple, 0)

	for {
		type minKey struct {
			value string
			found bool
			itr   Iterator
		}
		var target minKey
		for _, itr := range itrs { // later first
			if itr.n == nil || itr.n.tuple.Key > to { // undesired iterator
				continue
			}
			if !target.found {
				target.value = itr.n.tuple.Key
				target.itr = itr
				target.found = true
			} else if target.value > itr.n.tuple.Key {
				target.value = itr.n.tuple.Key
				target.itr = itr
			}
		}
		if !target.found {
			break
		}
		resp = append(resp, cloneTuple(target.itr.n.tuple))
		for i := 0; i < len(itrs); i++ { // later first
			if itrs[i].n == nil {
				continue
			}
			if target.value == itrs[i].n.tuple.Key {
				// advance
				itrs[i] = itrs[i].Next()
			}
		}

	}
	return resp
}

func (m *MemtableManager) rotateLocked() {
	old := m.current

	m.immutable = append(m.immutable, old)
	m.current = NewList()
}
