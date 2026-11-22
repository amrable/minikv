package memtable

import (
	"math/rand/v2"
	"sync"
	"unsafe"
)

const (
	MAX_LEVEL = 16
)

type List struct {
	mu   sync.RWMutex
	head *node
}

type Iterator struct {
	n *node
}

func (i *Iterator) Next() Iterator {
	if i.n.forward[0] != nil {
		return Iterator{i.n.forward[0]}
	}
	return Iterator{}
}

type node struct {
	tuple   Tuple
	forward []*node
}

func newNode() *node {
	return &node{forward: make([]*node, MAX_LEVEL)}
}

func NewList() *List {
	return &List{head: newNode()}
}

// returns highest key >= query
func (l *List) getIterator(query string) (Iterator, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	curr := l.head
	for i := MAX_LEVEL - 1; i >= 0; i-- {
		for curr.forward[i] != nil && query > curr.forward[i].tuple.Key {
			curr = curr.forward[i]
		}
	}
	curr = curr.forward[0]
	if curr == nil {
		return Iterator{}, false
	}
	return Iterator{curr}, true
}

func (l *List) put(t Tuple) {
	l.mu.Lock()
	defer l.mu.Unlock()

	update := make([]*node, MAX_LEVEL)
	curr := l.head
	for level := MAX_LEVEL - 1; level >= 0; level-- {
		for curr.forward[level] != nil && t.Key > curr.forward[level].tuple.Key {
			curr = curr.forward[level]
		}
		update[level] = curr
	}
	if curr.forward[0] != nil && curr.forward[0].tuple.Key == t.Key {
		curr.forward[0].tuple = cloneTuple(t)
		return
	}
	n := newNode()
	n.tuple = cloneTuple(t)
	for level := range MAX_LEVEL {
		n.forward[level] = update[level].forward[level]
		update[level].forward[level] = n
		if level > 0 && rand.IntN(2) == 0 {
			break
		}
	}
}

func (l *List) size() int {
	l.mu.RLock()
	defer l.mu.RUnlock()

	s := 0
	curr := l.head
	for curr != nil {
		s += int(unsafe.Sizeof(node{})) +
			len(curr.tuple.Key) +
			len(curr.tuple.Value) +
			len(curr.forward)*int(unsafe.Sizeof((*node)(nil)))
		curr = curr.forward[0]
	}
	return s
}

func cloneTuple(t Tuple) Tuple {
	t.Value = append([]byte(nil), t.Value...)
	return t
}
