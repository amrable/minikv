package skiplist

import (
	"math/rand/v2"
	"unsafe"
)

const (
	MAX_LEVEL = 16
)

type List struct {
	head *node
}

type node struct {
	tuple   Tuple
	forward []*node
}

func newNode() *node {
	return &node{forward: make([]*node, MAX_LEVEL)}
}

func New() *List {
	return &List{head: newNode()}
}

func (l *List) Get(k string) (Tuple, bool) {
	curr := l.head
	for i := MAX_LEVEL - 1; i >= 0; i-- {
		for curr.forward[i] != nil && k > curr.forward[i].tuple.Key {
			curr = curr.forward[i]
		}
	}
	curr = curr.forward[0]
	if curr == nil || curr.tuple.Key != k {
		return Tuple{}, false
	}
	return cloneTuple(curr.tuple), true
}

func (l *List) Put(t Tuple) {
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
	for level := 0; level < MAX_LEVEL; level++ {
		n.forward[level] = update[level].forward[level]
		update[level].forward[level] = n
		if level > 0 && rand.IntN(2) == 0 {
			break
		}
	}
}

func (l *List) Scan(from, to string) []Tuple {
	curr := l.head
	for i := MAX_LEVEL - 1; i >= 0; i-- {
		for curr.forward[i] != nil && from > curr.forward[i].tuple.Key {
			curr = curr.forward[i]
		}
	}
	ans := make([]Tuple, 0)
	curr = curr.forward[0]
	for curr != nil && curr.tuple.Key <= to {
		ans = append(ans, cloneTuple(curr.tuple))
		curr = curr.forward[0]
	}
	return ans
}

func (l *List) Entries() []Tuple {
	entries := make([]Tuple, 0)
	for curr := l.head.forward[0]; curr != nil; curr = curr.forward[0] {
		entries = append(entries, cloneTuple(curr.tuple))
	}
	return entries
}

func (l *List) MarshalBinary() ([]byte, error) {
	b := make([]byte, 0)
	curr := l.head.forward[0]
	for curr != nil {
		tupleBytes, err := curr.tuple.MarshalBinary()
		if err != nil {
			return nil, err
		}
		b = append(b, tupleBytes...)
		curr = curr.forward[0]
	}
	return b, nil
}

func (l *List) Size() int {
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
