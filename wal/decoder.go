package wal

import (
	"encoding/binary"
	"hash/crc32"
)

func fromBytes(data []byte) (record Record, n int, err error) {
	if len(data) == 0 {
		return
	}
	switch data[0] {
	case 0:
		return fromDeleteBytes(data)
	case 1:
		return fromPutBytes(data)
	}
	err = ErrUnsupportedOperation
	return
}

func fromPutBytes(b []byte) (record Record, n int, err error) {
	if len(b) < putHeaderLen {
		err = ErrInsufficientBytesRecord
		return
	}
	kLen := binary.BigEndian.Uint32(b[1:5])
	vLen := binary.BigEndian.Uint32(b[5:9])
	bLen := putHeaderLen + kLen + vLen + checkSumLen

	if kLen > MaxKeyLength {
		err = ErrKeyMaxLengthExceeded
		return
	}
	if vLen > MaxValueLength {
		err = ErrValueMaxLengthExceeded
		return
	}
	if len(b) < int(bLen) {
		err = ErrInsufficientBytesRecord
		return
	}
	if crc32.ChecksumIEEE(b[:bLen-checkSumLen]) != binary.BigEndian.Uint32(b[bLen-checkSumLen:bLen]) {
		err = ErrCorruptedRecord
		return
	}

	record = Record{
		Type:  Put,
		Key:   string(b[putHeaderLen : putHeaderLen+kLen]),
		Value: b[putHeaderLen+kLen : putHeaderLen+kLen+vLen],
	}
	n = int(bLen)
	return
}

func fromDeleteBytes(b []byte) (record Record, n int, err error) {
	if len(b) < deleteHeaderLen {
		err = ErrInsufficientBytesRecord
		return
	}
	kLen := binary.BigEndian.Uint32(b[1:5])
	bLen := deleteHeaderLen + kLen + checkSumLen

	if kLen > MaxKeyLength {
		err = ErrKeyMaxLengthExceeded
		return
	}
	if len(b) < int(bLen) {
		err = ErrInsufficientBytesRecord
		return
	}
	if crc32.ChecksumIEEE(b[:bLen-checkSumLen]) != binary.BigEndian.Uint32(b[bLen-checkSumLen:bLen]) {
		err = ErrCorruptedRecord
		return
	}

	record = Record{
		Type: Delete,
		Key:  string(b[deleteHeaderLen : deleteHeaderLen+kLen]),
	}
	n = int(bLen)
	return
}
