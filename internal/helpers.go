package internal

import (
	"encoding/binary"
	"errors"
	"fmt"
	"time"
)

var (
	ErrShortInput         = errors.New("msgpack data ends unexpectedly")
	ErrShortInputForTime  = errors.New("msgpack data is too short to hold a time")
	ErrShortInputForFloat = errors.New("msgpack data is too short to hold a float")
)

func SkipMultiple(data []byte, offset, num int) (int, error) {
	for num > 0 {
		if len(data) < offset {
			return 0, ErrShortInput
		}
		c, err := ValueLength(data[offset:])
		if err != nil {
			return 0, err
		}
		offset += c
		num--
	}
	return offset, nil
}

func decodeInt_ext(data []byte, extType int8) (int, error) {
	switch extType {
	case 17: // Length-prefixed entry
		ret, _, err := DecodeInt(data)
		return ret, err

	default:
		extType := extType // Only let it escape in this (unlikely) branch.
		return 0, fmt.Errorf("unexpected extension %d while expecting int", extType)
	}
}

func decodeFloat32_ext(data []byte, extType int8) (float32, error) {
	switch extType {
	case 17: // Length-prefixed entry
		ret, _, err := DecodeFloat32(data)
		return ret, err

	default:
		extType := extType // Only let it escape in this (unlikely) branch.
		return 0, fmt.Errorf("unexpected extension %d while expecting float32", extType)
	}
}

func decodeFloat64_ext(data []byte, extType int8) (float64, error) {
	switch extType {
	case 17: // Length-prefixed entry
		ret, _, err := DecodeFloat64(data)
		return ret, err

	default:
		extType := extType // Only let it escape in this (unlikely) branch.
		return 0, fmt.Errorf("unexpected extension %d while expecting float64", extType)
	}
}

func decodeBool_ext(data []byte, extType int8) (bool, error) {
	switch extType {
	case 17: // Length-prefixed entry
		ret, _, err := DecodeBool(data)
		return ret, err

	default:
		extType := extType // Only let it escape in this (unlikely) branch.
		return false, fmt.Errorf("unexpected extension %d while expecting bool", extType)
	}
}

func DecodeTimestamp(data []byte) (time.Time, error) {
	switch len(data) {
	case 4:
		return time.Unix(int64(binary.BigEndian.Uint32(data)), 0), nil
	case 8:
		n := binary.BigEndian.Uint64(data)
		return time.Unix(int64(n&0x00000003ffffffff), int64(n>>34)), nil
	case 12:
		nsec := binary.BigEndian.Uint32(data[:4])
		sec := binary.BigEndian.Uint64(data[4:])
		return time.Unix(int64(sec), int64(nsec)), nil
	}
	return time.Time{}, fmt.Errorf("failed to decode timestamp of %d bytes", len(data))
}

func decodeTime_ext(data []byte, extType int8) (time.Time, error) {
	switch extType {
	case -1: // Timestamp
		return DecodeTimestamp(data)

	case 17: // Length-prefixed entry
		ret, _, err := DecodeTime(data)
		return ret, err

	default:
		extType := extType // Only let it escape in this (unlikely) branch.
		return time.Time{}, fmt.Errorf("unexpected extension %d while expecting time", extType)
	}
}
