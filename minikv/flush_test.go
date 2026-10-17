package minikv

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/amrable/minikv/skiplist"
)

func countSSTables(t *testing.T, dir string) int {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(dir, "*.sst"))
	if err != nil {
		t.Fatal(err)
	}
	return len(files)
}

func TestAutomaticFlushPreservesLatestValues(t *testing.T) {
	dir := t.TempDir()
	db := Open(dir)
	model := map[string]string{}
	check := func() {
		t.Helper()
		for key, want := range model {
			got, ok, err := db.Get(key)
			if err != nil || !ok || string(got) != want {
				t.Fatalf("Get(%q)=%q,%v,%v; want %q", key, got, ok, err, want)
			}
		}
	}
	for round := 0; round < 3; round++ {
		for i := 0; i < 12; i++ {
			key := fmt.Sprintf("key-%02d", i)
			value := fmt.Sprintf("round-%d", round)
			if err := db.Put(key, []byte(value)); err != nil {
				t.Fatal(err)
			}
			model[key] = value
			if db.skiplist.Size() > MAX_INMEM_BYTES {
				t.Fatal("memtable still exceeds threshold after successful Put")
			}
		}
		check()
	}
	if n := countSSTables(t, dir); n < 3 {
		t.Fatalf("published %d files; want several automatic flushes", n)
	}
	// A small write remains in memory and overrides its disk value.
	if err := db.Put("key-00", []byte("memory")); err != nil {
		t.Fatal(err)
	}
	model["key-00"] = "memory"
	check()
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	db = Open(dir)
	check()
}

func TestDeleteFlushesAtMemoryThreshold(t *testing.T) {
	dir := t.TempDir()
	db := Open(dir)
	for i := 0; i < 12; i++ {
		if err := db.Delete(fmt.Sprintf("deleted-%02d", i), nil); err != nil {
			t.Fatal(err)
		}
		if db.skiplist.Size() > MAX_INMEM_BYTES {
			t.Fatal("deletion left memtable over threshold")
		}
	}
	if countSSTables(t, dir) == 0 {
		t.Fatal("deletion-only workload did not flush")
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	db = Open(dir)
	for i := 0; i < 12; i++ {
		key := fmt.Sprintf("deleted-%02d", i)
		if _, ok, err := db.Get(key); err != nil || ok {
			t.Fatalf("Get(%q): found=%v, err=%v", key, ok, err)
		}
	}
}

func TestFailedFlushPreservesMemtableForRetry(t *testing.T) {
	for _, operation := range []string{"put", "delete"} {
		t.Run(operation, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "db")
			if err := os.Mkdir(dir, 0755); err != nil {
				t.Fatal(err)
			}
			db := Open(dir)
			if err := db.Put("keep", []byte("safe")); err != nil {
				t.Fatal(err)
			}
			original := db.skiplist
			// A directory at the next temporary path forces publication to fail.
			blocked := filepath.Join(dir, "0000000001.sst.tmp")
			if err := os.Mkdir(blocked, 0755); err != nil {
				t.Fatal(err)
			}
			key := strings.Repeat("x", MAX_INMEM_BYTES)
			var err error
			if operation == "put" {
				err = db.Put(key, []byte("pending"))
			} else {
				err = db.Delete(key, nil)
			}
			if err == nil {
				t.Fatal("flush failure was not returned")
			}
			if db.skiplist != original {
				t.Fatal("failed flush replaced the memtable")
			}
			if got, ok := db.skiplist.Get("keep"); !ok || string(got.Value) != "safe" {
				t.Fatal("failed flush lost earlier data")
			}
			pending, ok := db.skiplist.Get(key)
			if !ok || pending.Deleted != (operation == "delete") {
				t.Fatal("failed flush lost pending entry")
			}
			if operation == "put" && string(pending.Value) != "pending" {
				t.Fatal("failed flush lost pending value")
			}
			if countSSTables(t, dir) != 0 {
				t.Fatal("failed flush published an SSTable")
			}
			if err := os.Remove(blocked); err != nil {
				t.Fatal(err)
			}
			if err := db.checkOverflow(); err != nil {
				t.Fatal(err)
			}
			if db.skiplist == original {
				t.Fatal("successful retry did not release old memtable")
			}
			reopened := Open(dir)
			if got, ok, err := reopened.Get("keep"); err != nil || !ok || string(got) != "safe" {
				t.Fatalf("retry lost data: %q,%v,%v", got, ok, err)
			}
			got, ok, err := reopened.Get(key)
			if err != nil || ok != (operation == "put") || (ok && string(got) != "pending") {
				t.Fatalf("retry lost pending entry: %q,%v,%v", got, ok, err)
			}
		})
	}
}

func TestMemtableSizeTracksOverwritesAndTombstones(t *testing.T) {
	l := skiplist.New()
	baseline := l.Size()
	l.Put("a", []byte("12345"))
	initial := l.Size()
	if initial <= baseline+6 {
		t.Fatal("size does not include node overhead")
	}
	l.Put("a", []byte("12"))
	if got := l.Size(); got != initial-3 {
		t.Fatalf("smaller overwrite size=%d; want %d", got, initial-3)
	}
	l.Put("a", []byte("123456789"))
	if got := l.Size(); got != initial+4 {
		t.Fatalf("larger overwrite size=%d; want %d", got, initial+4)
	}
	l.Delete("a")
	if got := l.Size(); got != initial-5 {
		t.Fatalf("tombstone size=%d; want %d", got, initial-5)
	}
}
