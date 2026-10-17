# MiniKV

A small key/value storage engine built in Go to learn how an LSM tree works.
Writes enter a skip list in memory, then flush to sorted, immutable files
(SSTables). Reads combine memory and disk, keeping the newest version of each key.

This is an educational project. It uses only the Go standard library and will
keep that dependency policy as it grows.

The storage implementation is written by hand. AI is used for tests and guidance
through the [Coach skill](https://github.com/amrable/agent-skills/blob/main/skills/coach/SKILL.md).

## Current features

The first four [roadmap phases](docs/roadmap.md) are implemented:

- Skip list with updates, lookups, and sorted iteration.
- SSTable persistence on clean shutdown and reads after reopening.
- Synchronous flushing when estimated memtable usage exceeds 1,024 bytes.
- Tombstones that preserve deletions across flushes and reopening.
- Inclusive range scans with sorted results and version resolution.

The threshold is currently the `MAX_INMEM_BYTES` constant in
`minikv/minikv.go`. It estimates memtable usage, including node overhead;
it is not a process memory limit.

## Usage

Requires Go 1.27.1 or later. Create the data directory before opening the database.

```go
package main

import (
    "fmt"
    "log"
    "os"

    "github.com/amrable/minikv/minikv"
)

func main() {
    if err := run(); err != nil {
        log.Fatal(err)
    }
}

func run() (err error) {
    if err = os.MkdirAll("./data", 0755); err != nil {
        return err
    }

    db := minikv.Open("./data")
    defer func() {
        if closeErr := db.Close(); err == nil {
            err = closeErr
        }
    }()

    if err = db.Put("name", []byte("MiniKV")); err != nil {
        return err
    }

    value, found, err := db.Get("name")
    if err != nil {
        return err
    }
    fmt.Printf("%s, found=%v\n", value, found)

    entries, err := db.Scan("a", "z")
    if err != nil {
        return err
    }
    for _, entry := range entries {
        fmt.Printf("%s=%s\n", entry.Key, entry.Value)
    }

    // The current Delete signature takes an unused value argument.
    return db.Delete("name", nil)
}
```

`Open` creates an empty memtable. It reads existing SSTables on demand.
`Put` and `Delete` update memory before attempting a threshold flush.
If that flush fails, they return an error and retain the modified memtable.

## Storage format

Each SSTable contains records sorted by key:

```text
[type: 1 byte][key length: 4 bytes][value length: 4 bytes][key][value]
```

Lengths use big-endian byte order. Type `0` stores a value; type `1` marks
a deletion. Empty values remain distinct from deletions.

Files use increasing generation names such as `0000000001.sst`.
Publishing a file writes `.sst.tmp`, syncs and closes it, renames it,
then syncs the containing directory. Reads ignore temporary files.

Point reads check memory first, then SSTables from newest to oldest.
The current implementation reads each visited SSTable into memory and scans
its records. Range scans resolve versions and tombstones before sorting results.

## Tests and benchmarks

```sh
go test ./...
go test ./minikv -run '^$' -bench . -benchmem
go test ./skiplist -run '^$' -bench . -benchmem
```

Tests cover persistence, version precedence, range scans, tombstones, automatic
flushes, and retaining memory after flush failures. Benchmarks measure reads and
close operations. File reads use the operating system's normal cache.

## Limits and next steps

MiniKV currently runs single-threaded. Use one database instance per directory.

Unflushed writes can be lost in a crash because there is no write-ahead log yet.
There is no compaction, so old versions and tombstones accumulate on disk.
SSTable indexes, Bloom filters, and corruption handling remain future work.

The next roadmap phase adds a write-ahead log and recovery. See
[the roadmap](docs/roadmap.md) for the learning sequence.
