package wal

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type Operation string

const (
	MaxKeyLength   = 1 << 20
	MaxValueLength = 1 << 20
)

const (
	Put    = Operation("PUT")
	Delete = Operation("DELETE")
)

type Record struct {
	Type  Operation
	Key   string
	Value []byte
}

var (
	ErrUnsupportedOperation   = errors.New("unsupported wal operation")
	ErrKeyMaxLengthExceeded   = errors.New("key length must fit into 1 MiB")
	ErrValueMaxLengthExceeded = errors.New("value length must fit into 1 MiB")

	ErrCorruptedRecord         = errors.New("corrupted record")
	ErrInsufficientBytesRecord = errors.New("insufficient bytes record")
)

type WAL struct {
	path string
	file *os.File
	mu   sync.Mutex
}

func Open(path string) (*WAL, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}
	dir, err := os.Open(filepath.Dir(path))
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("open WAL directory: %w", err)
	}

	err = dir.Sync()
	closeErr := dir.Close()
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("sync WAL directory: %w", err)
	}
	if closeErr != nil {
		f.Close()
		return nil, fmt.Errorf("close WAL directory: %w", closeErr)
	}
	return &WAL{
		path: path,
		file: f,
	}, nil
}

func (w *WAL) Append(record Record) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	recordBytes, err := toBytes(record)
	if err != nil {
		return err
	}
	if _, err := w.file.Write(recordBytes); err != nil {
		return fmt.Errorf("write WAL: %w", err)
	}

	if err := w.file.Sync(); err != nil {
		return fmt.Errorf("sync WAL: %w", err)
	}

	return nil
}

func (w *WAL) Replay(apply func(Record) error) error {
	if apply == nil {
		return nil
	}
	w.mu.Lock()
	data, err := os.ReadFile(w.path)
	if err != nil {
		w.mu.Unlock()
		return err
	}
	w.mu.Unlock()
	i := 0
	for i < len(data) {
		r, n, err := fromBytes(data[i:])
		if err != nil {
			if errors.Is(err, ErrInsufficientBytesRecord) {
				return nil
			}
			return err
		}
		if err := apply(r); err != nil {
			return err
		}
		i += n
	}
	return nil
}

func (w *WAL) Reset() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if err := w.file.Truncate(0); err != nil {
		return err
	}
	if _, err := w.file.Seek(0, 0); err != nil {
		return err
	}
	return w.file.Sync()
}

func (w *WAL) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.file.Close()
}
