package fastmsgpack

import (
	"encoding/binary"
	"errors"
	"time"

	"github.com/hexon/fastmsgpack/internal"
)

// Decoder gives a low-level api for stepping through msgpack data.
// Any []byte and string in return values might point into memory from the given data. Don't modify the input data until you're done with the return value.
type Decoder struct {
	data     []byte
	opt      internal.DecodeOptions
	skipInfo []skipInfo
	offset   int
}

type skipInfo struct {
	remainingElements int
	fastSkip          int
	forceJump         bool
}

// NewDecoder initializes a new Decoder.
func NewDecoder(data []byte, opts ...DecodeOption) *Decoder {
	d := &Decoder{
		data:     data,
		skipInfo: make([]skipInfo, 0, 8),
	}
	for _, o := range opts {
		o(&d.opt)
	}
	return d
}

type DecodeOption func(*internal.DecodeOptions)

func WithDict(dict *Dict) DecodeOption {
	return func(opt *internal.DecodeOptions) {
		opt.Dict = &internal.Dict{
			Strings:     dict.Strings,
			Interfaces:  dict.interfaces,
			JSONEncoded: &dict.jsonEncoded,
		}
	}
}

func WithFlavorSelector(field, value uint) DecodeOption {
	return func(opt *internal.DecodeOptions) {
		if opt.FlavorSelectors == nil {
			opt.FlavorSelectors = map[uint]uint{}
		}
		opt.FlavorSelectors[field] = value
	}
}

// DecodeValue decodes the next value in the msgpack data. Return types are: nil, bool, int, float32, float64, string, []byte, time.Time, []any, map[string]any or Extension.
func (d *Decoder) DecodeValue() (any, error) {
	v, c, err := decodeValue(d.data[d.offset:], d.opt)
	if err != nil {
		return nil, err
	}
	d.offset += c
	d.consumedOne()
	return v, nil
}

func (d *Decoder) DecodeString() (string, error) {
	v, c, err := internal.DecodeString(d.data[d.offset:], d.opt)
	if err != nil {
		return "", err
	}
	d.offset += c
	d.consumedOne()
	return v, nil
}

func (d *Decoder) DecodeInt() (int, error) {
	v, c, err := internal.DecodeInt(d.data[d.offset:], d.opt)
	if err != nil {
		return 0, err
	}
	d.offset += c
	d.consumedOne()
	return v, nil
}

func (d *Decoder) DecodeFloat32() (float32, error) {
	v, c, err := internal.DecodeFloat32(d.data[d.offset:], d.opt)
	if err != nil {
		return 0, err
	}
	d.offset += c
	d.consumedOne()
	return v, nil
}

func (d *Decoder) DecodeFloat64() (float64, error) {
	v, c, err := internal.DecodeFloat64(d.data[d.offset:], d.opt)
	if err != nil {
		return 0, err
	}
	d.offset += c
	d.consumedOne()
	return v, nil
}

func (d *Decoder) DecodeBool() (bool, error) {
	v, c, err := internal.DecodeBool(d.data[d.offset:], d.opt)
	if err != nil {
		return false, err
	}
	d.offset += c
	d.consumedOne()
	return v, nil
}

func (d *Decoder) DecodeTime() (time.Time, error) {
	v, c, err := internal.DecodeTime(d.data[d.offset:], d.opt)
	if err != nil {
		return time.Time{}, err
	}
	d.offset += c
	d.consumedOne()
	return v, nil
}

func (d *Decoder) DecodeMapLen() (int, error) {
	si, err := d.decodeMapLen()
	if err != nil {
		return 0, err
	}
	ret := si.remainingElements
	si.remainingElements *= 2
	d.consumingPush(si)
	return ret, nil
}

func (d *Decoder) decodeMapLen() (skipInfo, error) {
	data, end, forced, err := d.consumeWrappingExtensions()
	if err != nil {
		return skipInfo{}, err
	}
	if len(data) < 1 {
		return skipInfo{}, internal.ErrShortInput
	}
	ret := skipInfo{
		fastSkip:  end,
		forceJump: forced,
	}
	if data[0] >= 0x80 && data[0] <= 0x8f {
		d.offset++
		ret.remainingElements = int(data[0] - 0x80)
		return ret, nil
	}
	switch data[0] {
	case 0xde:
		if len(data) < 3 {
			return skipInfo{}, internal.ErrShortInput
		}
		d.offset += 3
		ret.remainingElements = int(binary.BigEndian.Uint16(data[1:3]))
		return ret, nil
	case 0xdf:
		if len(data) < 5 {
			return skipInfo{}, internal.ErrShortInput
		}
		d.offset += 5
		ret.remainingElements = int(binary.BigEndian.Uint32(data[1:5]))
		return ret, nil
	}
	return skipInfo{}, errors.New("unexpected " + internal.DescribeValue(data) + " when expecting map")
}

func (d *Decoder) DecodeArrayLen() (int, error) {
	si, err := d.decodeArrayLen()
	if err != nil {
		return 0, err
	}
	d.consumingPush(si)
	return si.remainingElements, nil
}

func (d *Decoder) decodeArrayLen() (skipInfo, error) {
	data, end, forced, err := d.consumeWrappingExtensions()
	if err != nil {
		return skipInfo{}, err
	}
	if len(data) < 1 {
		return skipInfo{}, internal.ErrShortInput
	}
	ret := skipInfo{
		fastSkip:  end,
		forceJump: forced,
	}
	if data[0] >= 0x90 && data[0] <= 0x9f {
		d.offset++
		ret.remainingElements = int(data[0] - 0x90)
		return ret, nil
	}
	switch data[0] {
	case 0xdc:
		if len(data) < 3 {
			return skipInfo{}, internal.ErrShortInput
		}
		d.offset += 3
		ret.remainingElements = int(binary.BigEndian.Uint16(data[1:3]))
		return ret, nil
	case 0xdd:
		if len(data) < 5 {
			return skipInfo{}, internal.ErrShortInput
		}
		d.offset += 5
		ret.remainingElements = int(binary.BigEndian.Uint32(data[1:5]))
		return ret, nil
	}
	return skipInfo{}, errors.New("unexpected " + internal.DescribeValue(data) + " when expecting array")
}

