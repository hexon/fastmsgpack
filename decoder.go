// Package fastmsgpack is a msgpack decoder. See the README.
package fastmsgpack

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/alecthomas/unsafeslice"
)

var thisLibraryRequires64Bits int = math.MaxInt64

// Decode the given data (with the optional given dictionary).
// Any []byte and string in the return value might point into memory from the given data. Don't modify the input data until you're done with the return value.
func Decode(data []byte, dict []string) (_ any, retErr error) {
	defer func() {
		if r := recover(); r != nil {
			retErr = fmt.Errorf("decoder panicked, likely bad input: %v", r)
		}
	}()
	rc := resolveCall{
		Resolver: Resolver{
			dict: dict,
		},
		data: data,
	}
	return rc.resolveValue(), rc.err
}

// NewResolver prepares a new resolver. It can be reused for multiple Resolve calls.
// You can't query the same field twice. You can't even query a child of something else you request (e.g. both "person.properties" and "person.properties.age"). This is the only reason NewResolver might return an error.
// The dictionary is optional and can be nil.
func NewResolver(fields []string, dict []string) (*Resolver, error) {
	interests := map[string]any{}
	for n, f := range fields {
		sp := strings.Split(f, ".")
		dst := interests
		for len(sp) > 1 {
			v := dst[sp[0]]
			m, ok := v.(map[string]any)
			if !ok {
				if v != nil {
					return nil, errors.New("NewResolver: conflicting fields requested")
				}
				m = map[string]any{}
				dst[sp[0]] = m
			}
			dst = m
			sp = sp[1:]
		}
		if dst[sp[0]] != nil {
			return nil, errors.New("NewResolver: conflicting fields requested: " + f)
		}
		dst[sp[0]] = n
	}
	return &Resolver{interests, dict, len(fields)}, nil
}

type Resolver struct {
	interests map[string]any
	dict      []string
	numFields int
}

// Resolve scans through the given data and returns an array with the fields you've requested from this Resolver.
// Any []byte and string in the return value might point into memory from the given data. Don't modify the input data until you're done with the return value.
func (r *Resolver) Resolve(data []byte) (foundFields []any, retErr error) {
	defer func() {
		if r := recover(); r != nil {
			retErr = fmt.Errorf("decoder panicked, likely bad input: %v", r)
		}
	}()
	rc := resolveCall{
		Resolver: *r,
		data:     data,
		result:   make([]any, r.numFields),
	}
	rc.recurseMap(rc.interests, false)
	return rc.result, rc.err
}

type resolveCall struct {
	Resolver
	data   []byte
	result []any
	err    error
	offset int
}

func (rc *resolveCall) recurseMap(interests map[string]any, mustSkip bool) {
	b := rc.data[rc.offset]
	rc.offset++
	var elements int
	switch b {
	case 0xde:
		elements = int(rc.readUint16())
	case 0xdf:
		elements = int(rc.readUint32())
	default:
		if b&0b11110000 != 0b10000000 {
			rc.offset--
			rc.err = fmt.Errorf("encountered msgpack byte %d while expecting a map at offset %d", b, rc.offset)
			return
		}
		elements = int(b & 0b00001111)
	}
	sought := len(interests)
	for i := 0; elements > i; i++ {
		kv := rc.resolveValue()
		if rc.err != nil {
			return
		}
		var k string
		switch kv := kv.(type) {
		case string:
			k = kv
		case []byte:
			k = unsafeslice.StringFromByteSlice(kv)
		default:
			rc.err = errors.New("fastmsgpack doesn't support non-string keys in maps")
			return
		}
		switch x := interests[k].(type) {
		case int:
			rc.result[x] = rc.resolveValue()
			sought--
		case map[string]any:
			sought--
			rc.recurseMap(x, mustSkip || sought > 0)
		default:
			rc.skipValue()
		}
		if rc.err != nil {
			return
		}
		if sought == 0 {
			if mustSkip {
				for i++; elements > i; i++ {
					rc.skipValue()
					rc.skipValue()
				}
			}
			return
		}
	}
}

