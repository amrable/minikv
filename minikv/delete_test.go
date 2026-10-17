package minikv

import "testing"

func TestDeleteAndReinsertPersistAcrossReopen(t *testing.T) {
	dir := t.TempDir()
	db := Open(dir)
	db.Put("a", []byte("old"))
	db.Put("empty", []byte{})
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	db = Open(dir)
	db.Delete("a", nil)
	check := func() {
		t.Helper()
		if got, ok, err := db.Get("a"); err != nil || ok {
			t.Fatalf("deleted key = %q, %v, %v; want missing", got, ok, err)
		}
		if got, ok, err := db.Get("empty"); err != nil || !ok || len(got) != 0 {
			t.Fatalf("empty value = %q, %v, %v; want present empty value", got, ok, err)
		}
	}
	check()
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	db = Open(dir)
	check()
	db.Put("a", []byte("restored"))
	if got, ok, err := db.Get("a"); err != nil || !ok || string(got) != "restored" {
		t.Fatalf("reinserted key = %q, %v, %v", got, ok, err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	db = Open(dir)
	if got, ok, err := db.Get("a"); err != nil || !ok || string(got) != "restored" {
		t.Fatalf("reopened reinserted key = %q, %v, %v", got, ok, err)
	}
}