func (d *Decoder) consumeWrappingExtensions() ([]byte, int, bool, error) {
	data := d.data[d.offset:]
	if len(data) < 3 {
		return data, 0, false, nil
	}
	switch data[1] {
	case 17:
		switch data[0] {
		case 0xd4, 0xd5, 0xd6, 0xd7, 0xd8:
			d.offset += 2
			return data[2:], d.offset + (1 << (data[0] - 0xd4)), false, nil
		}
	case 18:
		switch data[0] {
		case 0xd4, 0xd5, 0xd6, 0xd7, 0xd8:
			d.offset += 2
			return d.consumeFlavorSelector(data[2:][:1<<(data[0]-0xd4)])
		}
	}
	switch data[0] {
	case 0xc7:
		if len(data) < 3 {
			break
		}
		switch data[2] {
		case 17:
			d.offset += 3
			return data[3:], d.offset + int(data[1]), false, nil
		case 18:
			d.offset += 3
			return d.consumeFlavorSelector(data[3:][:data[1]])
		}
	case 0xc8:
		if len(data) < 4 {
			break
		}
		switch data[3] {
		case 17:
			d.offset += 4
			return data[4:], d.offset + int(binary.BigEndian.Uint16(data[1:3])), false, nil
		case 18:
			d.offset += 4
			return d.consumeFlavorSelector(data[4:][:binary.BigEndian.Uint16(data[1:3])])
		}
	case 0xc9:
		if len(data) < 6 {
			break
		}
		switch data[5] {
		case 17:
			d.offset += 6
			return data[6:], d.offset + int(binary.BigEndian.Uint32(data[1:5])), false, nil
		case 18:
			d.offset += 6
			return d.consumeFlavorSelector(data[6:][:binary.BigEndian.Uint32(data[1:5])])
		}
	}
	return data, 0, false, nil
}

func (d *Decoder) consumeFlavorSelector(data []byte) ([]byte, int, bool, error) {
	j, err := internal.DecodeFlavorPick(data, d.opt)
	if err != nil {
		return nil, 0, false, err
	}
	end := d.offset + len(data)
	d.offset += j
	return data[j:], end, true, nil
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

// DecodeRaw decodes the next value enough to know its length and returns the msgpack data for it while skipping over it.
func (d *Decoder) DecodeRaw() ([]byte, error) {
	b := d.data[d.offset:]
	c, err := internal.ValueLength(b)
	if err != nil {
		return nil, err
	}
	b = b[:c]
	d.offset += c
	d.consumedOne()
	return b, nil
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
		if d.skipInfo[l].forceJump {
			d.offset = d.skipInfo[l].fastSkip
		}
		d.skipInfo = d.skipInfo[:l]
	}
}

func (d *Decoder) consumingPush(add skipInfo) {
	if add.remainingElements == 0 {
		d.consumedOne()
		if add.forceJump {
			d.offset = add.fastSkip
		}
		return
	}
	l := len(d.skipInfo) - 1
	if l < 0 {
		d.skipInfo = append(d.skipInfo, add)
		return
	}
	if d.skipInfo[l].remainingElements > 1 {
		d.skipInfo[l].remainingElements--
		d.skipInfo = append(d.skipInfo, add)
	} else if d.skipInfo[l].forceJump {
		// Retain our parent's fastSkip and forceJump values.
		d.skipInfo[l].remainingElements = add.remainingElements
	} else {
		d.skipInfo[l] = add
	}
}

func decodeValue_array(data []byte, offset, num int, opt internal.DecodeOptions) ([]any, int, error) {
	ret := make([]any, num)
	for i := range ret {
		v, c, err := decodeValue(data[offset:], opt)
		if err != nil {
			return nil, 0, err
		}
		ret[i] = v
		offset += c
	}
	return ret, offset, nil
}

func decodeValue_map(data []byte, offset, num int, opt internal.DecodeOptions) (map[string]any, int, error) {
	ret := make(map[string]any, num)
	for num > 0 {
		k, c, err := internal.DecodeString(data[offset:], opt)
		if err != nil {
			return nil, 0, err
		}
		offset += c
		v, c, err := decodeValue(data[offset:], opt)
		if err != nil {
			return nil, 0, err
		}
		ret[k] = v
		offset += c
		num--
	}
	return ret, offset, nil
}

func decodeValue_ext(data []byte, extType int8, opt internal.DecodeOptions) (any, error) {
	switch extType {
	case -1: // Timestamp
		return internal.DecodeTimestamp(data)

	case -128: // Interned string
		n, ok := internal.DecodeBytesToUint(data)
		if !ok {
			return nil, errors.New("failed to decode index number of interned string")
		}
		return opt.Dict.LookupAny(n)

	case 17: // Length-prefixed entry
		ret, _, err := decodeValue(data, opt)
		return ret, err

	case 18: // Flavor pick
		j, err := internal.DecodeFlavorPick(data, opt)
		if err != nil {
			return nil, err
		}
		ret, _, err := decodeValue(data[j:], opt)
		return ret, err

	default:
		return Extension{Type: extType, Data: data}, nil
	}
}
