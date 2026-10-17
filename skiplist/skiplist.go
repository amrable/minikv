package skiplist

import (
	"encoding/binary"
	"math/rand/v2"
	"unsafe"
)

const (
	MAX_LEVEL = 16
)

type List struct {
	head *node
}

type Tuple struct {
	Key   string
	Entry Entry
}

type Entry struct {
	Value   []byte
	Deleted bool
}

type node struct {
	key     string
	value   Entry
	forward []*node
}

func newNode() *node {
	return &node{forward: make([]*node, MAX_LEVEL)}
}

func New() *List {
	return &List{head: newNode()}
}

func (l *List) Get(k string) (Entry, bool) {
	curr := l.head
	for i := MAX_LEVEL - 1; i >= 0; i-- {
		for curr.forward[i] != nil && k > curr.forward[i].key {
			curr = curr.forward[i]
		}
	}
	curr = curr.forward[0]
	if curr == nil || curr.key != k {
		return Entry{}, false
	}
	return curr.value, true
}

func (l *List) Put(k string, v []byte) {
	l.putEntry(k, v, false)
}

func (l *List) Delete(k string) {
	l.putEntry(k, nil, true)
}

func (l *List) putEntry(k string, v []byte, isTombsstone bool) {
	update := make([]*node, 0)
	curr := l.head
	for i := MAX_LEVEL - 1; i >= 0; i-- {
		for curr.forward[i] != nil && k > curr.forward[i].key {
			curr = curr.forward[i]
		}
		update = append(update, curr)
	}
	if curr.forward[0] != nil && curr.forward[0].key == k {
		curr.forward[0].value = Entry{v, isTombsstone}
		return
	}
	n := newNode()
	n.key = k
	n.value = Entry{v, isTombsstone}
	level := 0
	for i := MAX_LEVEL - 1; i >= 0; i-- {
		// we update at least once
		newForward := update[i].forward[level]
		update[i].forward[level] = n
		n.forward[level] = newForward
		level++

		if rand.IntN(2) == 0 {
			break
		}
	}
}

func (l *List) Scan(from, to string) []Tuple {
	curr := l.head
	for i := MAX_LEVEL - 1; i >= 0; i-- {
		for curr.forward[i] != nil && from > curr.forward[i].key {
			curr = curr.forward[i]
		}
	}
	ans := make([]Tuple, 0)
	curr = curr.forward[0]
	for curr != nil && curr.key <= to {
		ans = append(ans, Tuple{curr.key, curr.value})
		curr = curr.forward[0]
	}
	return ans
}

func (l *List) all() []Tuple {
	curr := l.head.forward[0]
	ans := make([]Tuple, 0)
	for curr != nil {
		ans = append(ans, Tuple{curr.key, curr.value})
		curr = curr.forward[0]
	}
	return ans
}

func (l *List) ToBytes() []byte {
	b := make([]byte, 0)
	for _, t := range l.all() {
		isTombStone := t.Entry.Deleted
		keyLength := len(t.Key)
		valueLength := len(t.Entry.Value)
		k := t.Key
		v := t.Entry.Value

		flag := byte(0)
		if isTombStone {
			flag = 1
		}
		b = append(b, flag)
		b = binary.BigEndian.AppendUint32(b, uint32(keyLength))
		b = binary.BigEndian.AppendUint32(b, uint32(valueLength))
		b = append(b, k...)
		b = append(b, v...)
	}
	return b
}

func (l *List) Size() int {
	s := 0
	curr := l.head
	for curr != nil {
		s += int(unsafe.Sizeof(node{})) +
			len(curr.key) +
			len(curr.value.Value) +
			len(curr.forward)*8
		curr = curr.forward[0]
	}
	return s
}
