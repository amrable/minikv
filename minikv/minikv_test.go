package minikv

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/amrable/minikv/skiplist"
)

func phase2NoPanic(t *testing.T) {
	t.Helper()
	if p := recover(); p != nil {
		t.Fatalf("database operation panicked: %v", p)
	}
}

// Public API done check: no test-only initialization.
func TestPhase2CloseReopen(t *testing.T) {
	defer phase2NoPanic(t)
	dir := t.TempDir()
	db := Open(dir)
	db.Put("c", []byte("third"))
	db.Put("a", []byte("old"))
	db.Put("a", []byte("updated"))
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	db = Open(dir)
	for key, want := range map[string]string{"a": "updated", "c": "third"} {
		got, ok, err := db.Get(key)
		if err != nil || !ok || string(got) != want {
			t.Errorf("Get(%q) = %q, %v, %v; want %q, true, nil", key, got, ok, err, want)
		}
	}
	if _, ok, err := db.Get("missing"); err != nil || ok {
		t.Fatalf("missing lookup: found=%v, err=%v", ok, err)
	}
}

// Initialize only the memtable to isolate lookup behavior from Open's setup.
func phase2DB(dir string) *KV {
	return &KV{dataPath: dir, skiplist: *skiplist.New()}
}

func phase2File(t *testing.T, dir, name string, values map[string]string) {
	t.Helper()
	l := skiplist.New()
	for k, v := range values {
		l.Put(k, []byte(v))
	}
	if err := os.WriteFile(filepath.Join(dir, name), l.ToBytes(), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestPhase2LookupMatchesMap(t *testing.T) {
	dir := t.TempDir()
	phase2File(t, dir, "000001.sst", map[string]string{"a": "old", "b": "disk-only"})
	phase2File(t, dir, "000002.sst", map[string]string{"a": "new"})
	// An unfinished file must not be decoded.
	if err := os.WriteFile(filepath.Join(dir, "000003.sst.tmp"), []byte("unfinished"), 0644); err != nil {
		t.Fatal(err)
	}
	for _, memory := range []bool{false, true} {
		name := "disk"
		if memory {
			name = "memory"
		}
		t.Run(name, func(t *testing.T) {
			db := phase2DB(dir)
			model := map[string]string{"a": "new", "b": "disk-only"}
			if memory {
				db.Put("a", []byte("memory"))
				model["a"] = "memory"
			}
			for _, key := range []string{"a", "b", "missing"} {
				want, exists := model[key]
				got, ok, err := db.Get(key)
				if err != nil || ok != exists || string(got) != want {
					t.Errorf("Get(%q) = %q, %v, %v; want %q, %v, nil", key, got, ok, err, want, exists)
				}
			}
		})
	}
}

func TestPhase2ClosePublishesSSTable(t *testing.T) {
	dir := t.TempDir()
	db := phase2DB(dir)
	db.Put("a", []byte("saved"))
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	files, err := filepath.Glob(filepath.Join(dir, "*.sst"))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Fatalf("published %d SSTables; want 1", len(files))
	}
	b, err := os.ReadFile(files[0])
	if err != nil {
		t.Fatal(err)
	}
	got, ok := deserializeGet("a", b)
	if !ok || got.Deleted || string(got.Value) != "saved" {
		t.Fatalf("published entry = %+v, %v", got, ok)
	}
}

func TestPhase2MultipleClosesPreserveFiles(t *testing.T) {
	dir := t.TempDir()
	db := Open(dir)
	db.Put("a", []byte("old"))
	db.Put("b", []byte("keep"))
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	db = Open(dir)
	db.Put("a", []byte("new"))
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	files, err := filepath.Glob(filepath.Join(dir, "*.sst"))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 {
		t.Errorf("published %d SSTables after two closes; want 2 immutable files", len(files))
	}
	db = Open(dir)
	for key, want := range map[string]string{"a": "new", "b": "keep"} {
		got, ok, err := db.Get(key)
		if err != nil || !ok || string(got) != want {
			t.Errorf("Get(%q) = %q, %v, %v; want %q, true, nil", key, got, ok, err, want)
		}
	}
}
