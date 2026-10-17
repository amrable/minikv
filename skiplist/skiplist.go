package skiplist

import "math/rand/v2"

const (
	MAX_LEVEL = 16
)

type List struct {
	head *node
}

type node struct {
	key     string
	value   []byte
	forward []*node
}

func newNode() *node {
	return &node{forward: make([]*node, MAX_LEVEL)}
}

func New() *List {
	return &List{head: newNode()}
}

func (l *List) Get(k string) ([]byte, bool) {
	curr := l.head
	for i := MAX_LEVEL - 1; i >= 0; i-- {
		for curr.forward[i] != nil && k > curr.forward[i].key {
			curr = curr.forward[i]
		}
	}
	curr = curr.forward[0]
	if curr == nil || curr.key != k {
		return nil, false
	}
	return curr.value, true
}

func (l *List) Put(k string, v []byte) {
	update := make([]*node, 0)
	curr := l.head
	for i := MAX_LEVEL - 1; i >= 0; i-- {
		for curr.forward[i] != nil && k > curr.forward[i].key {
			curr = curr.forward[i]
		}
		update = append(update, curr)
	}
	if curr.forward[0] != nil && curr.forward[0].key == k {
		curr.forward[0].value = v
		return
	}
	n := newNode()
	n.key = k
	n.value = v
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

func (l *List) Delete(k string) {
	update := make([]*node, 0)
	curr := l.head
	for i := MAX_LEVEL - 1; i >= 0; i-- {
		for curr.forward[i] != nil && k > curr.forward[i].key {
			curr = curr.forward[i]
		}
		update = append(update, curr)
	}
	curr = curr.forward[0]
	if curr == nil || curr.key != k { // nothing to delete
		return
	}
	level := 0
	for i := MAX_LEVEL - 1; i >= 0; i-- {
		if update[i].forward[level] != curr {
			break
		}
		update[i].forward[level] = curr.forward[level]

		level++
		if level == len(curr.forward) {
			break
		}
	}
}

type Pair struct {
	Key   string
	Value []byte
}

func (l *List) Scan(from, to string) []Pair {
	curr := l.head
	for i := MAX_LEVEL - 1; i >= 0; i-- {
		for curr.forward[i] != nil && from > curr.forward[i].key {
			curr = curr.forward[i]
		}
	}
	ans := make([]Pair, 0)
	curr = curr.forward[0]
	for curr != nil && curr.key <= to {
		ans = append(ans, Pair{curr.key, curr.value})
		curr = curr.forward[0]
	}
	return ans
}
