// Code generated by internal/codegen. DO NOT EDIT.

package internal

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"time"
)

func ValueLength(data []byte) (int, error) {
	if len(data) < 1 {
		return 0, ErrShortInput
	}
	if data[0] < 0xc0 {
		if data[0] <= 0x7f {
			return 1, nil
		}
		if data[0] <= 0x8f {
			return SkipMultiple(data, 1, 2*(int(data[0]&0b00001111)))
		}
		if data[0] <= 0x9f {
			return SkipMultiple(data, 1, int(data[0]&0b00001111))
		}
		s := int(data[0]&0b00011111) + 1
		return s, nil
	}
	if data[0] >= 0xe0 {
		return 1, nil
	}
	switch data[0] {
	case 0xc0:
		return 1, nil
	case 0xc2:
		return 1, nil
	case 0xc3:
		return 1, nil
	case 0xc4:
		if len(data) < 2 {
			return 0, ErrShortInput
		}
		s := int(data[1]) + 2
		return s, nil
	case 0xc5:
		if len(data) < 3 {
			return 0, ErrShortInput
		}
		s := int(binary.BigEndian.Uint16(data[1:3])) + 3
		return s, nil
	case 0xc6:
		if len(data) < 5 {
			return 0, ErrShortInput
		}
		s := int(binary.BigEndian.Uint32(data[1:5])) + 5
		return s, nil
	case 0xc7:
		if len(data) < 3 {
			return 0, ErrShortInput
		}
		s := int(data[1]) + 3
		return s, nil
	case 0xc8:
		if len(data) < 4 {
			return 0, ErrShortInput
		}
		s := int(binary.BigEndian.Uint16(data[1:3])) + 4
		return s, nil
	case 0xc9:
		if len(data) < 6 {
			return 0, ErrShortInput
		}
		s := int(binary.BigEndian.Uint32(data[1:5])) + 6
		return s, nil
	case 0xca:
		return 5, nil
	case 0xcb:
		return 9, nil
	case 0xcc:
		return 2, nil
	case 0xcd:
		return 3, nil
	case 0xce:
		return 5, nil
	case 0xcf:
		return 9, nil
	case 0xd0:
		return 2, nil
	case 0xd1:
		return 3, nil
	case 0xd2:
		return 5, nil
	case 0xd3:
		return 9, nil
	case 0xd4:
		return 3, nil
	case 0xd5:
		return 4, nil
	case 0xd6:
		return 6, nil
	case 0xd7:
		return 10, nil
	case 0xd8:
		return 18, nil
	case 0xd9:
		if len(data) < 2 {
			return 0, ErrShortInput
		}
		s := int(data[1]) + 2
		return s, nil
	case 0xda:
		if len(data) < 3 {
			return 0, ErrShortInput
		}
		s := int(binary.BigEndian.Uint16(data[1:3])) + 3
		return s, nil
	case 0xdb:
		if len(data) < 5 {
			return 0, ErrShortInput
		}
		s := int(binary.BigEndian.Uint32(data[1:5])) + 5
		return s, nil
	case 0xdc:
		return SkipMultiple(data, 3, int(binary.BigEndian.Uint16(data[1:3])))
	case 0xdd:
		return SkipMultiple(data, 5, int(binary.BigEndian.Uint32(data[1:5])))
	case 0xde:
		return SkipMultiple(data, 3, 2*(int(binary.BigEndian.Uint16(data[1:3]))))
	case 0xdf:
		return SkipMultiple(data, 5, 2*(int(binary.BigEndian.Uint32(data[1:5]))))
	}
	return 0, errors.New("unexpected 0xc1")
}

