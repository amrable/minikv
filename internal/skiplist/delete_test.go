package skiplist

import "testing"

func TestDeletePreservesOtherLevels(t *testing.T) {
	l := New()
	a, b, c := newNode(), newNode(), newNode()
	a.key, b.key, c.key = "a", "b", "c"
	a.value, b.value, c.value = Entry{Value: []byte("a")}, Entry{Value: []byte("b")}, Entry{Value: []byte("c")}
	// Bottom: HEAD -> a -> b -> c. Upper: HEAD -> a -> c.
	l.head.forward[0] = a
	a.forward[0], b.forward[0] = b, c
	l.head.forward[1] = a
	a.forward[1] = c

	l.Delete("b")
	if got, ok := l.Get("b"); !ok || !got.Deleted || len(got.Value) != 0 {
		t.Fatalf("Get(b) = %+v, %v; want tombstone, true", got, ok)
	}
	if a.forward[0] != b || b.forward[0] != c {
		t.Fatal("tombstoning b changed bottom-level connections")
	}
	if a.forward[1] != c {
		t.Fatal("deleting b changed the upper connection from a to c")
	}
}

func TestDeleteMatchesMap(t *testing.T) {
	l := New()
	model := make(map[string]Entry)
	keys := []string{"a", "b", "c", "d", "e"}
	check := func() {
		t.Helper()
		for _, key := range keys {
			want, exists := model[key]
			got, ok := l.Get(key)
			if ok != exists || string(got.Value) != string(want.Value) || got.Deleted != want.Deleted {
				t.Fatalf("Get(%q) = %+v, %v; want %+v, %v", key, got, ok, want, exists)
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
					t.Fatalf("unexpected key %q at level %d", n.key, level)
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
	check()
	for _, key := range []string{"c", "a", "e", "b", "d"} {
		l.Put(key, []byte(key))
		model[key] = Entry{Value: []byte(key)}
	}
	// Tombstones retain nodes, including after repeated deletion.
	for _, key := range []string{"c", "a", "e", "c", "b", "d"} {
		l.Delete(key)
		model[key] = Entry{Deleted: true}
		check()
	}
	l.Put("c", []byte("reinserted"))
	model["c"] = Entry{Value: []byte("reinserted")}
	check()
}

func TestDeleteAbsentKeyCreatesTombstone(t *testing.T) {
	l := New()
	l.Delete("disk-only")
	got, ok := l.Get("disk-only")
	if !ok || !got.Deleted {
		t.Fatalf("Get(disk-only) = %+v, %v; want tombstone, true", got, ok)
	}
	pairs := l.Scan("", "z")
	if len(pairs) != 1 || pairs[0].Key != "disk-only" || !pairs[0].Entry.Deleted {
		t.Fatalf("Scan must retain the tombstone for flushing; got %+v", pairs)
	}
}

func TestEmptyValueIsNotTombstone(t *testing.T) {
	l := New()
	l.Put("empty", []byte{})
	got, ok := l.Get("empty")
	if !ok || got.Deleted || len(got.Value) != 0 {
		t.Fatalf("Get(empty) = %+v, %v; want empty value, true", got, ok)
	}
}
