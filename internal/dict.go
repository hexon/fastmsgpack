package internal

import (
	"errors"
	"fmt"
)

type Dict struct {
	Strings    []string
	Interfaces []any
}

func (d *Dict) LookupAny(n uint) (any, error) {
	if d == nil {
		return nil, errors.New("encountered interned string, but no dict was configured")
	}
	if n >= uint(len(d.Interfaces)) {
		n2 := n
		return nil, fmt.Errorf("interned string %d is out of bounds for the dict (%d entries)", n2, len(d.Interfaces))
	}
	return d.Interfaces[n], nil
}

func (d *Dict) LookupString(n uint) (string, error) {
	if d == nil {
		return "", errors.New("encountered interned string, but no dict was configured")
	}
	if n >= uint(len(d.Strings)) {
		n2 := n
		return "", fmt.Errorf("interned string %d is out of bounds for the dict (%d entries)", n2, len(d.Strings))
	}
	return d.Strings[n], nil
}