func DescribeValue(data []byte) string {
	if len(data) < 1 {
		return "empty input"
	}
	if data[0] < 0xc0 {
		if data[0] <= 0x7f {
			return fmt.Sprintf("positive fixint (%d)", int(data[0]))
		}
		if data[0] <= 0x8f {
			return fmt.Sprintf("fixmap (%d entries)", int(data[0]&0b00001111))
		}
		if data[0] <= 0x9f {
			return fmt.Sprintf("fixarray (%d entries)", int(data[0]&0b00001111))
		}
		s := int(data[0]&0b00011111) + 1
		if len(data) < s {
			return "truncated fixstr"
		}
		return fmt.Sprintf("fixstr (%q)", UnsafeStringCast(data[1:s]))
	}
	if data[0] >= 0xe0 {
		return fmt.Sprintf("negative fixint (%d)", int(int8(data[0])))
	}
	switch data[0] {
	case 0xc0:
		return "nil"
	case 0xc2:
		return "false"
	case 0xc3:
		return "true"
	case 0xc4:
		if len(data) < 2 {
			return "truncated bin 8"
		}
		s := int(data[1]) + 2
		if len(data) < s {
			return "truncated bin 8"
		}
		return "bin 8"
	case 0xc5:
		if len(data) < 3 {
			return "truncated bin 16"
		}
		s := int(binary.BigEndian.Uint16(data[1:3])) + 3
		if len(data) < s {
			return "truncated bin 16"
		}
		return "bin 16"
	case 0xc6:
		if len(data) < 5 {
			return "truncated bin 32"
		}
		s := int(binary.BigEndian.Uint32(data[1:5])) + 5
		if len(data) < s {
			return "truncated bin 32"
		}
		return "bin 32"
	case 0xc7:
		if len(data) < 3 {
			return "truncated ext 8"
		}
		s := int(data[1]) + 3
		if len(data) < s {
			return "truncated ext 8"
		}
		return fmt.Sprintf("ext 8 (type %d, %d bytes)", int8(data[2]), len(data[3:s]))
	case 0xc8:
		if len(data) < 4 {
			return "truncated ext 16"
		}
		s := int(binary.BigEndian.Uint16(data[1:3])) + 4
		if len(data) < s {
			return "truncated ext 16"
		}
		return fmt.Sprintf("ext 16 (type %d, %d bytes)", int8(data[3]), len(data[4:s]))
	case 0xc9:
		if len(data) < 6 {
			return "truncated ext 32"
		}
		s := int(binary.BigEndian.Uint32(data[1:5])) + 6
		if len(data) < s {
			return "truncated ext 32"
		}
		return fmt.Sprintf("ext 32 (type %d, %d bytes)", int8(data[5]), len(data[6:s]))
	case 0xca:
		if len(data) < 5 {
			return "truncated float 32"
		}
		return fmt.Sprintf("float 32 (%f)", math.Float32frombits(binary.BigEndian.Uint32(data[1:5])))
	case 0xcb:
		if len(data) < 9 {
			return "truncated float 64"
		}
		return fmt.Sprintf("float 64 (%f)", math.Float64frombits(binary.BigEndian.Uint64(data[1:9])))
	case 0xcc:
		if len(data) < 2 {
			return "truncated uint 8"
		}
		return fmt.Sprintf("uint 8 (%d)", int(data[1]))
	case 0xcd:
		if len(data) < 3 {
			return "truncated uint 16"
		}
		return fmt.Sprintf("uint 16 (%d)", int(binary.BigEndian.Uint16(data[1:3])))
	case 0xce:
		if len(data) < 5 {
			return "truncated uint 32"
		}
		return fmt.Sprintf("uint 32 (%d)", int(binary.BigEndian.Uint32(data[1:5])))
	case 0xcf:
		if len(data) < 9 {
			return "truncated uint 64"
		}
		return fmt.Sprintf("uint 64 (%d)", int(binary.BigEndian.Uint64(data[1:9])))
	case 0xd0:
		if len(data) < 2 {
			return "truncated int 8"
		}
		return fmt.Sprintf("int 8 (%d)", int(int8(data[1])))
	case 0xd1:
		if len(data) < 3 {
			return "truncated int 16"
		}
		return fmt.Sprintf("int 16 (%d)", int(int16(binary.BigEndian.Uint16(data[1:3]))))
	case 0xd2:
		if len(data) < 5 {
			return "truncated int 32"
		}
		return fmt.Sprintf("int 32 (%d)", int(int32(binary.BigEndian.Uint32(data[1:5]))))
	case 0xd3:
		if len(data) < 9 {
			return "truncated int 64"
		}
		return fmt.Sprintf("int 64 (%d)", int(int64(binary.BigEndian.Uint64(data[1:9]))))
	case 0xd4:
		if len(data) < 3 {
			return "truncated fixext 1"
		}
		return fmt.Sprintf("fixext 1 (type %d, %d bytes)", int8(data[1]), len(data[2:3]))
	case 0xd5:
		if len(data) < 4 {
			return "truncated fixext 2"
		}
		return fmt.Sprintf("fixext 2 (type %d, %d bytes)", int8(data[1]), len(data[2:4]))
	case 0xd6:
		if len(data) < 6 {
			return "truncated fixext 4"
		}
		return fmt.Sprintf("fixext 4 (type %d, %d bytes)", int8(data[1]), len(data[2:6]))
	case 0xd7:
		if len(data) < 10 {
			return "truncated fixext 8"
		}
		return fmt.Sprintf("fixext 8 (type %d, %d bytes)", int8(data[1]), len(data[2:10]))
	case 0xd8:
		if len(data) < 18 {
			return "truncated fixext 16"
		}
		return fmt.Sprintf("fixext 16 (type %d, %d bytes)", int8(data[1]), len(data[2:18]))
	case 0xd9:
		if len(data) < 2 {
			return "truncated str 8"
		}
		s := int(data[1]) + 2
		if len(data) < s {
			return "truncated str 8"
		}
		return fmt.Sprintf("str 8 (%q)", UnsafeStringCast(data[2:s]))
	case 0xda:
		if len(data) < 3 {
			return "truncated str 16"
		}
		s := int(binary.BigEndian.Uint16(data[1:3])) + 3
		if len(data) < s {
			return "truncated str 16"
		}
		return fmt.Sprintf("str 16 (%q)", UnsafeStringCast(data[3:s]))
	case 0xdb:
		if len(data) < 5 {
			return "truncated str 32"
		}
		s := int(binary.BigEndian.Uint32(data[1:5])) + 5
		if len(data) < s {
			return "truncated str 32"
		}
		return fmt.Sprintf("str 32 (%q)", UnsafeStringCast(data[5:s]))
	case 0xdc:
		if len(data) < 3 {
			return "truncated array 16"
		}
		return fmt.Sprintf("array 16 (%d entries)", int(binary.BigEndian.Uint16(data[1:3])))
	case 0xdd:
		if len(data) < 5 {
			return "truncated array 32"
		}
		return fmt.Sprintf("array 32 (%d entries)", int(binary.BigEndian.Uint32(data[1:5])))
	case 0xde:
		if len(data) < 3 {
			return "truncated map 16"
		}
		return fmt.Sprintf("map 16 (%d entries)", int(binary.BigEndian.Uint16(data[1:3])))
	case 0xdf:
		if len(data) < 5 {
			return "truncated map 32"
		}
		return fmt.Sprintf("map 32 (%d entries)", int(binary.BigEndian.Uint32(data[1:5])))
	}
	return "0xc1"
}

