package fastmsgpack

import (
	"encoding/binary"
	"fmt"
	"math"
	"reflect"
	"time"

	"github.com/hexon/fastmsgpack/internal"
)

type EncodeOptions struct {
	CompactInts bool
	Dict        map[string]int
}

// Encode calls EncodeOptions.Decode with the default options.
func Encode(dst []byte, v any) ([]byte, error) {
	return EncodeOptions{}.Encode(dst, v)
}

// Encode appends the msgpack representation to dst and returns the result.
func (o EncodeOptions) Encode(dst []byte, v any) ([]byte, error) {
	switch v := v.(type) {
	case nil:
		return append(dst, 0xc0), nil
	case bool:
		if v {
			return append(dst, 0xc3), nil
		}
		return append(dst, 0xc2), nil

	case []byte:
		if len(v) <= math.MaxUint8 {
			dst = append(dst, 0xc4, uint8(len(v)))
		} else if len(v) <= math.MaxUint16 {
			dst = append(dst, 0xc5, byte(len(v)>>8), byte(len(v)))
		} else if len(v) <= math.MaxUint32 {
			dst = append(dst, 0xc6, byte(len(v)>>24), byte(len(v)>>16), byte(len(v)>>8), byte(len(v)))
		} else {
			return nil, fmt.Errorf("fastmsgpack.Encode: byte slice too long to encode (len %d)", len(v))
		}
		return append(dst, v...), nil
	case string:
		if idx, ok := o.Dict[v]; ok {
			if idx <= math.MaxUint8 {
				return append(dst, 0xd4, 128, byte(idx)), nil
			} else if idx <= math.MaxUint16 {
				return append(dst, 0xd5, 128, byte(idx>>8), byte(idx)), nil
			} else if idx <= math.MaxUint32 {
				return append(dst, 0xd6, 128, byte(idx>>24), byte(idx>>16), byte(idx>>8), byte(idx)), nil
			} else if idx <= math.MaxInt64 {
				return append(dst, 0xd7, 128, byte(idx>>56), byte(idx>>48), byte(idx>>40), byte(idx>>32), byte(idx>>24), byte(idx>>16), byte(idx>>8), byte(idx)), nil
			}
		}
		if len(v) < 32 {
			dst = append(dst, 0xa0|byte(len(v)))
		} else if len(v) <= math.MaxUint8 {
			dst = append(dst, 0xd9, uint8(len(v)))
		} else if len(v) <= math.MaxUint16 {
			dst = append(dst, 0xda, byte(len(v)>>8), byte(len(v)))
		} else if len(v) <= math.MaxUint32 {
			dst = append(dst, 0xdb, byte(len(v)>>24), byte(len(v)>>16), byte(len(v)>>8), byte(len(v)))
		} else {
			return nil, fmt.Errorf("fastmsgpack.Encode: string too long to encode (len %d)", len(v))
		}
		return append(dst, v...), nil

	case float32:
		var buf [5]byte
		buf[0] = 0xca
		binary.BigEndian.PutUint32(buf[1:], math.Float32bits(v))
		return append(dst, buf[:]...), nil
	case float64:
		var buf [9]byte
		buf[0] = 0xcb
		binary.BigEndian.PutUint64(buf[1:], math.Float64bits(v))
		return append(dst, buf[:]...), nil
	case int:
		return appendCompactInt(dst, v), nil
	case uint:
		return appendCompactUint(dst, v), nil
	case int64:
		if o.CompactInts {
			return appendCompactInt(dst, int(v)), nil
		}
		uv := uint64(v)
		return append(dst, 0xd3, byte(uv>>56), byte(uv>>48), byte(uv>>40), byte(uv>>32), byte(uv>>24), byte(uv>>16), byte(uv>>8), byte(uv)), nil
	case uint64:
		if o.CompactInts {
			return appendCompactUint(dst, uint(v)), nil
		}
		return append(dst, 0xcf, byte(v>>56), byte(v>>48), byte(v>>40), byte(v>>32), byte(v>>24), byte(v>>16), byte(v>>8), byte(v)), nil
	case int32:
		if o.CompactInts {
			return appendCompactInt(dst, int(v)), nil
		}
		uv := uint32(v)
		return append(dst, 0xd2, byte(uv>>24), byte(uv>>16), byte(uv>>8), byte(uv)), nil
	case uint32:
		if o.CompactInts {
			return appendCompactUint(dst, uint(v)), nil
		}
		return append(dst, 0xce, byte(v>>24), byte(v>>16), byte(v>>8), byte(v)), nil
	case int16:
		if o.CompactInts {
			return appendCompactInt(dst, int(v)), nil
		}
		uv := uint16(v)
		return append(dst, 0xd1, byte(uv>>8), byte(uv)), nil
	case uint16:
		if o.CompactInts {
			return appendCompactUint(dst, uint(v)), nil
		}
		return append(dst, 0xcd, byte(v>>8), byte(v)), nil
	case int8:
		if o.CompactInts {
			return appendCompactInt(dst, int(v)), nil
		}
		return append(dst, 0xd0, byte(v)), nil
	case uint8:
		if o.CompactInts {
			return appendCompactUint(dst, uint(v)), nil
		}
		return append(dst, 0xcc, byte(v)), nil

	case map[string]any:
		dst, err := internal.AppendMapLen(dst, len(v))
		if err != nil {
			return nil, err
		}
		for k, sv := range v {
			dst, err = o.Encode(dst, k)
			if err != nil {
				return nil, err
			}
			dst, err = o.Encode(dst, sv)
			if err != nil {
				return nil, err
			}
		}
		return dst, nil
	case []any:
		dst, err := internal.AppendArrayLen(dst, len(v))
		if err != nil {
			return nil, err
		}
		for _, e := range v {
			dst, err = o.Encode(dst, e)
			if err != nil {
				return nil, err
			}
		}
		return dst, nil

	case time.Time:
		secs := v.Unix()
		nanos := v.Nanosecond()
		if secs>>34 != 0 {
			return append(dst, 0xc7, 12, 255, byte(nanos>>24), byte(nanos>>16), byte(nanos>>8), byte(nanos), byte(secs>>56), byte(secs>>48), byte(secs>>40), byte(secs>>32), byte(secs>>24), byte(secs>>16), byte(secs>>8), byte(secs)), nil
		}
		val := uint64(nanos<<34) | uint64(secs)
		if val&0xffffffff00000000 == 0 {
			return append(dst, 0xd6, 255, byte(val>>24), byte(val>>16), byte(val>>8), byte(val)), nil
		}
		return append(dst, 0xd7, 255, byte(val>>56), byte(val>>48), byte(val>>40), byte(val>>32), byte(val>>24), byte(val>>16), byte(val>>8), byte(val)), nil
	case Extension:
		return v.AppendMsgpack(dst)

	case interface{ MarshalMsgpack() ([]byte, error) }:
		b, err := v.MarshalMsgpack()
		if err != nil {
			return nil, err
		}
		return append(dst, b...), nil

	default:
		rv := reflect.ValueOf(v)
		switch rv.Kind() {
		case reflect.Pointer:
			return o.Encode(dst, rv.Elem().Interface())

		case reflect.Map:
			dst, err := internal.AppendMapLen(dst, rv.Len())
			if err != nil {
				return nil, err
			}
			iter := rv.MapRange()
			for iter.Next() {
				dst, err = o.Encode(dst, iter.Key().Interface())
				if err != nil {
					return nil, err
				}
				dst, err = o.Encode(dst, iter.Value().Interface())
				if err != nil {
					return nil, err
				}
			}
			return dst, nil
		case reflect.Slice, reflect.Array:
			dst, err := internal.AppendArrayLen(dst, rv.Len())
			if err != nil {
				return nil, err
			}
			for i := 0; rv.Len() > i; i++ {
				dst, err = o.Encode(dst, rv.Index(i).Interface())
				if err != nil {
					return nil, err
				}
			}
			return dst, nil

		default:
			return nil, fmt.Errorf("fastmsgpack.Encode: don't know how to encode %T", v)
		}
	}
}

