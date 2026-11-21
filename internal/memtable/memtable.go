package memtable

import "sync"

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

	if t, ok := m.current.Get(k); ok {
		return t, ok
	}
	for i := len(m.immutable) - 1; i >= 0; i-- {
		if t, ok := m.immutable[i].Get(k); ok {
			return t, ok
		}
	}
	return Tuple{}, false
}

func (m *MemtableManager) Put(t Tuple) (Tuple, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.current.Put(t)
	if m.current.Size() > MAX_MEMTABLE_SIZE {
		m.rotateLocked()
	}
	return Tuple{}, false
}

func (m *MemtableManager) Delete(k string) (Tuple, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.current.Put(Tuple{Key: k, Deleted: true})
	if m.current.Size() > MAX_MEMTABLE_SIZE {
		m.rotateLocked()
	}
	return Tuple{}, false
}

func (m *MemtableManager) rotateLocked() {
	old := m.current

	m.immutable = append(m.immutable, old)
	m.current = NewList()
}
