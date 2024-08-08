package fastmsgpack

import (
	"errors"
	"fmt"
)

// MakeDict prepares a dictionary.
func MakeDict(dict []string) *Dict {
	ret := &Dict{
		Strings:    dict,
		interfaces: make([]any, len(dict)),
	}
	for i, s := range dict {
		// Converting a string to an any does an allocation, so we do them all upfront and only once per dict.
		ret.interfaces[i] = s
	}
	return ret
}

// Dict is a dictionary for smaller msgpack. Instead of putting the string into the binary data, we use the number of the entry in the dictionary.
// Dictionaries should be the same between encoders and decoders. Adding new entries at the end is safe, as long as all decoders have the new dict before trying to decode newly encoded msgpack.
type Dict struct {
	Strings    []string
	interfaces []any
}

func (d *Dict) lookupAny(n uint) (any, error) {
	if d == nil {
		return nil, errors.New("encountered interned string, but no dict was configured")
	}
	if n >= uint(len(d.interfaces)) {
		n2 := n
		return nil, fmt.Errorf("interned string %d is out of bounds for the dict (%d entries)", n2, len(d.interfaces))
	}
	return d.interfaces[n], nil
}

func (d *Dict) lookupString(n uint) (string, error) {
	if d == nil {
		return "", errors.New("encountered interned string, but no dict was configured")
	}
	if n >= uint(len(d.Strings)) {
		n2 := n
		return "", fmt.Errorf("interned string %d is out of bounds for the dict (%d entries)", n2, len(d.Strings))
	}
	return d.Strings[n], nil
}
