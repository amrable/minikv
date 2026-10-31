package wal

import (
	"encoding/binary"
	"hash/crc32"
)

const (
	deleteHeaderLen = 5
	checkSumLen     = 4
	putHeaderLen    = 9
)

func toBytes(r Record) ([]byte, error) {
	switch r.Type {
	case Put:
		return toPutBytes(r)
	case Delete:
		return toDeleteBytes(r)
	}
	return nil, ErrUnsupportedOperation
}

func toPutBytes(r Record) ([]byte, error) {
	kLen := len(r.Key)
	vLen := len(r.Value)
	bLen := putHeaderLen + kLen + vLen + checkSumLen

	if kLen > MaxKeyLength {
		return nil, ErrKeyMaxLengthExceeded
	}
	if vLen > MaxValueLength {
		return nil, ErrValueMaxLengthExceeded
	}

	b := make([]byte, bLen)
	b[0] = 1
	binary.BigEndian.PutUint32(b[1:5], uint32(kLen))
	binary.BigEndian.PutUint32(b[5:9], uint32(vLen))
	copy(b[putHeaderLen:putHeaderLen+kLen], []byte(r.Key))
	copy(b[putHeaderLen+kLen:putHeaderLen+kLen+vLen], []byte(r.Value))
	binary.BigEndian.PutUint32(b[putHeaderLen+kLen+vLen:], crc32.ChecksumIEEE(b[:putHeaderLen+kLen+vLen]))
	return b, nil
}

func toDeleteBytes(r Record) ([]byte, error) {
	if len(r.Value) > 0 {
		return nil, ErrCorruptedRecord
	}
	kLen := len(r.Key)
	bLen := deleteHeaderLen + kLen + checkSumLen
	if kLen > MaxKeyLength {
		return nil, ErrKeyMaxLengthExceeded
	}

	b := make([]byte, bLen)

	binary.BigEndian.PutUint32(b[1:5], uint32(kLen))
	copy(b[deleteHeaderLen:deleteHeaderLen+kLen], []byte(r.Key))
	binary.BigEndian.PutUint32(b[deleteHeaderLen+kLen:], crc32.ChecksumIEEE(b[:deleteHeaderLen+kLen]))

	return b, nil
}