func DecodeInt(data []byte) (int, int, error) {
	if len(data) < 1 {
		return 0, 0, ErrShortInput
	}
	if data[0] <= 0x7f {
		return int(data[0]), 1, nil
	}
	if data[0] >= 0xe0 {
		return int(int8(data[0])), 1, nil
	}
	switch data[0] {
	case 0xca:
		if len(data) < 5 {
			return 0, 0, ErrShortInput
		}
		return int(math.Float32frombits(binary.BigEndian.Uint32(data[1:5]))), 5, nil
	case 0xcb:
		if len(data) < 9 {
			return 0, 0, ErrShortInput
		}
		return int(math.Float64frombits(binary.BigEndian.Uint64(data[1:9]))), 9, nil
	case 0xcc:
		if len(data) < 2 {
			return 0, 0, ErrShortInput
		}
		return int(data[1]), 2, nil
	case 0xcd:
		if len(data) < 3 {
			return 0, 0, ErrShortInput
		}
		return int(binary.BigEndian.Uint16(data[1:3])), 3, nil
	case 0xce:
		if len(data) < 5 {
			return 0, 0, ErrShortInput
		}
		return int(binary.BigEndian.Uint32(data[1:5])), 5, nil
	case 0xcf:
		if len(data) < 9 {
			return 0, 0, ErrShortInput
		}
		return int(binary.BigEndian.Uint64(data[1:9])), 9, nil
	case 0xd0:
		if len(data) < 2 {
			return 0, 0, ErrShortInput
		}
		return int(int8(data[1])), 2, nil
	case 0xd1:
		if len(data) < 3 {
			return 0, 0, ErrShortInput
		}
		return int(int16(binary.BigEndian.Uint16(data[1:3]))), 3, nil
	case 0xd2:
		if len(data) < 5 {
			return 0, 0, ErrShortInput
		}
		return int(int32(binary.BigEndian.Uint32(data[1:5]))), 5, nil
	case 0xd3:
		if len(data) < 9 {
			return 0, 0, ErrShortInput
		}
		return int(int64(binary.BigEndian.Uint64(data[1:9]))), 9, nil
	}

	// Try extension decoding in case of a length-prefixed entry (#17)
	switch data[0] {
	case 0xc7:
		if len(data) < 3 {
			return 0, 0, ErrShortInput
		}
		s := int(data[1]) + 3
		if len(data) < s {
			return 0, 0, ErrShortInput
		}
		ret, err := decodeInt_ext(data[3:s], int8(data[2]))
		return ret, s, err
	case 0xc8:
		if len(data) < 4 {
			return 0, 0, ErrShortInput
		}
		s := int(binary.BigEndian.Uint16(data[1:3])) + 4
		if len(data) < s {
			return 0, 0, ErrShortInput
		}
		ret, err := decodeInt_ext(data[4:s], int8(data[3]))
		return ret, s, err
	case 0xc9:
		if len(data) < 6 {
			return 0, 0, ErrShortInput
		}
		s := int(binary.BigEndian.Uint32(data[1:5])) + 6
		if len(data) < s {
			return 0, 0, ErrShortInput
		}
		ret, err := decodeInt_ext(data[6:s], int8(data[5]))
		return ret, s, err
	case 0xd4:
		if len(data) < 3 {
			return 0, 0, ErrShortInput
		}
		ret, err := decodeInt_ext(data[2:3], int8(data[1]))
		return ret, 3, err
	case 0xd5:
		if len(data) < 4 {
			return 0, 0, ErrShortInput
		}
		ret, err := decodeInt_ext(data[2:4], int8(data[1]))
		return ret, 4, err
	case 0xd6:
		if len(data) < 6 {
			return 0, 0, ErrShortInput
		}
		ret, err := decodeInt_ext(data[2:6], int8(data[1]))
		return ret, 6, err
	case 0xd7:
		if len(data) < 10 {
			return 0, 0, ErrShortInput
		}
		ret, err := decodeInt_ext(data[2:10], int8(data[1]))
		return ret, 10, err
	case 0xd8:
		if len(data) < 18 {
			return 0, 0, ErrShortInput
		}
		ret, err := decodeInt_ext(data[2:18], int8(data[1]))
		return ret, 18, err
	}
	return 0, 0, errors.New("unexpected " + DescribeValue(data) + " when expecting int")
}

