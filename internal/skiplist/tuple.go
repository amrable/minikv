package skiplist

import (
	"encoding/binary"
	"errors"
	"fmt"
)

type Tuple struct {
	Key     string
	Value   []byte
	Deleted bool
}

// NewTupleFromBytes decodes the first tuple in bytes. Any bytes after that
// tuple are left for the caller to decode as the next tuple.
func NewTupleFromBytes(bytes []byte) (Tuple, error) {
	t, _, err := DecodeTuple(bytes)
	return t, err
}

// DecodeTuple decodes the first tuple and returns the number of bytes consumed.
func DecodeTuple(bytes []byte) (Tuple, int, error) {
	t := Tuple{}
	if len(bytes) < 9 {
		return t, 0, errors.New("decode tuple: header is incomplete")
	}
	if bytes[0] > 1 {
		return t, 0, fmt.Errorf("decode tuple: invalid deleted flag %d", bytes[0])
	}
	keyLength := binary.BigEndian.Uint32(bytes[1:5])
	valueLength := binary.BigEndian.Uint32(bytes[5:9])
	totalLength := uint64(9) + uint64(keyLength) + uint64(valueLength)
	if totalLength > uint64(len(bytes)) {
		return t, 0, errors.New("decode tuple: payload is incomplete")
	}

	t.Deleted = bytes[0] == 1
	keyEnd := 9 + int(keyLength)
	valueEnd := keyEnd + int(valueLength)
	t.Key = string(bytes[9:keyEnd])
	t.Value = append([]byte(nil), bytes[keyEnd:valueEnd]...)
	return t, int(totalLength), nil
}

// MarshalBinary encodes a tuple as a flag, two big-endian uint32 lengths, then the
// key and value bytes.
func (t Tuple) MarshalBinary() ([]byte, error) {
	keyLength := len(t.Key)
	valueLength := len(t.Value)
	if uint64(keyLength) > uint64(^uint32(0)) || uint64(valueLength) > uint64(^uint32(0)) {
		return nil, errors.New("encode tuple: key or value exceeds uint32 length limit")
	}

	flag := byte(0)
	if t.Deleted {
		flag = 1
	}
	b := make([]byte, 0, 9+keyLength+valueLength)
	b = append(b, flag)
	b = binary.BigEndian.AppendUint32(b, uint32(keyLength))
	b = binary.BigEndian.AppendUint32(b, uint32(valueLength))
	b = append(b, t.Key...)
	b = append(b, t.Value...)
	return b, nil
}
