package skiplist_test

import (
	"testing"

	"github.com/amrable/minikv/skiplist"
)

func TestList(t *testing.T) {
	l := skiplist.New()
	l.Put("a", []byte("b"))
	l.Put("aa", []byte("bb"))
	l.Put("aaa", []byte("bbb"))
	gotB, ok := l.Get("a")
	if !ok || gotB.Deleted || string(gotB.Value) != "b" {
		t.Fatal("wrong answer")
	}
	gotB, ok = l.Get("aa")
	if !ok || gotB.Deleted || string(gotB.Value) != "bb" {
		t.Fatal("wrong answer")
	}
}