func DecodeFloat32(data []byte) (float32, int, error) {
	if len(data) < 1 {
		return 0, 0, ErrShortInput
	}
	switch data[0] {
	case 0xca:
		if len(data) < 5 {
			return 0, 0, ErrShortInput
		}
		return math.Float32frombits(binary.BigEndian.Uint32(data[1:5])), 5, nil
	case 0xcb:
		if len(data) < 9 {
			return 0, 0, ErrShortInput
		}
		return float32(math.Float64frombits(binary.BigEndian.Uint64(data[1:9]))), 9, nil
	case 0xcc:
		if len(data) < 2 {
			return 0, 0, ErrShortInput
		}
		return float32(int(data[1])), 2, nil
	case 0xcd:
		if len(data) < 3 {
			return 0, 0, ErrShortInput
		}
		return float32(int(binary.BigEndian.Uint16(data[1:3]))), 3, nil
	case 0xce:
		if len(data) < 5 {
			return 0, 0, ErrShortInput
		}
		return float32(int(binary.BigEndian.Uint32(data[1:5]))), 5, nil
	case 0xcf:
		if len(data) < 9 {
			return 0, 0, ErrShortInput
		}
		return float32(int(binary.BigEndian.Uint64(data[1:9]))), 9, nil
	case 0xd0:
		if len(data) < 2 {
			return 0, 0, ErrShortInput
		}
		return float32(int(int8(data[1]))), 2, nil
	case 0xd1:
		if len(data) < 3 {
			return 0, 0, ErrShortInput
		}
		return float32(int(int16(binary.BigEndian.Uint16(data[1:3])))), 3, nil
	case 0xd2:
		if len(data) < 5 {
			return 0, 0, ErrShortInput
		}
		return float32(int(int32(binary.BigEndian.Uint32(data[1:5])))), 5, nil
	case 0xd3:
		if len(data) < 9 {
			return 0, 0, ErrShortInput
		}
		return float32(int(int64(binary.BigEndian.Uint64(data[1:9])))), 9, nil
	}

	// Try extension decoding in case of a length-prefixed entry (#17)
	switch data[0] {
	case 0xc7:
		if len(data) < 3 {
			return 0, 0, ErrShortInput
		}
		s := int(data[1]) + 3
		if len(data) < s {
			return 0, 0, ErrShortInput
		}
		ret, err := decodeFloat32_ext(data[3:s], int8(data[2]))
		return ret, s, err
	case 0xc8:
		if len(data) < 4 {
			return 0, 0, ErrShortInput
		}
		s := int(binary.BigEndian.Uint16(data[1:3])) + 4
		if len(data) < s {
			return 0, 0, ErrShortInput
		}
		ret, err := decodeFloat32_ext(data[4:s], int8(data[3]))
		return ret, s, err
	case 0xc9:
		if len(data) < 6 {
			return 0, 0, ErrShortInput
		}
		s := int(binary.BigEndian.Uint32(data[1:5])) + 6
		if len(data) < s {
			return 0, 0, ErrShortInput
		}
		ret, err := decodeFloat32_ext(data[6:s], int8(data[5]))
		return ret, s, err
	case 0xd4:
		if len(data) < 3 {
			return 0, 0, ErrShortInput
		}
		ret, err := decodeFloat32_ext(data[2:3], int8(data[1]))
		return ret, 3, err
	case 0xd5:
		if len(data) < 4 {
			return 0, 0, ErrShortInput
		}
		ret, err := decodeFloat32_ext(data[2:4], int8(data[1]))
		return ret, 4, err
	case 0xd6:
		if len(data) < 6 {
			return 0, 0, ErrShortInput
		}
		ret, err := decodeFloat32_ext(data[2:6], int8(data[1]))
		return ret, 6, err
	case 0xd7:
		if len(data) < 10 {
			return 0, 0, ErrShortInput
		}
		ret, err := decodeFloat32_ext(data[2:10], int8(data[1]))
		return ret, 10, err
	case 0xd8:
		if len(data) < 18 {
			return 0, 0, ErrShortInput
		}
		ret, err := decodeFloat32_ext(data[2:18], int8(data[1]))
		return ret, 18, err
	}
	return 0, 0, errors.New("unexpected " + DescribeValue(data) + " when expecting float32")
}