func (rc *resolveCall) resolveValue() any {
	b := rc.data[rc.offset]
	rc.offset++
	switch b & 0b11100000 {
	case 0b00000000, 0b00100000, 0b01000000, 0b01100000:
		return int(b)
	case 0b11100000:
		return int(int8(b))
	case 0b10100000:
		l := int(b & 0b00011111)
		rc.offset += l
		return unsafeslice.StringFromByteSlice(rc.data[rc.offset-l : rc.offset])
	case 0b10000000:
		if b&0b11110000 == 0b10010000 {
			return rc.resolveArray(int(b & 0b00001111))
		} else {
			return rc.resolveMap(int(b & 0b00001111))
		}
	}
	switch b {
	case 0xc0:
		return nil
	case 0xc2:
		return false
	case 0xc3:
		return true
	case 0xcc:
		return int(rc.readUint8())
	case 0xcd:
		return int(rc.readUint16())
	case 0xce:
		return int(rc.readUint32())
	case 0xcf:
		return int(rc.readUint64())
	case 0xd0:
		return int(int8(rc.readUint8()))
	case 0xd1:
		return int(int16(rc.readUint16()))
	case 0xd2:
		return int(int32(rc.readUint32()))
	case 0xd3:
		return int(int64(rc.readUint64()))
	case 0xca:
		return math.Float32frombits(rc.readUint32())
	case 0xcb:
		return math.Float64frombits(rc.readUint64())
	case 0xd9:
		l := int(rc.readUint8())
		return unsafeslice.StringFromByteSlice(rc.readBytes(l))
	case 0xda:
		l := int(rc.readUint16())
		return unsafeslice.StringFromByteSlice(rc.readBytes(l))
	case 0xdb:
		l := int(rc.readUint32())
		return unsafeslice.StringFromByteSlice(rc.readBytes(l))
	case 0xc4:
		l := int(rc.readUint8())
		return rc.readBytes(l)
	case 0xc5:
		l := int(rc.readUint16())
		return rc.readBytes(l)
	case 0xc6:
		l := int(rc.readUint32())
		return rc.readBytes(l)
	case 0xdc:
		return rc.resolveArray(int(rc.readUint16()))
	case 0xdd:
		return rc.resolveArray(int(rc.readUint32()))
	case 0xde:
		return rc.resolveMap(int(rc.readUint16()))
	case 0xdf:
		return rc.resolveMap(int(rc.readUint32()))
	case 0xd4:
		rc.offset += 2
		return rc.readExtension(rc.data[rc.offset-2], rc.data[rc.offset-1:rc.offset])
	case 0xd5:
		rc.offset += 3
		return rc.readExtension(rc.data[rc.offset-3], rc.data[rc.offset-2:rc.offset])
	case 0xd6:
		rc.offset += 5
		return rc.readExtension(rc.data[rc.offset-5], rc.data[rc.offset-4:rc.offset])
	case 0xd7:
		rc.offset += 9
		return rc.readExtension(rc.data[rc.offset-9], rc.data[rc.offset-8:rc.offset])
	case 0xd8:
		rc.offset += 17
		return rc.readExtension(rc.data[rc.offset-17], rc.data[rc.offset-16:rc.offset])
	case 0xc7:
		l := int(rc.readUint8())
		rc.offset += 1 + l
		return rc.readExtension(rc.data[rc.offset-l-1], rc.data[rc.offset-l:rc.offset])
	case 0xc8:
		l := int(rc.readUint16())
		rc.offset += 1 + l
		return rc.readExtension(rc.data[rc.offset-l-1], rc.data[rc.offset-l:rc.offset])
	case 0xc9:
		l := int(rc.readUint32())
		rc.offset += 1 + l
		return rc.readExtension(rc.data[rc.offset-l-1], rc.data[rc.offset-l:rc.offset])
	default:
		rc.offset--
		rc.err = fmt.Errorf("unexpected msgpack byte %d while decoding at offset %d", b, rc.offset)
		return rc.err
	}
}

func (rc *resolveCall) resolveArray(elements int) []any {
	ret := make([]any, elements)
	for i := 0; elements > i; i++ {
		ret[i] = rc.resolveValue()
		if rc.err != nil {
			return nil
		}
	}
	return ret
}

func (rc *resolveCall) resolveMap(elements int) map[string]any {
	ret := make(map[string]any, elements)
	for i := 0; elements > i; i++ {
		kv := rc.resolveValue()
		if rc.err != nil {
			return nil
		}
		var k string
		switch kv := kv.(type) {
		case string:
			k = kv
		case []byte:
			k = unsafeslice.StringFromByteSlice(kv)
		default:
			rc.err = errors.New("fastmsgpack doesn't support non-string keys in maps")
			return nil
		}
		ret[k] = rc.resolveValue()
		if rc.err != nil {
			return nil
		}
	}
	return ret
}

func (rc *resolveCall) readUint8() uint8 {
	rc.offset++
	return uint8(rc.data[rc.offset-1])
}

