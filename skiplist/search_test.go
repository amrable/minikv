package skiplist

import "testing"

func TestListMatchesMap(t *testing.T) {
	l := New()
	model := make(map[string]string)
	for _, key := range []string{"d", "a", "f", "c", "b", "e", "c"} {
		value := "value-" + key
		if _, exists := model[key]; exists {
			value = "updated-" + key
		}
		l.Put(key, []byte(value))
		model[key] = value
		for key, want := range model {
			got, ok := l.Get(key)
			if !ok || got.Deleted || string(got.Value) != want {
				t.Fatalf("Get(%q) = %+v, %v; want %q, true", key, got, ok, want)
			}
		}
	}
	for _, key := range []string{"", "bb", "z"} {
		if _, ok := l.Get(key); ok {
			t.Fatalf("Get(%q) found a missing key", key)
		}
	}
	for level := 0; level < MAX_LEVEL; level++ {
		previous := l.head
		count := 0
		for n := previous.forward[level]; n != nil; n = n.forward[level] {
			count++
			if count > len(model) {
				t.Fatalf("level %d contains duplicates or a cycle", level)
			}
			if previous != l.head && previous.key >= n.key {
				t.Fatalf("level %d is not sorted: %q before %q", level, previous.key, n.key)
			}
			previous = n
		}
		if level == 0 && count != len(model) {
			t.Fatalf("bottom level has %d nodes; want %d", count, len(model))
		}
	}
}

func TestPutReplacesExistingValue(t *testing.T) {
	l := New()
	l.Put("a", []byte("old"))
	l.Put("a", []byte("new"))
	value, ok := l.Get("a")
	if !ok || value.Deleted || string(value.Value) != "new" {
		t.Fatalf("Get(a) = %+v, %v; want new, true", value, ok)
	}
	count := 0
	for n := l.head.forward[0]; n != nil; n = n.forward[0] {
		count++
	}
	if count != 1 {
		t.Fatalf("bottom level contains %d nodes; want 1", count)
	}
}

func TestGetAcrossBottomLevel(t *testing.T) {
	l := New()
	// A valid skip list where these nodes participate only in level 0.
	previous := l.head
	for _, key := range []string{"a", "b", "c", "d"} {
		n := newNode()
		n.key = key
		n.value = Entry{Value: []byte(key)}
		previous.forward[0] = n
		previous = n
	}
	value, ok := l.Get("d")
	if !ok || value.Deleted || string(value.Value) != "d" {
		t.Fatalf("Get(d) = %+v, %v; want d, true", value, ok)
	}
}
