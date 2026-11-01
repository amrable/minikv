package minikv

import (
	"fmt"
	"iter"

	"github.com/amrable/minikv/internal/skiplist"
	"github.com/amrable/minikv/internal/sstmanager"
	"github.com/amrable/minikv/internal/wal"
)

const MAX_INMEM_BYTES = 1024

type SSTManager interface {
	Tables() iter.Seq2[[]byte, error]
	Flush(data []byte) error
}

type WAL interface {
	Append(wal.Record) error
	Reset() error
	Close() error
}

type KV struct {
	sst      SSTManager
	wal      WAL
	skiplist *skiplist.List
}

type Entry struct {
	Key     string
	Value   []byte
	Deleted bool
}

type Record struct {
	Value   []byte
	Deleted bool
}

type DebugState struct {
	Memtable      []Entry
	MemtableBytes int
}

// DebugState returns a point-in-time view of the in-memory skiplist.
func (kv *KV) DebugState() DebugState {
	tuples := kv.skiplist.Entries()
	entries := make([]Entry, 0, len(tuples))
	for _, tuple := range tuples {
		entries = append(entries, Entry{
			Key:     tuple.Key,
			Value:   tuple.Value,
			Deleted: tuple.Deleted,
		})
	}
	return DebugState{
		Memtable:      entries,
		MemtableBytes: kv.skiplist.Size(),
	}
}

func Open(dirPath string) (*KV, error) {
	w, err := wal.Open(dirPath)
	if err != nil {
		return nil, err
	}
	kv := KV{
		wal:      w,
		sst:      sstmanager.New(dirPath),
		skiplist: skiplist.New(),
	}
	if err := w.Replay(func(r wal.Record) error {
		switch r.Type {
		case wal.Put:
			kv.skiplist.Put(skiplist.Tuple{Key: r.Key, Value: r.Value, Deleted: false})
		case wal.Delete:
			kv.skiplist.Put(skiplist.Tuple{Key: r.Key, Value: nil, Deleted: true})
		}
		return nil
	}); err != nil {
		_ = w.Close()
		return nil, fmt.Errorf("replay WAL: %w", err)
	}
	return &kv, nil
}

func (kv *KV) Get(k string) ([]byte, bool, error) {
	entry, ok := kv.skiplist.Get(k)
	if ok {
		if entry.Deleted {
			return nil, false, nil
		}
		return entry.Value, true, nil
	}
	for data, err := range kv.sst.Tables() {
		if err != nil {
			return nil, false, err
		}
		for len(data) > 0 {
			tuple, consumed, err := skiplist.DecodeTuple(data)
			if err != nil {
				return nil, false, err
			}
			if tuple.Key == k {
				if tuple.Deleted {
					return nil, false, nil
				}
				return tuple.Value, true, nil
			}
			if tuple.Key > k {
				break
			}
			data = data[consumed:]
		}
	}
	return nil, false, nil
}

func (kv *KV) Put(k string, v []byte) error {
	err := kv.wal.Append(wal.Record{
		Type:  wal.Put,
		Key:   k,
		Value: v,
	})
	if err != nil {
		return err
	}
	kv.skiplist.Put(skiplist.Tuple{Key: k, Value: v, Deleted: false})
	return nil
}

func (kv *KV) Delete(k string, v []byte) error {
	err := kv.wal.Append(wal.Record{
		Type: wal.Delete,
		Key:  k,
	})
	if err != nil {
		return err
	}
	kv.skiplist.Put(skiplist.Tuple{Key: k, Value: nil, Deleted: true})
	return nil
}

func (kv *KV) Close() error {
	skipListByte, err := kv.skiplist.MarshalBinary()
	if err != nil {
		return err
	}
	errSSTManagerClose := kv.sst.Flush(skipListByte)
	if errSSTManagerClose != nil {
		return errSSTManagerClose
	}
	errWALReset := kv.wal.Reset()
	if errWALReset != nil {
		return errWALReset
	}
	return kv.wal.Close()
}
