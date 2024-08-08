package fastmsgpack

import (
	"fmt"

	"github.com/alecthomas/unsafeslice"
	"github.com/hexon/fastmsgpack/internal"
)

// Size returns the number of bytes the first entry in the given msgpack data is.
func Size(data []byte) (int, error) {
	rc := resolveCall{
		data: data,
	}
	rc.skipValue()
	return rc.offset, rc.err
}

// SplitArray splits a msgpack array into the msgpack chunks of its components.
// The returned slices point into the given data.
func SplitArray(data []byte) ([][]byte, error) {
	data = data[internal.DecodeLengthPrefixExtension(data):]
	elements, consume, ok := internal.DecodeArrayLen(data)
	if !ok {
		return nil, fmt.Errorf("encountered msgpack byte %02x while expecting an array at offset %d", data[0], 0)
	}
	ret := make([][]byte, elements)
	rc := resolveCall{
		data:   data,
		offset: consume,
	}
	for i := 0; elements > i; i++ {
		start := rc.offset
		rc.skipValue()
		if rc.err != nil {
			return nil, rc.err
		}
		ret[i] = data[start:rc.offset]
	}
	return ret, nil
}

// SplitMap splits a msgpack map into string-keys and the msgpack-values. It does not decode the values.
// The returned slices point into the given data.
func SplitMap(data []byte, dict *Dict) ([]string, [][]byte, error) {
	data = data[internal.DecodeLengthPrefixExtension(data):]
	elements, consume, ok := internal.DecodeMapLen(data)
	if !ok {
		return nil, nil, fmt.Errorf("encountered msgpack byte %02x while expecting a map at offset %d", data[0], 0)
	}
	keys := make([]string, elements)
	values := make([][]byte, elements)
	rc := resolveCall{
		dict:   dict,
		data:   data,
		offset: consume,
	}
	for i := 0; elements > i; i++ {
		kv := rc.resolveValue()
		if rc.err != nil {
			return nil, nil, rc.err
		}
		switch kv := kv.(type) {
		case string:
			keys[i] = kv
		case []byte:
			keys[i] = unsafeslice.StringFromByteSlice(kv)
		default:
			return nil, nil, fmt.Errorf("fastmsgpack doesn't support non-string keys in maps (like %T)", kv)
		}
		start := rc.offset
		rc.skipValue()
		if rc.err != nil {
			return nil, nil, rc.err
		}
		values[i] = data[start:rc.offset]
	}
	return keys, values, nil
}