func appendCompactInt(dst []byte, i int) []byte {
	if i < 128 {
		if i > -32 {
			// Values between -32 and 0 can be encoded as a negative fixint.
			// Values between 0 and 127 can be encoded as a positive fixint.
			return append(dst, byte(i))
		}
	}
	ui := uint(i)
	ai := i
	if i < 0 {
		ai = -i
	}
	if ai <= math.MaxInt8 {
		return append(dst, 0xd0, byte(ui))
	} else if ai <= math.MaxInt16 {
		return append(dst, 0xd1, byte(ui>>8), byte(ui))
	} else if ai <= math.MaxInt32 {
		return append(dst, 0xd2, byte(ui>>24), byte(ui>>16), byte(ui>>8), byte(ui))
	} else {
		return append(dst, 0xd3, byte(ui>>56), byte(ui>>48), byte(ui>>40), byte(ui>>32), byte(ui>>24), byte(ui>>16), byte(ui>>8), byte(ui))
	}
}

func appendCompactUint(dst []byte, i uint) []byte {
	if i < 128 {
		return append(dst, byte(i))
	} else if i <= math.MaxUint8 {
		return append(dst, 0xcc, byte(i))
	} else if i <= math.MaxUint16 {
		return append(dst, 0xcd, byte(i>>8), byte(i))
	} else if i <= math.MaxUint32 {
		return append(dst, 0xce, byte(i>>24), byte(i>>16), byte(i>>8), byte(i))
	} else {
		return append(dst, 0xcf, byte(i>>56), byte(i>>48), byte(i>>40), byte(i>>32), byte(i>>24), byte(i>>16), byte(i>>8), byte(i))
	}
}
