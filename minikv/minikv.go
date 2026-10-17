package minikv

import (
	"encoding/binary"

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
	scannedEntries := make([]Entry, 0)
	for _, e := range kv.skiplist.Scan(from, to) {
		if e.Entry.Deleted {
			continue
		}
		scannedEntries = append(scannedEntries, Entry{e.Key, e.Entry.Value})
	}
	for b, err := range filemanager.Walk(kv.dataPath) {
		if err != nil {
			return nil, err
		}
		diskEntries := deserializeScan(from, to, b)
		scannedEntries = merge(scannedEntries, diskEntries)
	}
	return scannedEntries, nil
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
	for i+9 < len(b) {
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

func deserializeScan(from, to string, b []byte) []Entry {
	entries := make([]Entry, 0)
	i := 0
	for i+9 < len(b) {
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

		if string(keyByte) >= from && string(keyByte) <= to && isTombStone == 0 {
			e := Entry{
				Key:   string(keyByte),
				Value: valueByte,
			}
			entries = append(entries, e)
		}
		i += 9 + int(keyLength) + int(valueLength)
	}
	return entries
}

func merge(a, b []Entry) []Entry {
	out := make([]Entry, 0, len(a)+len(b))

	i, j := 0, 0
	for i < len(a) && j < len(b) {
		if a[i].Key < b[j].Key {
			out = append(out, a[i])
			i++
		} else {
			out = append(out, b[j])
			j++
		}
	}
	for i < len(a) {
		out = append(out, a[i])
		i++
	}
	for j < len(b) {
		out = append(out, b[j])
		j++
	}
	return out
}
