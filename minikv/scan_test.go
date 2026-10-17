package minikv

import (
	"sort"
	"testing"
)

func TestScanMatchesMapAcrossMemoryAndDisk(t *testing.T) {
	dir := t.TempDir()
	db := Open(dir)
	model := map[string]string{}
	put := func(k, v string) { db.Put(k, []byte(v)); model[k] = v }
	remove := func(k string) { db.Delete(k, nil); delete(model, k) }
	check := func(stage string) {
		t.Helper()
		for _, bounds := range [][2]string{{"", "z"}, {"b", "d"}, {"c", "c"}, {"d", "d"}, {"x", "z"}, {"z", "a"}} {
			got, err := db.Scan(bounds[0], bounds[1])
			if err != nil {
				t.Fatal(err)
			}
			var keys []string
			for key := range model {
				if key >= bounds[0] && key <= bounds[1] {
					keys = append(keys, key)
				}
			}
			sort.Strings(keys)
			if len(got) != len(keys) {
				t.Fatalf("%s Scan%v = %+v; want keys %v", stage, bounds, got, keys)
			}
			for i, key := range keys {
				if got[i].Key != key || string(got[i].Value) != model[key] {
					t.Fatalf("%s Scan%v entry %d = %+v; want %q=%q", stage, bounds, i, got[i], key, model[key])
				}
			}
		}
	}
	check("empty")
	put("e", "last")
	put("a", "first")
	put("c", "old")
	put("d", "delete later")
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	db = Open(dir)
	put("c", "new")
	remove("d")
	put("b", "")
	check("memory over disk")
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	db = Open(dir)
	check("disk over disk")
	put("d", "restored")
	remove("a")
	check("reinsert and delete")
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	db = Open(dir)
	check("reopened")
}

func TestScanEmptyKeyAndValueSurviveReopen(t *testing.T) {
	dir := t.TempDir()
	db := Open(dir)
	db.Put("", []byte{})
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	db = Open(dir)
	got, err := db.Scan("", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Key != "" || len(got[0].Value) != 0 {
		t.Fatalf("Scan(empty, empty) = %+v; want one empty key/value", got)
	}
}
