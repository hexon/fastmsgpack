package internal

import (
	"encoding/binary"
	"errors"
	"fmt"
	"maps"
	"time"
	"unsafe"
)

var (
	ErrVoid              = errors.New("tried to decode a void value")
	ErrShortInput        = errors.New("msgpack data ends unexpectedly")
	ErrShortInputForTime = errors.New("msgpack data is too short to hold a time")
)

type DecodeOptions struct {
	Dict            *Dict
	FlavorSelectors map[uint]uint
	Injections      map[uint][]byte
}

func (d DecodeOptions) Clone() DecodeOptions {
	return DecodeOptions{
		Dict:            d.Dict,
		FlavorSelectors: maps.Clone(d.FlavorSelectors),
		Injections:      maps.Clone(d.Injections),
	}
}

func UnsafeStringCast(data []byte) string {
	return unsafe.String(unsafe.SliceData(data), len(data))
}

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

func decodeString_ext(data []byte, extType int8, opt DecodeOptions) (string, error) {
	switch extType {
	case -128: // Interned string
		n, ok := DecodeBytesToUint(data)
		if !ok {
			return "", errors.New("failed to decode index number of interned string")
		}
		return opt.Dict.LookupString(n)

	case 17: // Length-prefixed entry
		ret, _, err := DecodeString(data, opt)
		return ret, err

	case 18: // Flavor pick
		j, err := DecodeFlavorPick(data, opt)
		if err != nil {
			return "", err
		}
		ret, _, err := DecodeString(data[j:], opt)
		return ret, err

	case 19: // Void
		return "", ErrVoid

	case 20: // Injection
		b, err := DecodeInjectionExtension(data, opt)
		if err != nil {
			return "", err
		}
		ret, _, err := DecodeString(b, opt)
		return ret, err

	default:
		extType := extType // Only let it escape in this (unlikely) branch.
		return "", fmt.Errorf("unexpected extension %d while expecting string", extType)
	}
}

func decodeInt_ext(data []byte, extType int8, opt DecodeOptions) (int, error) {
	switch extType {
	case 17: // Length-prefixed entry
		ret, _, err := DecodeInt(data, opt)
		return ret, err

	case 18: // Flavor pick
		j, err := DecodeFlavorPick(data, opt)
		if err != nil {
			return 0, err
		}
		ret, _, err := DecodeInt(data[j:], opt)
		return ret, err

	default:
		extType := extType // Only let it escape in this (unlikely) branch.
		return 0, fmt.Errorf("unexpected extension %d while expecting int", extType)
	}
}

func decodeFloat32_ext(data []byte, extType int8, opt DecodeOptions) (float32, error) {
	switch extType {
	case 17: // Length-prefixed entry
		ret, _, err := DecodeFloat32(data, opt)
		return ret, err

	case 18: // Flavor pick
		j, err := DecodeFlavorPick(data, opt)
		if err != nil {
			return 0, err
		}
		ret, _, err := DecodeFloat32(data[j:], opt)
		return ret, err

	case 19: // Void
		return 0, ErrVoid

	case 20: // Injection
		b, err := DecodeInjectionExtension(data, opt)
		if err != nil {
			return 0, err
		}
		ret, _, err := DecodeFloat32(b, opt)
		return ret, err

	default:
		extType := extType // Only let it escape in this (unlikely) branch.
		return 0, fmt.Errorf("unexpected extension %d while expecting float32", extType)
	}
}

func decodeFloat64_ext(data []byte, extType int8, opt DecodeOptions) (float64, error) {
	switch extType {
	case 17: // Length-prefixed entry
		ret, _, err := DecodeFloat64(data, opt)
		return ret, err

	case 18: // Flavor pick
		j, err := DecodeFlavorPick(data, opt)
		if err != nil {
			return 0, err
		}
		ret, _, err := DecodeFloat64(data[j:], opt)
		return ret, err

	case 19: // Void
		return 0, ErrVoid

	case 20: // Injection
		b, err := DecodeInjectionExtension(data, opt)
		if err != nil {
			return 0, err
		}
		ret, _, err := DecodeFloat64(b, opt)
		return ret, err

	default:
		extType := extType // Only let it escape in this (unlikely) branch.
		return 0, fmt.Errorf("unexpected extension %d while expecting float64", extType)
	}
}

