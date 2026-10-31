package wal

import "errors"

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
