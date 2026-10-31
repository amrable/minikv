package minikv

import (
	"fmt"

	"github.com/amrable/minikv/internal/skiplist"
	"github.com/amrable/minikv/internal/wal"
)

const MAX_INMEM_BYTES = 1024

type SSTManager interface {
	Get(string) ([]byte, bool, error)
	Close() error
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
	Key   string
	Value []byte
}

type Record struct {
	Value   []byte
	Deleted bool
}

func Open(dirPath string) (*KV, error) {
	// todo: init sstmanager
	w, err := wal.Open(dirPath)
	if err != nil {
		return nil, err
	}
	kv := KV{
		wal:      w,
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
	return kv.sst.Get(k)
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
	errSSTManagerClose := kv.sst.Close()
	if errSSTManagerClose != nil {
		return errSSTManagerClose
	}
	errWALReset := kv.wal.Reset()
	if errWALReset != nil {
		return errWALReset
	}
	return kv.wal.Close()
}
