package minikv_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/amrable/minikv/minikv"
)

var benchmarkValue []byte

func benchmarkKey(file, record int) string {
	return fmt.Sprintf("f%04d-k%08d", file, record)
}

func BenchmarkGetMemory(b *testing.B) {
	db := minikv.Open(b.TempDir())
	keys := make([]string, 10000)
	value := []byte("0123456789abcdef")
	for i := range keys {
		keys[i] = benchmarkKey(0, i)
		db.Put(keys[i], value)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		got, ok, err := db.Get(keys[i%len(keys)])
		if err != nil || !ok || string(got) != string(value) {
			b.Fatalf("Get = %q, %v, %v", got, ok, err)
		}
		benchmarkValue = got
	}
}

// Setup uses Close/Open so measured reads use published SSTables.
// Reads use the operating system's normal file cache; these are not cold-disk measurements.
func BenchmarkGetDisk(b *testing.B) {
	const recordsPerFile = 100
	for _, files := range []int{1, 10, 100} {
		b.Run(fmt.Sprintf("Files=%d", files), func(b *testing.B) {
			dir := b.TempDir()
			value := []byte("0123456789abcdef")
			for file := 0; file < files; file++ {
				db := minikv.Open(dir)
				for record := 0; record < recordsPerFile; record++ {
					db.Put(benchmarkKey(file, record), value)
				}
				if err := db.Close(); err != nil {
					b.Fatal(err)
				}
			}
			db := minikv.Open(dir)
			for _, scenario := range []struct {
				name, key string
				found     bool
			}{
				{"Newest", benchmarkKey(files-1, recordsPerFile-1), true},
				{"Oldest", benchmarkKey(0, recordsPerFile-1), true},
				{"Missing", "missing-key", false},
			} {
				b.Run(scenario.name, func(b *testing.B) {
					// Warm the lookup and verify correctness outside timing.
					got, ok, err := db.Get(scenario.key)
					if err != nil || ok != scenario.found || (ok && string(got) != string(value)) {
						b.Fatalf("Get = %q, %v, %v", got, ok, err)
					}
					b.ReportAllocs()
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						got, ok, err := db.Get(scenario.key)
						if err != nil || ok != scenario.found {
							b.Fatalf("Get: found=%v, err=%v", ok, err)
						}
						benchmarkValue = got
					}
				})
			}
		})
	}
}

// Each iteration closes the same populated memtable into an empty directory.
// Removing the published SSTable is outside timing, preventing directory growth.
func BenchmarkClose(b *testing.B) {
	for _, records := range []int{1000, 10000} {
		b.Run(fmt.Sprintf("Records=%d", records), func(b *testing.B) {
			dir := b.TempDir()
			db := minikv.Open(dir)
			value := []byte("0123456789abcdef")
			var serializedBytes int64
			for i := 0; i < records; i++ {
				key := benchmarkKey(0, i)
				db.Put(key, value)
				serializedBytes += int64(9 + len(key) + len(value))
			}
			b.SetBytes(serializedBytes)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if err := db.Close(); err != nil {
					b.Fatal(err)
				}
				b.StopTimer()
				files, err := filepath.Glob(filepath.Join(dir, "*.sst"))
				if err != nil {
					b.Fatal(err)
				}
				if len(files) != 1 {
					b.Fatalf("published %d files; want 1", len(files))
				}
				if err := os.Remove(files[0]); err != nil {
					b.Fatal(err)
				}
				b.StartTimer()
			}
		})
	}
}
