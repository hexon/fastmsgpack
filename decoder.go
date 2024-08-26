package fastmsgpack

import (
	"errors"
	"time"

	"github.com/hexon/fastmsgpack/internal"
)

var ErrVoid = internal.ErrVoid

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
	elements, c, end, forceJump, err := internal.DecodeMapLen(d.data[d.offset:], d.opt)
	if err != nil {
		return 0, err
	}
	if end > 0 {
		end += d.offset
	}
	d.offset += c
	d.consumingPush(skipInfo{
		remainingElements: elements * 2,
		fastSkip:          end,
		forceJump:         forceJump,
	})
	return elements, nil
}

func (d *Decoder) DecodeArrayLen() (int, error) {
	elements, c, end, forceJump, err := internal.DecodeArrayLen(d.data[d.offset:], d.opt)
	if err != nil {
		return 0, err
	}
	if end > 0 {
		end += d.offset
	}
	d.offset += c
	d.consumingPush(skipInfo{
		remainingElements: elements,
		fastSkip:          end,
		forceJump:         forceJump,
	})
	return elements, nil
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
	var voided int
	for i := range ret {
		v, c, err := decodeValue(data[offset:], opt)
		offset += c
		if err != nil {
			if err == ErrVoid {
				voided++
				continue
			}
			return nil, 0, err
		}
		ret[i-voided] = v
	}
	ret = ret[:len(ret)-voided]
	return ret, offset, nil
}

func decodeValue_map(data []byte, offset, num int, opt internal.DecodeOptions) (map[string]any, int, error) {
	ret := make(map[string]any, num)
	for num > 0 {
		num--
		k, c, err := internal.DecodeString(data[offset:], opt)
		if err != nil {
			if err == ErrVoid {
				offset, err = internal.SkipMultiple(data, offset, 2)
				if err == nil {
					continue
				}
			}
			return nil, 0, err
		}
		offset += c
		v, c, err := decodeValue(data[offset:], opt)
		if err != nil {
			if err == ErrVoid {
				c, err = internal.ValueLength(data[offset:])
				if err == nil {
					offset += c
					continue
				}
			}
			return nil, 0, err
		}
		ret[k] = v
		offset += c
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

	case 19:
		return nil, ErrVoid

	default:
		return Extension{Type: extType, Data: data}, nil
	}
}
