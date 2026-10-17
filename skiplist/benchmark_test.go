package skiplist_test

import (
	"fmt"
	"math/rand/v2"
	"testing"

	"github.com/amrable/minikv/skiplist"
)

var benchmarkValue []byte
var benchmarkPairs []skiplist.Pair

func BenchmarkSkipList(b *testing.B) {
	for _, size := range []int{1_000, 10_000} {
		b.Run(fmt.Sprintf("N=%d", size), func(b *testing.B) {
			keys := make([]string, size)
			missing := make([]string, size)
			for i := range keys {
				keys[i] = fmt.Sprintf("%08d", i)
				missing[i] = keys[i] + "x"
			}
			order := rand.New(rand.NewPCG(1, 2)).Perm(size)
			value := []byte("0123456789abcdef") // Shared 16-byte value; allocation is outside timing.
			populate := func() *skiplist.List {
				l := skiplist.New()
				for _, i := range order {
					l.Put(keys[i], value)
				}
				return l
			}
			for _, operation := range []string{"GetHit", "GetMiss", "Update", "Scan100"} {
				b.Run(operation, func(b *testing.B) {
					l := populate()
					b.ReportAllocs()
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						index := order[i%size]
						switch operation {
						case "GetHit":
							var ok bool
							benchmarkValue, ok = l.Get(keys[index])
							if !ok {
								b.Fatal("existing key not found")
							}
						case "GetMiss":
							var ok bool
							benchmarkValue, ok = l.Get(missing[index])
							if ok {
								b.Fatal("missing key found")
							}
						case "Update":
							l.Put(keys[index], value)
						case "Scan100":
							start := index % (size - 99)
							benchmarkPairs = l.Scan(keys[start], keys[start+99])
							if len(benchmarkPairs) != 100 {
								b.Fatal("wrong scan length")
							}
						}
					}
				})
			}
			for _, operation := range []string{"Insert", "Delete"} {
				b.Run(operation, func(b *testing.B) {
					b.ReportAllocs()
					b.ResetTimer()
					b.StopTimer()
					// Reset between batches so inserts stay new and deletes hit existing keys.
					for done := 0; done < b.N; {
						l := skiplist.New()
						if operation == "Delete" {
							l = populate()
						}
						count := min(size, b.N-done)
						b.StartTimer()
						for _, index := range order[:count] {
							if operation == "Insert" {
								l.Put(keys[index], value)
							} else {
								l.Delete(keys[index])
							}
						}
						b.StopTimer()
						done += count
					}
				})
			}
		})
	}
}
