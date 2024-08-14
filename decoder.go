package fastmsgpack

import (
	"encoding/binary"
	"errors"
	"fmt"
	"time"

	"github.com/hexon/fastmsgpack/internal"
)

// Decoder gives a low-level api for stepping through msgpack data.
// Any []byte and string in return values might point into memory from the given data. Don't modify the input data until you're done with the return value.
type Decoder struct {
	data     []byte
	dict     *Dict
	skipInfo []skipInfo
	offset   int
}

type skipInfo struct {
	remainingElements int
	fastSkip          int
}

// NewDecoder initializes a new Decoder.
// The dictionary is optional and can be nil.
func NewDecoder(data []byte, dict *Dict) *Decoder {
	return &Decoder{
		data:     data,
		dict:     dict,
		skipInfo: make([]skipInfo, 0, 8),
	}
}

// DecodeValue decodes the next value in the msgpack data. Return types are: nil, bool, int, float32, float64, string, []byte, time.Time, []any, map[string]any or Extension.
func (d *Decoder) DecodeValue() (any, error) {
	v, c, err := decodeValue(d.data[d.offset:], d.dict)
	if err != nil {
		return nil, err
	}
	d.offset += c
	d.consumedOne()
	return v, nil
}

func (d *Decoder) DecodeString() (string, error) {
	v, c, err := decodeString(d.data[d.offset:], d.dict)
	if err != nil {
		return "", err
	}
	d.offset += c
	d.consumedOne()
	return v, nil
}

func (d *Decoder) DecodeInt() (int, error) {
	v, c, err := internal.DecodeInt(d.data[d.offset:])
	if err != nil {
		return 0, err
	}
	d.offset += c
	d.consumedOne()
	return v, nil
}

func (d *Decoder) DecodeFloat32() (float32, error) {
	v, c, err := internal.DecodeFloat32(d.data[d.offset:])
	if err != nil {
		return 0, err
	}
	d.offset += c
	d.consumedOne()
	return v, nil
}

func (d *Decoder) DecodeFloat64() (float64, error) {
	v, c, err := internal.DecodeFloat64(d.data[d.offset:])
	if err != nil {
		return 0, err
	}
	d.offset += c
	d.consumedOne()
	return v, nil
}

func (d *Decoder) DecodeBool() (bool, error) {
	v, c, err := internal.DecodeBool(d.data[d.offset:])
	if err != nil {
		return false, err
	}
	d.offset += c
	d.consumedOne()
	return v, nil
}

func (d *Decoder) DecodeTime() (time.Time, error) {
	v, c, err := internal.DecodeTime(d.data[d.offset:])
	if err != nil {
		return time.Time{}, err
	}
	d.offset += c
	d.consumedOne()
	return v, nil
}

func (d *Decoder) DecodeMapLen() (int, error) {
	n, fastSkip, err := d.decodeMapLen()
	if err != nil {
		return 0, err
	}
	d.consumedOne()
	if n > 0 {
		d.skipInfo = append(d.skipInfo, skipInfo{n * 2, fastSkip})
	}
	return n, nil
}

func (d *Decoder) decodeMapLen() (int, int, error) {
	data, fastSkip := d.consumeLengthPrefixEntry()
	if len(data) < 1 {
		return 0, 0, internal.ErrShortInput
	}
	if data[0] >= 0x80 && data[0] <= 0x8f {
		d.offset++
		return int(data[0] - 0x80), fastSkip, nil
	}
	switch data[0] {
	case 0xde:
		if len(data) < 3 {
			return 0, 0, internal.ErrShortInput
		}
		d.offset += 3
		return int(binary.BigEndian.Uint16(data[1:3])), fastSkip, nil
	case 0xdf:
		if len(data) < 5 {
			return 0, 0, internal.ErrShortInput
		}
		d.offset += 5
		return int(binary.BigEndian.Uint32(data[1:5])), fastSkip, nil
	}
	return 0, 0, errors.New("unexpected " + internal.DescribeValue(data) + " when expecting map")
}

func (d *Decoder) DecodeArrayLen() (int, error) {
	n, fastSkip, err := d.decodeArrayLen()
	if err != nil {
		return 0, err
	}
	d.consumedOne()
	if n > 0 {
		d.skipInfo = append(d.skipInfo, skipInfo{n, fastSkip})
	}
	return n, nil
}

func (d *Decoder) decodeArrayLen() (int, int, error) {
	data, fastSkip := d.consumeLengthPrefixEntry()
	if len(data) < 1 {
		return 0, 0, internal.ErrShortInput
	}
	if data[0] >= 0x90 && data[0] <= 0x9f {
		d.offset++
		return int(data[0] - 0x90), fastSkip, nil
	}
	switch data[0] {
	case 0xdc:
		if len(data) < 3 {
			return 0, 0, internal.ErrShortInput
		}
		d.offset += 3
		return int(binary.BigEndian.Uint16(data[1:3])), fastSkip, nil
	case 0xdd:
		if len(data) < 5 {
			return 0, 0, internal.ErrShortInput
		}
		d.offset += 5
		return int(binary.BigEndian.Uint32(data[1:5])), fastSkip, nil
	}
	return 0, 0, errors.New("unexpected " + internal.DescribeValue(data) + " when expecting array")
}

