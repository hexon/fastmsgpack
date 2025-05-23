package fastmsgpack

import (
	"github.com/hexon/fastmsgpack/internal"
)

// Size returns the number of bytes the first entry in the given msgpack data is.
func Size(data []byte) (int, error) {
	return internal.ValueLength(data)
}

// SplitArray splits a msgpack array into the msgpack chunks of its components.
// The returned slices point into the given data.
func SplitArray(data []byte) ([][]byte, error) {
	d := NewDecoder(data)
	elements, err := d.DecodeArrayLen()
	if err != nil {
		return nil, err
	}
	ret := make([][]byte, elements)
	for i := 0; elements > i; i++ {
		v, err := d.DecodeRaw()
		if err != nil {
			return nil, err
		}
		ret[i] = v
	}
	return ret, nil
}

// SplitMap splits a msgpack map into string-keys and the msgpack-values. It does not decode the values.
// The returned slices point into the given data.
func SplitMap(data []byte, dict *Dict) ([]string, [][]byte, error) {
	d := NewDecoder(data)
	elements, err := d.DecodeMapLen()
	if err != nil {
		return nil, nil, err
	}

	keys := make([]string, elements)
	values := make([][]byte, elements)
	for i := 0; elements > i; i++ {
		keys[i], err = d.DecodeString()
		if err != nil {
			return nil, nil, err
		}
		values[i], err = d.DecodeRaw()
		if err != nil {
			return nil, nil, err
		}
	}
	return keys, values, nil
}
