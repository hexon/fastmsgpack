package internal

import (
	"encoding/binary"
	"fmt"
	"math"
)

func DecodeBytesToUint(data []byte) (uint, bool) {
	switch len(data) {
	case 1:
		return uint(data[0]), true
	case 2:
		return uint(binary.BigEndian.Uint16(data)), true
	case 4:
		return uint(binary.BigEndian.Uint32(data)), true
	case 8:
		return uint(binary.BigEndian.Uint64(data)), true
	default:
		return 0, false
	}
}

func AppendMapLen(dst []byte, l int) ([]byte, error) {
	if l < 16 {
		return append(dst, 0x80|byte(l)), nil
	} else if l < math.MaxUint16 {
		return append(dst, 0xde, byte(l>>8), byte(l)), nil
	} else if l < math.MaxUint32 {
		return append(dst, 0xdf, byte(l>>24), byte(l>>16), byte(l>>8), byte(l)), nil
	} else {
		return nil, fmt.Errorf("fastmsgpack.Encode: map too long to encode (len %d)", l)
	}
}

func AppendArrayLen(dst []byte, l int) ([]byte, error) {
	if l < 16 {
		return append(dst, 0x90|byte(l)), nil
	} else if l < math.MaxUint16 {
		return append(dst, 0xdc, byte(l>>8), byte(l)), nil
	} else if l < math.MaxUint32 {
		return append(dst, 0xdd, byte(l>>24), byte(l>>16), byte(l>>8), byte(l)), nil
	} else {
		return nil, fmt.Errorf("fastmsgpack.Encode: map too long to encode (len %d)", l)
	}
}
