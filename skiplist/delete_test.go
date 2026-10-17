package skiplist

import "testing"

func TestDeletePreservesOtherLevels(t *testing.T) {
	l := New()
	a, b, c := newNode(), newNode(), newNode()
	a.key, b.key, c.key = "a", "b", "c"
	a.value, b.value, c.value = []byte("a"), []byte("b"), []byte("c")
	// Bottom: HEAD -> a -> b -> c. Upper: HEAD -> a -> c.
	l.head.forward[0] = a
	a.forward[0], b.forward[0] = b, c
	l.head.forward[1] = a
	a.forward[1] = c

	l.Delete("b")
	if _, ok := l.Get("b"); ok {
		t.Fatal("deleted key b is still present")
	}
	if a.forward[0] != c {
		t.Fatal("bottom level should connect a to c")
	}
	if a.forward[1] != c {
		t.Fatal("deleting b changed the upper connection from a to c")
	}
}

func TestDeleteMatchesMap(t *testing.T) {
	l := New()
	model := make(map[string]string)
	keys := []string{"a", "b", "c", "d", "e"}
	check := func() {
		t.Helper()
		for _, key := range keys {
			want, exists := model[key]
			got, ok := l.Get(key)
			if ok != exists || string(got) != want {
				t.Fatalf("Get(%q) = %q, %v; want %q, %v", key, got, ok, want, exists)
			}
		}
		for level := 0; level < MAX_LEVEL; level++ {
			count := 0
			var previous *node
			for n := l.head.forward[level]; n != nil; n = n.forward[level] {
				count++
				if count > len(model) {
					t.Fatalf("level %d contains extra nodes or a cycle", level)
				}
				if _, exists := model[n.key]; !exists {
					t.Fatalf("deleted key %q remains at level %d", n.key, level)
				}
				if previous != nil && previous.key >= n.key {
					t.Fatalf("level %d is not sorted", level)
				}
				previous = n
			}
			if level == 0 && count != len(model) {
				t.Fatalf("bottom level has %d nodes; want %d", count, len(model))
			}
		}
	}
	l.Delete("missing") // Empty list.
	check()
	for _, key := range []string{"c", "a", "e", "b", "d"} {
		l.Put(key, []byte(key))
		model[key] = key
	}
	// Missing key, middle, first, last, repeated deletion, then empty again.
	for _, key := range []string{"bb", "c", "a", "e", "c", "b", "d"} {
		l.Delete(key)
		delete(model, key)
		check()
	}
	l.Put("c", []byte("reinserted"))
	model["c"] = "reinserted"
	check()
}