func (d *Decoder) consumeLengthPrefixEntry() ([]byte, int) {
	data := d.data[d.offset:]
	if len(data) < 3 {
		return data, 0
	}
	if data[1] == 17 {
		switch data[0] {
		case 0xd4, 0xd5, 0xd6, 0xd7, 0xd8:
			d.offset += 2
			return data[2:], d.offset + (1 << (data[0] - 0xd4))
		}
	}
	switch data[0] {
	case 0xc7:
		if len(data) >= 3 && data[2] == 17 {
			d.offset += 3
			return data[3:], d.offset + int(data[1])
		}
	case 0xc8:
		if len(data) >= 4 && data[3] == 17 {
			d.offset += 4
			return data[4:], d.offset + int(binary.BigEndian.Uint16(data[1:3]))
		}
	case 0xc9:
		if len(data) >= 6 && data[5] == 17 {
			d.offset += 6
			return data[6:], d.offset + int(binary.BigEndian.Uint32(data[1:5]))
		}
	}
	return data, 0
}

func (d *Decoder) Skip() error {
	c, err := internal.ValueLength(d.data[d.offset:])
	if err != nil {
		return err
	}
	d.offset += c
	d.consumedOne()
	return nil
}

// Break out of the map or array we're currently in.
// This can only be called before the last element of the array/map is read, because otherwise you'd break out one level higher.
func (d *Decoder) Break() error {
	l := len(d.skipInfo) - 1
	if l < 0 {
		return errors.New("fastmsgpack.Decoder.Break: can't Break at the top level")
	}
	si := d.skipInfo[l]
	d.skipInfo = d.skipInfo[:l]
	if si.fastSkip > 0 {
		d.offset = si.fastSkip
		return nil
	}
	c, err := internal.SkipMultiple(d.data, d.offset, si.remainingElements)
	if err != nil {
		return err
	}
	d.offset = c
	return nil
}

// PeekType returns the type of next entry without changing the state of the Decoder.
// PeekType returning another value than TypeInvalid does not guarantee decoding it will succeed.
func (d *Decoder) PeekType() ValueType {
	return DecodeType(d.data[d.offset:])
}

func (d *Decoder) consumedOne() {
	l := len(d.skipInfo) - 1
	if l < 0 {
		return
	}
	if d.skipInfo[l].remainingElements > 1 {
		d.skipInfo[l].remainingElements--
	} else {
		d.skipInfo = d.skipInfo[:l]
	}
}

func decodeValue_array(data []byte, offset, num int, dict *Dict) ([]any, int, error) {
	ret := make([]any, num)
	for i := range ret {
		v, c, err := decodeValue(data[offset:], dict)
		if err != nil {
			return nil, 0, err
		}
		ret[i] = v
		offset += c
	}
	return ret, offset, nil
}

func decodeValue_map(data []byte, offset, num int, dict *Dict) (map[string]any, int, error) {
	ret := make(map[string]any, num)
	for num > 0 {
		k, c, err := decodeString(data[offset:], dict)
		if err != nil {
			return nil, 0, err
		}
		offset += c
		v, c, err := decodeValue(data[offset:], dict)
		if err != nil {
			return nil, 0, err
		}
		ret[k] = v
		offset += c
		num--
	}
	return ret, offset, nil
}

func decodeValue_ext(data []byte, extType int8, dict *Dict) (any, error) {
	switch extType {
	case -1: // Timestamp
		return internal.DecodeTimestamp(data)

	case -128: // Interned string
		n, ok := internal.DecodeBytesToUint(data)
		if !ok {
			return nil, errors.New("failed to decode index number of interned string")
		}
		return dict.lookupAny(n)

	case 17: // Length-prefixed entry
		ret, _, err := decodeValue(data, dict)
		return ret, err

	default:
		return Extension{Type: extType, Data: data}, nil
	}
}

func decodeString_ext(data []byte, extType int8, dict *Dict) (string, error) {
	switch extType {
	case -128: // Interned string
		n, ok := internal.DecodeBytesToUint(data)
		if !ok {
			return "", errors.New("failed to decode index number of interned string")
		}
		return dict.lookupString(n)

	case 17: // Length-prefixed entry
		ret, _, err := decodeString(data, dict)
		return ret, err

	default:
		extType := extType // Only let it escape in this (unlikely) branch.
		return "", fmt.Errorf("unexpected extension %d while expecting string", extType)
	}
}
