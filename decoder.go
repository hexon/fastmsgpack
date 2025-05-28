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
	data        []byte
	opt         internal.DecodeOptions
	nestingInfo []nestingInfo
	offset      int
}

type nestingInfo struct {
	returnTo          []byte
	remainingElements int
	end               int
}

// NewDecoder initializes a new Decoder.
func NewDecoder(data []byte, opts ...DecodeOption) *Decoder {
	d := &Decoder{
		data:        data,
		nestingInfo: make([]nestingInfo, 0, 8),
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

// WithInjection replaces any encountered extension 20 encoding number $field with the given msgpack data.
func WithInjection(field uint, msgpack []byte) DecodeOption {
	return func(opt *internal.DecodeOptions) {
		if opt.Injections == nil {
			opt.Injections = map[uint][]byte{}
		}
		opt.Injections[field] = msgpack
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
	elements, c, end, stepIn, err := internal.DecodeMapLen(d.data[d.offset:], d.opt)
	if err != nil {
		return 0, err
	}
	if end > 0 {
		end += d.offset
	}
	d.offset += c
	d.consumingPush(elements*2, c, end, stepIn)
	return elements, nil
}

func (d *Decoder) DecodeArrayLen() (int, error) {
	elements, c, end, stepIn, err := internal.DecodeArrayLen(d.data[d.offset:], d.opt)
	if err != nil {
		return 0, err
	}
	if end > 0 {
		end += d.offset
	}
	d.offset += c
	d.consumingPush(elements, c, end, stepIn)
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

// DecodeLazy returns a new decoder for the next value in the stream, and progresses the current decoder over that value.
// It is equivalent to `NewDecoder(d.DecodeRaw(), sameOptions...)`.
func (d *Decoder) DecodeLazy() (*Decoder, error) {
	b, err := d.DecodeRaw()
	if err != nil {
		return nil, err
	}
	return &Decoder{
		data: b,
		opt:  d.opt,
	}, nil
}

// Break out of the map or array we're currently in.
// This can only be called before the last element of the array/map is read, because otherwise you'd break out one level higher.
func (d *Decoder) Break() error {
	l := len(d.nestingInfo) - 1
	if l < 0 {
		return errors.New("fastmsgpack.Decoder.Break: can't Break at the top level")
	}
	ni := d.nestingInfo[l]
	d.nestingInfo[l].returnTo = nil // don't retain the pointer
	d.nestingInfo = d.nestingInfo[:l]
	switch {
	case ni.returnTo != nil:
		d.data = ni.returnTo
		fallthrough
	case ni.end > 0:
		d.offset = ni.end
		return nil
	}
	c, err := internal.SkipMultiple(d.data, d.offset, ni.remainingElements)
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
	l := len(d.nestingInfo) - 1
	if l < 0 {
		return
	}
	if d.nestingInfo[l].remainingElements > 1 {
		d.nestingInfo[l].remainingElements--
	} else {
		if d.nestingInfo[l].returnTo != nil {
			d.data = d.nestingInfo[l].returnTo
			d.offset = d.nestingInfo[l].end
			d.nestingInfo[l].returnTo = nil // don't retain the pointer
		}
		d.nestingInfo = d.nestingInfo[:l]
	}
}

func (d *Decoder) consumingPush(elements, consume, end int, stepIn []byte) {
	// invariant: If stepIn != nil, end is known (and not 0)
	if elements == 0 {
		d.consumedOne()
		// We either have an extension-wrapped value and $end is known; or we have a simple 0x80 empty map, in which case d.offset is already adjusted.
		if end > 0 {
			d.offset = end
		}
		return
	}
	add := nestingInfo{
		remainingElements: elements,
		end:               end,
	}
	if stepIn != nil {
		add.returnTo = d.data
		add.end = end
		d.data = stepIn
		d.offset = consume
	}
	l := len(d.nestingInfo) - 1
	if l < 0 {
		d.nestingInfo = append(d.nestingInfo, add)
		return
	}
	if d.nestingInfo[l].remainingElements > 1 {
		d.nestingInfo[l].remainingElements--
		d.nestingInfo = append(d.nestingInfo, add)
	} else if d.nestingInfo[l].returnTo != nil {
		// Retain our parent's returnTo and end values, because we are the last child of our parent our end is their end.
		d.nestingInfo[l].remainingElements = add.remainingElements
	} else {
		d.nestingInfo[l] = add
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

	case 20: // Injection
		b, err := internal.DecodeInjectionExtension(data, opt)
		if err != nil {
			return nil, err
		}
		ret, _, err := decodeValue(b, opt)
		return ret, err

	default:
		return Extension{Type: extType, Data: data}, nil
	}
}