func DecodeFloat64(data []byte) (float64, int, error) {
	if len(data) < 1 {
		return 0, 0, ErrShortInput
	}
	switch data[0] {
	case 0xca:
		if len(data) < 5 {
			return 0, 0, ErrShortInput
		}
		return float64(math.Float32frombits(binary.BigEndian.Uint32(data[1:5]))), 5, nil
	case 0xcb:
		if len(data) < 9 {
			return 0, 0, ErrShortInput
		}
		return math.Float64frombits(binary.BigEndian.Uint64(data[1:9])), 9, nil
	case 0xcc:
		if len(data) < 2 {
			return 0, 0, ErrShortInput
		}
		return float64(int(data[1])), 2, nil
	case 0xcd:
		if len(data) < 3 {
			return 0, 0, ErrShortInput
		}
		return float64(int(binary.BigEndian.Uint16(data[1:3]))), 3, nil
	case 0xce:
		if len(data) < 5 {
			return 0, 0, ErrShortInput
		}
		return float64(int(binary.BigEndian.Uint32(data[1:5]))), 5, nil
	case 0xcf:
		if len(data) < 9 {
			return 0, 0, ErrShortInput
		}
		return float64(int(binary.BigEndian.Uint64(data[1:9]))), 9, nil
	case 0xd0:
		if len(data) < 2 {
			return 0, 0, ErrShortInput
		}
		return float64(int(int8(data[1]))), 2, nil
	case 0xd1:
		if len(data) < 3 {
			return 0, 0, ErrShortInput
		}
		return float64(int(int16(binary.BigEndian.Uint16(data[1:3])))), 3, nil
	case 0xd2:
		if len(data) < 5 {
			return 0, 0, ErrShortInput
		}
		return float64(int(int32(binary.BigEndian.Uint32(data[1:5])))), 5, nil
	case 0xd3:
		if len(data) < 9 {
			return 0, 0, ErrShortInput
		}
		return float64(int(int64(binary.BigEndian.Uint64(data[1:9])))), 9, nil
	}

	// Try extension decoding in case of a length-prefixed entry (#17)
	switch data[0] {
	case 0xc7:
		if len(data) < 3 {
			return 0, 0, ErrShortInput
		}
		s := int(data[1]) + 3
		if len(data) < s {
			return 0, 0, ErrShortInput
		}
		ret, err := decodeFloat64_ext(data[3:s], int8(data[2]))
		return ret, s, err
	case 0xc8:
		if len(data) < 4 {
			return 0, 0, ErrShortInput
		}
		s := int(binary.BigEndian.Uint16(data[1:3])) + 4
		if len(data) < s {
			return 0, 0, ErrShortInput
		}
		ret, err := decodeFloat64_ext(data[4:s], int8(data[3]))
		return ret, s, err
	case 0xc9:
		if len(data) < 6 {
			return 0, 0, ErrShortInput
		}
		s := int(binary.BigEndian.Uint32(data[1:5])) + 6
		if len(data) < s {
			return 0, 0, ErrShortInput
		}
		ret, err := decodeFloat64_ext(data[6:s], int8(data[5]))
		return ret, s, err
	case 0xd4:
		if len(data) < 3 {
			return 0, 0, ErrShortInput
		}
		ret, err := decodeFloat64_ext(data[2:3], int8(data[1]))
		return ret, 3, err
	case 0xd5:
		if len(data) < 4 {
			return 0, 0, ErrShortInput
		}
		ret, err := decodeFloat64_ext(data[2:4], int8(data[1]))
		return ret, 4, err
	case 0xd6:
		if len(data) < 6 {
			return 0, 0, ErrShortInput
		}
		ret, err := decodeFloat64_ext(data[2:6], int8(data[1]))
		return ret, 6, err
	case 0xd7:
		if len(data) < 10 {
			return 0, 0, ErrShortInput
		}
		ret, err := decodeFloat64_ext(data[2:10], int8(data[1]))
		return ret, 10, err
	case 0xd8:
		if len(data) < 18 {
			return 0, 0, ErrShortInput
		}
		ret, err := decodeFloat64_ext(data[2:18], int8(data[1]))
		return ret, 18, err
	}
	return 0, 0, errors.New("unexpected " + DescribeValue(data) + " when expecting float64")
}

