package skiplist_test

import (
	"sort"
	"testing"

	"github.com/amrable/minikv/skiplist"
)

func TestScanMatchesMap(t *testing.T) {
	l := skiplist.New()
	model := map[string]skiplist.Entry{"a": {Value: []byte("first")}, "c": {Value: []byte("middle")}, "e": {Value: []byte("last")}}
	for _, key := range []string{"e", "a", "c"} {
		l.Put(key, model[key].Value)
	}
	l.Put("c", []byte("updated"))
	model["c"] = skiplist.Entry{Value: []byte("updated")}
	l.Put("d", []byte("deleted"))
	l.Delete("d")
	model["d"] = skiplist.Entry{Deleted: true}
	for _, bounds := range [][2]string{{"", "z"}, {"b", "d"}, {"c", "c"}, {"e", "e"}, {"b", "b"}, {"z", "zz"}, {"d", "b"}} {
		t.Run(bounds[0]+".."+bounds[1], func(t *testing.T) {
			want := []string{}
			for key := range model {
				if key >= bounds[0] && key <= bounds[1] {
					want = append(want, key)
				}
			}
			sort.Strings(want)
			got := l.Scan(bounds[0], bounds[1])
			if len(got) != len(want) {
				t.Fatalf("got %d pairs; want %d", len(got), len(want))
			}
			for i, key := range want {
				if got[i].Key != key || string(got[i].Entry.Value) != string(model[key].Value) || got[i].Entry.Deleted != model[key].Deleted {
					t.Fatalf("pair %d = (%q, %+v); want (%q, %+v)", i, got[i].Key, got[i].Entry, key, model[key])
				}
			}
		})
	}
	if got := skiplist.New().Scan("", "z"); len(got) != 0 {
		t.Fatal("empty list returned pairs")
	}
}

func TestScan(t *testing.T) {
	l := skiplist.New()
	for _, key := range []string{"d", "a", "c", "b"} {
		l.Put(key, []byte("value-"+key))
	}
	// Bounds are inclusive: b and c should both appear.
	got := l.Scan("b", "c")
	want := []string{"b", "c"}
	if len(got) != len(want) {
		t.Fatalf("Scan(b, c) returned %d pairs; want 2", len(got))
	}
	for i, key := range want {
		if got[i].Key != key || got[i].Entry.Deleted || string(got[i].Entry.Value) != "value-"+key {
			t.Fatalf("pair %d = (%q, %+v); want (%q, %+v)", i, got[i].Key, got[i].Entry, key, "value-"+key)
		}
	}
}
