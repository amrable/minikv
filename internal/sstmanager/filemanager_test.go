package filemanager_test

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/amrable/minikv/filemanager"
)

func TestWritePublishesBytes(t *testing.T) {
	dir := t.TempDir()
	want := []byte{0, 1, 255, 'a', '\n'}
	if err := filemanager.Write(dir, want); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "0000000001.sst" {
		t.Fatalf("directory entries = %v; want only 0000000001.sst", entries)
	}
	got, err := os.ReadFile(filepath.Join(dir, entries[0].Name()))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("saved bytes = %v; want %v", got, want)
	}
}

func TestWriteGenerationsAndWalkMatchesModel(t *testing.T) {
	dir := t.TempDir()
	// Cross 9 -> 10 to check numeric ordering and zero padding.
	var model []string
	for i := 1; i <= 12; i++ {
		payload := fmt.Sprintf("generation-%d", i)
		if err := filemanager.Write(dir, []byte(payload)); err != nil {
			t.Fatal(err)
		}
		model = append([]string{payload}, model...)
		// Earlier SSTables must remain unchanged.
		for n := 1; n <= i; n++ {
			path := filepath.Join(dir, fmt.Sprintf("%010d.sst", n))
			got, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != fmt.Sprintf("generation-%d", n) {
				t.Fatalf("generation %d was overwritten: %q", n, got)
			}
		}
	}
	var got []string
	for payload, err := range filemanager.Walk(dir) {
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, string(payload))
	}
	if !reflect.DeepEqual(got, model) {
		t.Fatalf("Walk = %q; want %q", got, model)
	}
}

func TestWalkIgnoresUnpublishedFilesAndDirectories(t *testing.T) {
	dir := t.TempDir()
	for name, content := range map[string]string{
		"0000000001.sst":     "published",
		"0000000002.sst.tmp": "unfinished",
		"notes.txt":          "unrelated",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Mkdir(filepath.Join(dir, "0000000003.sst"), 0755); err != nil {
		t.Fatal(err)
	}
	var got []string
	for payload, err := range filemanager.Walk(dir) {
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, string(payload))
	}
	if !reflect.DeepEqual(got, []string{"published"}) {
		t.Fatalf("Walk = %q; want only published", got)
	}
}

func TestWalkEmptyDirectory(t *testing.T) {
	for _, err := range filemanager.Walk(t.TempDir()) {
		t.Fatalf("empty directory yielded an entry: %v", err)
	}
}

func TestMissingDirectoryReportsError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing")
	if err := filemanager.Write(path, []byte("data")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Write error = %v; want not-exist", err)
	}
	calls := 0
	for _, err := range filemanager.Walk(path) {
		calls++
		if !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("Walk error = %v; want not-exist", err)
		}
	}
	if calls != 1 {
		t.Fatalf("Walk yielded %d errors; want 1", calls)
	}
}

func TestWalkCanStopEarly(t *testing.T) {
	dir := t.TempDir()
	for _, payload := range []string{"old", "new"} {
		if err := filemanager.Write(dir, []byte(payload)); err != nil {
			t.Fatal(err)
		}
	}
	calls := 0
	for payload, err := range filemanager.Walk(dir) {
		if err != nil {
			t.Fatal(err)
		}
		calls++
		if string(payload) != "new" {
			t.Fatalf("first payload = %q; want new", payload)
		}
		break
	}
	if calls != 1 {
		t.Fatalf("received %d entries; want 1", calls)
	}
}