func DecodeBool(data []byte) (bool, int, error) {
	if len(data) < 1 {
		return false, 0, ErrShortInput
	}
	switch data[0] {
	case 0xc2:
		return false, 1, nil
	case 0xc3:
		return true, 1, nil
	}

	// Try extension decoding in case of a length-prefixed entry (#17)
	switch data[0] {
	case 0xc7:
		if len(data) < 3 {
			return false, 0, ErrShortInput
		}
		s := int(data[1]) + 3
		if len(data) < s {
			return false, 0, ErrShortInput
		}
		ret, err := decodeBool_ext(data[3:s], int8(data[2]))
		return ret, s, err
	case 0xc8:
		if len(data) < 4 {
			return false, 0, ErrShortInput
		}
		s := int(binary.BigEndian.Uint16(data[1:3])) + 4
		if len(data) < s {
			return false, 0, ErrShortInput
		}
		ret, err := decodeBool_ext(data[4:s], int8(data[3]))
		return ret, s, err
	case 0xc9:
		if len(data) < 6 {
			return false, 0, ErrShortInput
		}
		s := int(binary.BigEndian.Uint32(data[1:5])) + 6
		if len(data) < s {
			return false, 0, ErrShortInput
		}
		ret, err := decodeBool_ext(data[6:s], int8(data[5]))
		return ret, s, err
	case 0xd4:
		if len(data) < 3 {
			return false, 0, ErrShortInput
		}
		ret, err := decodeBool_ext(data[2:3], int8(data[1]))
		return ret, 3, err
	case 0xd5:
		if len(data) < 4 {
			return false, 0, ErrShortInput
		}
		ret, err := decodeBool_ext(data[2:4], int8(data[1]))
		return ret, 4, err
	case 0xd6:
		if len(data) < 6 {
			return false, 0, ErrShortInput
		}
		ret, err := decodeBool_ext(data[2:6], int8(data[1]))
		return ret, 6, err
	case 0xd7:
		if len(data) < 10 {
			return false, 0, ErrShortInput
		}
		ret, err := decodeBool_ext(data[2:10], int8(data[1]))
		return ret, 10, err
	case 0xd8:
		if len(data) < 18 {
			return false, 0, ErrShortInput
		}
		ret, err := decodeBool_ext(data[2:18], int8(data[1]))
		return ret, 18, err
	}
	return false, 0, errors.New("unexpected " + DescribeValue(data) + " when expecting bool")
}

