package skiplist_test

import (
	"reflect"
	"testing"

	"github.com/amrable/minikv/skiplist"
)

func TestAllEmpty(t *testing.T) {
	if got := skiplist.New().All(); len(got) != 0 {
		t.Fatalf("All() = %+v; want no tuples", got)
	}
}

func TestAllSortedWithTombstone(t *testing.T) {
	l := skiplist.New()
	l.Put("c", []byte("old"))
	l.Put("a", []byte("first"))
	l.Put("b", []byte("removed"))
	l.Put("c", []byte("updated"))
	l.Delete("b")
	want := []skiplist.Tuple{
		{Key: "a", Entry: skiplist.Entry{Value: []byte("first")}},
		{Key: "b", Entry: skiplist.Entry{Deleted: true}},
		{Key: "c", Entry: skiplist.Entry{Value: []byte("updated")}},
	}
	if got := l.All(); !reflect.DeepEqual(got, want) {
		t.Fatalf("All() = %+v; want %+v", got, want)
	}
}