func decodeBool_ext(data []byte, extType int8, opt DecodeOptions) (bool, error) {
	switch extType {
	case 17: // Length-prefixed entry
		ret, _, err := DecodeBool(data, opt)
		return ret, err

	case 18: // Flavor pick
		j, err := DecodeFlavorPick(data, opt)
		if err != nil {
			return false, err
		}
		ret, _, err := DecodeBool(data[j:], opt)
		return ret, err

	case 19: // Void
		return false, ErrVoid

	case 20: // Injection
		b, err := DecodeInjectionExtension(data, opt)
		if err != nil {
			return false, err
		}
		ret, _, err := DecodeBool(b, opt)
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

func decodeTime_ext(data []byte, extType int8, opt DecodeOptions) (time.Time, error) {
	switch extType {
	case -1: // Timestamp
		return DecodeTimestamp(data)

	case 17: // Length-prefixed entry
		ret, _, err := DecodeTime(data, opt)
		return ret, err

	case 18: // Flavor pick
		j, err := DecodeFlavorPick(data, opt)
		if err != nil {
			return time.Time{}, err
		}
		ret, _, err := DecodeTime(data[j:], opt)
		return ret, err

	case 19: // Void
		return time.Time{}, ErrVoid

	case 20: // Injection
		b, err := DecodeInjectionExtension(data, opt)
		if err != nil {
			return time.Time{}, err
		}
		ret, _, err := DecodeTime(b, opt)
		return ret, err

	default:
		extType := extType // Only let it escape in this (unlikely) branch.
		return time.Time{}, fmt.Errorf("unexpected extension %d while expecting time", extType)
	}
}

func decodeMapLen_ext(data []byte, extType int8, opt DecodeOptions) (elements, consume int, stepIn []byte, err error) {
	switch extType {
	case 17: // Length-prefixed entry
		elements, consume, _, stepIn, err = DecodeMapLen(data, opt)
		return elements, consume, stepIn, err

	case 18: // Flavor pick
		j, err := DecodeFlavorPick(data, opt)
		if err != nil {
			return 0, 0, nil, err
		}
		elements, consume, _, stepIn, err = DecodeMapLen(data[j:], opt)
		if err != nil {
			return 0, 0, nil, err
		}
		if stepIn == nil {
			return elements, consume + j, data, nil
		}
		return elements, consume, stepIn, nil

	case 19: // Void
		return 0, 0, nil, ErrVoid

	case 20: // Injection
		b, err := DecodeInjectionExtension(data, opt)
		if err != nil {
			return 0, 0, nil, err
		}
		elements, consume, _, stepIn, err = DecodeMapLen(b, opt)
		if err != nil {
			return 0, 0, nil, err
		}
		if stepIn == nil {
			stepIn = b
		}
		return elements, consume, stepIn, nil

	default:
		extType := extType // Only let it escape in this (unlikely) branch.
		return 0, 0, nil, fmt.Errorf("unexpected extension %d while expecting map", extType)
	}
}

func decodeArrayLen_ext(data []byte, extType int8, opt DecodeOptions) (elements, consume int, stepIn []byte, err error) {
	switch extType {
	case 17: // Length-prefixed entry
		elements, consume, _, stepIn, err = DecodeArrayLen(data, opt)
		return elements, consume, stepIn, err

	case 18: // Flavor pick
		j, err := DecodeFlavorPick(data, opt)
		if err != nil {
			return 0, 0, nil, err
		}
		elements, consume, _, stepIn, err = DecodeArrayLen(data[j:], opt)
		if err != nil {
			return 0, 0, nil, err
		}
		if stepIn == nil {
			return elements, consume + j, data, nil
		}
		return elements, consume, stepIn, nil

	case 19: // Void
		return 0, 0, nil, ErrVoid

	case 20: // Injection
		b, err := DecodeInjectionExtension(data, opt)
		if err != nil {
			return 0, 0, nil, err
		}
		elements, consume, _, stepIn, err = DecodeArrayLen(b, opt)
		if err != nil {
			return 0, 0, nil, err
		}
		if stepIn == nil {
			stepIn = b
		}
		return elements, consume, stepIn, nil

	default:
		extType := extType // Only let it escape in this (unlikely) branch.
		return 0, 0, nil, fmt.Errorf("unexpected extension %d while expecting array", extType)
	}
}

func DecodeInjectionExtension(data []byte, opt DecodeOptions) ([]byte, error) {
	n, ok := DecodeBytesToUint(data)
	if !ok {
		return nil, errors.New("failed to decode index number of inject extension")
	}
	b, ok := opt.Injections[n]
	if !ok {
		return nil, fmt.Errorf("data tried to look at injection %d", n)
	}
	return b, nil
}
