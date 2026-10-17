package minikv

import (
	"encoding/binary"
	"slices"
	"strings"

	"github.com/amrable/minikv/filemanager"
	"github.com/amrable/minikv/skiplist"
)

type KV struct {
	dataPath string
	skiplist skiplist.List
}

type Entry struct {
	Key   string
	Value []byte
}

type Record struct {
	Value   []byte
	Deleted bool
}

func Open(dataPath string) *KV {
	return &KV{
		dataPath: dataPath,
		skiplist: *skiplist.New(),
	}
}

func (kv *KV) Get(k string) ([]byte, bool, error) {
	entry, ok := kv.skiplist.Get(k)
	if ok {
		if entry.Deleted {
			return nil, false, nil
		}
		return entry.Value, true, nil
	}
	for b, err := range filemanager.Walk(kv.dataPath) {
		if err != nil {
			return nil, false, err
		}
		diskEntry, ok := deserializeGet(k, b)
		if ok {
			if diskEntry.Deleted {
				return nil, false, nil
			}
			return diskEntry.Value, true, nil
		}
	}
	return nil, false, nil
}

func (kv *KV) Scan(from, to string) ([]Entry, error) {
	inMem := make(map[string]Record)
	for _, e := range kv.skiplist.Scan(from, to) {
		inMem[e.Key] = Record{e.Entry.Value, e.Entry.Deleted}
	}
	for b, err := range filemanager.Walk(kv.dataPath) {
		if err != nil {
			return nil, err
		}
		diskEntries := deserializeScan(from, to, b)
		merge(inMem, diskEntries)
	}
	out := make([]Entry, 0, len(inMem))
	for k, v := range inMem {
		if v.Deleted {
			continue
		}
		out = append(out, Entry{k, v.Value})
	}
	slices.SortFunc(out, func(a, b Entry) int {
		return strings.Compare(a.Key, b.Key)
	})
	return out, nil
}

func (kv *KV) Put(k string, v []byte) {
	kv.skiplist.Put(k, v)
}

func (kv *KV) Delete(k string, v []byte) {
	kv.skiplist.Delete(k)
}

func (kv *KV) Close() error {
	return filemanager.Write(kv.dataPath, kv.skiplist.ToBytes())
}

func deserializeGet(key string, b []byte) (skiplist.Entry, bool) {
	i := 0
	for i+9 <= len(b) {
		isTombStone := b[i]
		keyLengthByte := b[i+1 : i+5]
		valueLengthByte := b[i+5 : i+9]

		keyLength := binary.BigEndian.Uint32(keyLengthByte)
		valueLength := binary.BigEndian.Uint32(valueLengthByte)

		keyStartIndex := i + 9
		keyEndIndex := i + 9 + int(keyLength)
		valueEndIndex := i + 9 + int(keyLength) + int(valueLength)

		keyByte := b[keyStartIndex:keyEndIndex]
		valueByte := b[keyEndIndex:valueEndIndex]

		if string(keyByte) == key {
			return skiplist.Entry{
				Value:   valueByte,
				Deleted: isTombStone == 1,
			}, true
		}
		i += 9 + int(keyLength) + int(valueLength)
	}
	return skiplist.Entry{}, false
}

func deserializeScan(from, to string, b []byte) map[string]Record {
	entries := make(map[string]Record)
	i := 0
	for i+9 <= len(b) {
		isTombStone := bool(b[i] != 0)
		keyLengthByte := b[i+1 : i+5]
		valueLengthByte := b[i+5 : i+9]

		keyLength := binary.BigEndian.Uint32(keyLengthByte)
		valueLength := binary.BigEndian.Uint32(valueLengthByte)

		keyStartIndex := i + 9
		keyEndIndex := i + 9 + int(keyLength)
		valueEndIndex := i + 9 + int(keyLength) + int(valueLength)

		keyByte := b[keyStartIndex:keyEndIndex]
		valueByte := b[keyEndIndex:valueEndIndex]

		if string(keyByte) >= from && string(keyByte) <= to {
			entries[string(keyByte)] = Record{Value: valueByte, Deleted: isTombStone}
		}
		i += 9 + int(keyLength) + int(valueLength)
	}
	return entries
}

func merge(a, b map[string]Record) {
	for k, v := range b {
		if _, ok := a[k]; !ok {
			a[k] = v
		}
	}
}
