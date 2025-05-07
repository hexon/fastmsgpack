package internal

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
)

var (
	ErrNotExtension = errors.New("data is not an extension")
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

func DecodeExtensionHeader(data []byte) (int8, []byte, error) {
	switch data[0] {
	case 0xd4:
		if len(data) < 3 {
			return 0, nil, ErrShortInput
		}
		return int8(data[1]), data[2:3], nil
	case 0xd5:
		if len(data) < 4 {
			return 0, nil, ErrShortInput
		}
		return int8(data[1]), data[2:4], nil
	case 0xd6:
		if len(data) < 6 {
			return 0, nil, ErrShortInput
		}
		return int8(data[1]), data[2:6], nil
	case 0xd7:
		if len(data) < 10 {
			return 0, nil, ErrShortInput
		}
		return int8(data[1]), data[2:10], nil
	case 0xd8:
		if len(data) < 18 {
			return 0, nil, ErrShortInput
		}
		return int8(data[1]), data[2:18], nil
	case 0xc7:
		if len(data) < 3 {
			return 0, nil, ErrShortInput
		}
		s := int(data[1]) + 3
		if len(data) < s {
			return 0, nil, ErrShortInput
		}
		return int8(data[2]), data[3:s], nil
	case 0xc8:
		if len(data) < 4 {
			return 0, nil, ErrShortInput
		}
		s := int(binary.BigEndian.Uint16(data[1:3])) + 4
		if len(data) < s {
			return 0, nil, ErrShortInput
		}
		return int8(data[3]), data[4:s], nil
	case 0xc9:
		if len(data) < 6 {
			return 0, nil, ErrShortInput
		}
		s := int(binary.BigEndian.Uint32(data[1:5])) + 6
		if len(data) < s {
			return 0, nil, ErrShortInput
		}
		return int8(data[5]), data[6:s], nil
	default:
		return 0, nil, ErrNotExtension
	}
}

// DecodeLengthPrefixExtension returns the number of bytes to be skipped past to get to the real entry.
func DecodeLengthPrefixExtension(data []byte) int {
	switch data[0] {
	case 0xd4, 0xd5, 0xd6, 0xd7, 0xd8:
		if data[1] != 17 {
			return 0
		}
		return 2
	case 0xc7:
		if data[2] != 17 {
			return 0
		}
		return 3
	case 0xc8:
		if data[3] != 17 {
			return 0
		}
		return 4
	case 0xc9:
		if data[5] != 17 {
			return 0
		}
		return 6
	default:
		return 0
	}
}

func DecodeUnwrappedMapLen(data []byte) (int, int, bool) {
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

func DecodeUnwrappedArrayLen(data []byte) (int, int, bool) {
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
