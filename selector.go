package fastmsgpack

import (
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/hexon/fastmsgpack/internal"
)

// Select returns a new msgpack containing only the requested fields.
// The result is appended to dst and returned. dst can be nil.
func (r *Resolver) Select(dst, data []byte) (_ []byte, retErr error) {
	defer func() {
		if r := recover(); r != nil {
			retErr = fmt.Errorf("decoder panicked, likely bad input: %v", r)
		}
	}()
	rc := resolveCall{
		dict:     r.dict,
		data:     data,
		selected: dst,
	}
	rc.selectFromMap(r.interests, false)
	return rc.selected, rc.err
}

func (rc *resolveCall) selectFromMap(interests map[string]any, mustSkip bool) {
	fastSkipStart := -1
	if l := internal.DecodeLengthPrefixExtension(rc.data[rc.offset:]); l > 0 {
		fastSkipStart = rc.offset
		rc.offset += l
	}
	elements, consume, ok := internal.DecodeMapLen(rc.data[rc.offset:])
	if !ok {
		rc.err = fmt.Errorf("encountered msgpack byte %02x while expecting a map at offset %d", rc.data[rc.offset], rc.offset)
		return
	}
	rc.offset += consume

	rc.selected = append(rc.selected, 0xdf, 0, 0, 0, 0)
	lengthOffset := len(rc.selected) - 4
	newLength := 0

	sought := len(interests)
	for i := 0; elements > i; i++ {
		keyAt := rc.offset
		kv := rc.resolveValue()
		if rc.err != nil {
			return
		}
		var k string
		switch kv := kv.(type) {
		case string:
			k = kv
		case []byte:
			k = internal.UnsafeStringCast(kv)
		default:
			rc.err = errors.New("fastmsgpack doesn't support non-string keys in maps")
			return
		}
		switch x := interests[k].(type) {
		case int:
			rc.skipValue()
			rc.selected = append(rc.selected, rc.data[keyAt:rc.offset]...)
			sought--
			newLength++
		case map[string]any:
			rc.selected = append(rc.selected, rc.data[keyAt:rc.offset]...)
			sought--
			rc.selectFromMap(x, mustSkip || sought > 0)
			newLength++
		case subresolver:
			rc.selected = append(rc.selected, rc.data[keyAt:rc.offset]...)
			sought--
			rc.selectFromArray(x, mustSkip || sought > 0)
			newLength++
		default:
			rc.skipValue()
		}
		if rc.err != nil {
			return
		}
		if sought == 0 {
			if mustSkip {
				i++
				if fastSkipStart != -1 && elements > i {
					// This map was wrapped with a length-encoding. Jump back to the beginning, so we skip over the entire object at once.
					rc.offset = fastSkipStart
					rc.skipValue()
					break
				}
				for ; elements > i; i++ {
					rc.skipValue()
					rc.skipValue()
				}
			}
			break
		}
	}
	binary.BigEndian.PutUint32(rc.selected[lengthOffset:], uint32(newLength))
}

func (rc *resolveCall) selectFromArray(sub subresolver, mustSkip bool) {
	rc.offset += internal.DecodeLengthPrefixExtension(rc.data[rc.offset:])
	elements, consume, ok := internal.DecodeArrayLen(rc.data[rc.offset:])
	if !ok {
		rc.err = fmt.Errorf("encountered msgpack byte %02x while expecting an array at offset %d", rc.data[rc.offset], rc.offset)
		return
	}
	rc.selected = append(rc.selected, rc.data[rc.offset:][:consume]...)
	rc.offset += consume
	for i := 0; elements > i; i++ {
		rc.selectFromMap(sub.interests, mustSkip || i < elements-1)
		if rc.err != nil {
			return
		}
	}
}
