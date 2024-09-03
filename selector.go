package fastmsgpack

import (
	"encoding/binary"

	"github.com/hexon/fastmsgpack/internal"
)

// Select returns a new msgpack containing only the requested fields.
// The result is appended to dst and returned. dst can be nil.
func (r *Resolver) Select(dst, data []byte) (_ []byte, retErr error) {
	sc := selectCall{
		decoder:  NewDecoder(data, r.decodeOptions...),
		selected: dst,
	}
	if err := sc.selectFromMap(r.interests, false, nil); err != nil {
		return nil, err
	}
	return sc.selected, nil
}

type selectCall struct {
	decoder  *Decoder
	result   []any
	selected []byte
}

func (sc *selectCall) selectFromMap(interests map[string]any, mustSkip bool, uncommitted []byte) error {
	elements, err := sc.decoder.DecodeMapLen()
	if err != nil {
		if err == ErrVoid {
			if err := sc.decoder.Skip(); err != nil {
				return err
			}
		}
		return err
	}

	sc.selected = append(sc.selected, uncommitted...)
	sc.selected = append(sc.selected, 0xdf, 0, 0, 0, 0)
	lengthOffset := len(sc.selected) - 4
	newLength := 0

	sought := len(interests)
	for elements > 0 {
		elements--
		rawKey, err := sc.decoder.DecodeRaw()
		if err != nil {
			return err
		}
		k, _, err := internal.DecodeString(rawKey, sc.decoder.opt)
		if err != nil {
			return err
		}
		switch x := interests[k].(type) {
		case int:
			sought--
			v, err := sc.decoder.DecodeRaw()
			if err != nil {
				if err == ErrVoid {
					break
				}
				return err
			}
			sc.selected = append(sc.selected, rawKey...)
			sc.selected = append(sc.selected, v...)
			newLength++
		case map[string]any:
			sought--
			if err := sc.selectFromMap(x, mustSkip || sought > 0, rawKey); err != nil {
				if err == ErrVoid {
					break
				}
				return err
			}
			newLength++
		case subresolver:
			sought--
			if err := sc.selectFromArray(x, mustSkip || sought > 0, rawKey); err != nil {
				if err == ErrVoid {
					break
				}
				return err
			}
			newLength++
		default:
			if err := sc.decoder.Skip(); err != nil {
				return err
			}
		}
		if elements == 0 {
			break
		}
		if sought == 0 {
			if mustSkip {
				if err := sc.decoder.Break(); err != nil {
					return err
				}
			}
			break
		}
	}
	binary.BigEndian.PutUint32(sc.selected[lengthOffset:], uint32(newLength))
	return nil
}

func (sc *selectCall) selectFromArray(sub subresolver, mustSkip bool, uncommitted []byte) error {
	elements, err := sc.decoder.DecodeArrayLen()
	if err != nil {
		if err == ErrVoid {
			if err := sc.decoder.Skip(); err != nil {
				return err
			}
		}
		return err
	}
	sc.selected = append(sc.selected, uncommitted...)
	sc.selected = append(sc.selected, 0xdd, 0, 0, 0, 0)
	lengthOffset := len(sc.selected) - 4
	newLength := 0
	for i := 0; elements > i; i++ {
		if err := sc.selectFromMap(sub.interests, mustSkip || i < elements-1, nil); err != nil {
			if err == ErrVoid {
				continue
			}
			return err
		}
		newLength++
	}
	binary.BigEndian.PutUint32(sc.selected[lengthOffset:], uint32(newLength))
	return nil
}
