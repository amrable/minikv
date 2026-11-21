package memtable_test

import (
	"fmt"
	"testing"

	"github.com/amrable/minikv/internal/memtable"
)

func TestXxx(t *testing.T) {
	m := memtable.New()

	m.Put(memtable.Tuple{Key: "key", Value: []byte("value")})

	got, _ := m.Get("key")
	fmt.Printf("%+v\n", string(got.Value))
}