func (rc *resolveCall) readUint16() uint16 {
	rc.offset += 2
	return binary.BigEndian.Uint16(rc.data[rc.offset-2:])
}

func (rc *resolveCall) readUint32() uint32 {
	rc.offset += 4
	return binary.BigEndian.Uint32(rc.data[rc.offset-4:])
}

func (rc *resolveCall) readUint64() uint64 {
	rc.offset += 8
	return binary.BigEndian.Uint64(rc.data[rc.offset-8:])
}

func (rc *resolveCall) readBytes(n int) []byte {
	rc.offset += n
	return rc.data[rc.offset-n : rc.offset]
}

func (rc *resolveCall) readExtension(extType uint8, data []byte) any {
	switch int8(extType) {
	case -1:
		switch len(data) {
		case 4:
			return time.Unix(int64(binary.BigEndian.Uint32(data)), 0)
		case 8:
			n := binary.BigEndian.Uint64(data)
			return time.Unix(int64(n&0x00000003ffffffff), int64(n>>34))
		case 12:
			nsec := binary.BigEndian.Uint32(data[:4])
			sec := binary.BigEndian.Uint64(data[4:])
			return time.Unix(int64(sec), int64(nsec))
		}
		rc.err = fmt.Errorf("failed to decode timestamp of %d bytes", len(data))
		return rc.err

	case int8(math.MinInt8): // Interned string
		n, ok := decodeBytesToUint(data)
		if !ok {
			rc.err = errors.New("failed to decode index number of interned string")
			return rc.err
		}
		if n >= uint(len(rc.dict)) {
			rc.err = fmt.Errorf("interned string %d is out of bounds for the dict (%d entries)", n, len(rc.dict))
			return rc.err
		}
		return rc.dict[n]

	default:
		return Extension{Type: int8(extType), Data: data}
	}
}

func decodeBytesToUint(data []byte) (uint, bool) {
	switch len(data) {
	case 1:
		return uint(data[0]), true
	case 2:
		return uint(binary.BigEndian.Uint16(data)), true
	case 4:
		return uint(binary.BigEndian.Uint32(data)), true
	case 8:
		return uint(binary.BigEndian.Uint64(data)), true
	default:
		return 0, false
	}
}

type Extension struct {
	Data []byte
	Type int8
}

func (rc *resolveCall) skipValue() {
	b := rc.data[rc.offset]
	rc.offset++
	switch b & 0b11100000 {
	case 0b00000000, 0b00100000, 0b01000000, 0b01100000:
		return
	case 0b11100000:
		return
	case 0b10100000:
		rc.offset += int(b & 0b00011111)
		return
	case 0b10000000:
		if b&0b11110000 == 0b10010000 {
			rc.skipValues(int(b & 0b00001111))
		} else {
			rc.skipValues(2 * int(b&0b00001111))
		}
		return
	}
	switch b {
	case 0xc0:
	case 0xc2:
	case 0xc3:
	case 0xcc, 0xd0:
		rc.offset++
	case 0xcd, 0xd1:
		rc.offset += 2
	case 0xce, 0xd2, 0xca:
		rc.offset += 4
	case 0xcf, 0xd3, 0xcb:
		rc.offset += 8
	case 0xd9, 0xc4:
		rc.offset += int(rc.readUint8())
	case 0xda, 0xc5:
		rc.offset += int(rc.readUint16())
	case 0xdb, 0xc6:
		rc.offset += int(rc.readUint32())
	case 0xdc:
		rc.skipValues(int(rc.readUint16()))
	case 0xdd:
		rc.skipValues(int(rc.readUint32()))
	case 0xde:
		rc.skipValues(2 * int(rc.readUint16()))
	case 0xdf:
		rc.skipValues(2 * int(rc.readUint32()))
	case 0xd4:
		rc.offset += 2
	case 0xd5:
		rc.offset += 3
	case 0xd6:
		rc.offset += 5
	case 0xd7:
		rc.offset += 9
	case 0xd8:
		rc.offset += 17
	case 0xc7:
		rc.offset += 1 + int(rc.readUint8())
	case 0xc8:
		rc.offset += 1 + int(rc.readUint16())
	case 0xc9:
		rc.offset += 1 + int(rc.readUint32())
	default:
		rc.offset--
		rc.err = errors.New("unexpected msgpack byte while decoding: " + strconv.FormatInt(int64(b), 10))
	}
}

func (rc *resolveCall) skipValues(n int) {
	for i := 0; n > i; i++ {
		rc.skipValue()
	}
}