func DecodeTime(data []byte) (time.Time, int, error) {
	if len(data) < 6 {
		return time.Time{}, 0, ErrShortInputForTime
	}
	switch data[0] {
	case 0xc7:
		s := int(data[1]) + 3
		if len(data) < s {
			return time.Time{}, 0, ErrShortInput
		}
		ret, err := decodeTime_ext(data[3:s], int8(data[2]))
		return ret, s, err
	case 0xc8:
		s := int(binary.BigEndian.Uint16(data[1:3])) + 4
		if len(data) < s {
			return time.Time{}, 0, ErrShortInput
		}
		ret, err := decodeTime_ext(data[4:s], int8(data[3]))
		return ret, s, err
	case 0xc9:
		s := int(binary.BigEndian.Uint32(data[1:5])) + 6
		if len(data) < s {
			return time.Time{}, 0, ErrShortInput
		}
		ret, err := decodeTime_ext(data[6:s], int8(data[5]))
		return ret, s, err
	case 0xd4:
		ret, err := decodeTime_ext(data[2:3], int8(data[1]))
		return ret, 3, err
	case 0xd5:
		ret, err := decodeTime_ext(data[2:4], int8(data[1]))
		return ret, 4, err
	case 0xd6:
		ret, err := decodeTime_ext(data[2:6], int8(data[1]))
		return ret, 6, err
	case 0xd7:
		if len(data) < 10 {
			return time.Time{}, 0, ErrShortInput
		}
		ret, err := decodeTime_ext(data[2:10], int8(data[1]))
		return ret, 10, err
	case 0xd8:
		if len(data) < 18 {
			return time.Time{}, 0, ErrShortInput
		}
		ret, err := decodeTime_ext(data[2:18], int8(data[1]))
		return ret, 18, err
	}
	return time.Time{}, 0, errors.New("unexpected " + DescribeValue(data) + " when expecting time")
}
