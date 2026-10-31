package wal_test

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/amrable/minikv/wal"
)

func TestWal(t *testing.T) {
	path := filepath.Join(t.TempDir(), "001.wal")
	w, err := wal.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	r := wal.Record{
		wal.Put,
		"key",
		[]byte("value"),
	}
	if err := w.Append(r); err != nil {
		t.Fatal(err)
	}

	got := make([]wal.Record, 0)
	if err := w.Replay(func(replayedRecord wal.Record) error {
		got = append(got, replayedRecord)
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	if len(got) != 1 {
		t.Fatalf("expected 1 record, got %d", len(got))
	}

	assertRecordEqual(t, got[0], r)
}

func assertRecordEqual(t *testing.T, a, b wal.Record) {
	t.Helper()
	if a.Key == b.Key &&
		bytes.Equal(a.Value, b.Value) &&
		a.Type == b.Type {
		return
	}
	t.Fatalf("records are not equal")
}
