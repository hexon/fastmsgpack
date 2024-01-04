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

func DecodeMapLen(data []byte) (int, int, bool) {
	switch data[0] {
	case 0xde:
		return int(binary.BigEndian.Uint16(data[1:3])), 3, true
	case 0xdf:
		return int(binary.BigEndian.Uint32(data[1:5])), 5, true
	default:
		if data[0]&0b11110000 != 0b10000000 {
			return 0, 0, false
		}
		return int(data[0] & 0b00001111), 1, true
	}
}

func DecodeArrayLen(data []byte) (int, int, bool) {
	switch data[0] {
	case 0xdc:
		return int(binary.BigEndian.Uint16(data[1:3])), 3, true
	case 0xdd:
		return int(binary.BigEndian.Uint32(data[1:5])), 5, true
	default:
		if data[0]&0b11110000 != 0b10010000 {
			return 0, 0, false
		}
		return int(data[0] & 0b00001111), 1, true
	}
}
