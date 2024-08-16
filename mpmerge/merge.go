package mpmerge

import (
	"errors"
	"fmt"
	"slices"

	"github.com/hexon/fastmsgpack"
	"github.com/hexon/fastmsgpack/internal"
)

type Merger interface {
	descend(dst, data []byte, o fastmsgpack.EncodeOptions, readDict *fastmsgpack.Dict) ([]byte, error)
}

type StringMap struct {
	Changes map[string]Merger
}

type Array struct {
	Changes []Merger
}

type Each struct {
	Change Merger
}

type DeleteEntry struct{}

type Value struct {
	Value any
}

type EncodedValue []byte

func Merge(dst, data []byte, o fastmsgpack.EncodeOptions, readDict *fastmsgpack.Dict, m Merger) ([]byte, error) {
	return m.descend(dst, data, o, readDict)
}

func (m StringMap) descend(dst, data []byte, o fastmsgpack.EncodeOptions, readDict *fastmsgpack.Dict) ([]byte, error) {
	dec := fastmsgpack.NewDecoder(data, fastmsgpack.WithDict(readDict))
	elements, err := dec.DecodeMapLen()
	if err != nil {
		return nil, err
	}
	newSize := len(m.Changes)
	for _, sm := range m.Changes {
		if _, ok := sm.(DeleteEntry); ok {
			newSize--
		}
	}
	keys := make([]string, elements)
	values := make([][]byte, elements)
	for i := 0; elements > i; i++ {
		k, err := dec.DecodeString()
		if err != nil {
			return nil, err
		}
		keys[i] = k
		if _, def := m.Changes[k]; !def {
			newSize++
		}
		values[i], err = dec.DecodeRaw()
		if err != nil {
			return nil, err
		}
	}
	dst, err = internal.AppendMapLen(dst, newSize)
	if err != nil {
		return nil, err
	}
	for i := 0; elements > i; i++ {
		sm := m.Changes[keys[i]]
		if _, del := sm.(DeleteEntry); del {
			continue
		}
		dst, err = o.Encode(dst, keys[i])
		if err != nil {
			return nil, err
		}
		if sm == nil {
			dst = append(dst, values[i]...)
			continue
		}
		dst, err = sm.descend(dst, values[i], o, readDict)
		if err != nil {
			return nil, err
		}
	}
	for k, sm := range m.Changes {
		if slices.Contains(keys, k) {
			continue
		}
		if _, del := sm.(DeleteEntry); del {
			continue
		}
		dst, err = o.Encode(dst, k)
		if err != nil {
			return nil, err
		}
		switch sm := sm.(type) {
		case Value:
			dst, err = o.Encode(dst, sm.Value)
			if err != nil {
				return nil, err
			}
		case EncodedValue:
			dst = append(dst, sm...)
		case StringMap:
			dst, err = sm.descend(dst, []byte{0x80}, o, readDict)
			if err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("fastmsgpack/mpmerge: tried to apply %T to non-existent map key %q", sm, k)
		}
	}
	return dst, nil
}

func (m Array) descend(dst, data []byte, o fastmsgpack.EncodeOptions, readDict *fastmsgpack.Dict) ([]byte, error) {
	dec := fastmsgpack.NewDecoder(data, fastmsgpack.WithDict(readDict))
	elements, err := dec.DecodeArrayLen()
	if err != nil {
		return nil, err
	}
	newSize := len(m.Changes)
	if elements > len(m.Changes) {
		newSize += elements - len(m.Changes)
	}
	for _, sm := range m.Changes {
		if _, ok := sm.(DeleteEntry); ok {
			newSize--
		}
	}
	dst, err = internal.AppendArrayLen(dst, newSize)
	if err != nil {
		return nil, err
	}
	for i, sm := range m.Changes {
		if i >= elements {
			switch sm := sm.(type) {
			case Value:
				dst, err = o.Encode(dst, sm.Value)
				if err != nil {
					return nil, err
				}
			case EncodedValue:
				dst = append(dst, sm...)
			case StringMap:
				dst, err = sm.descend(dst, []byte{0x80}, o, readDict)
				if err != nil {
					return nil, err
				}
			default:
				return nil, fmt.Errorf("fastmsgpack/mpmerge: tried to apply %T to out of bounds array entry %d", sm, i)
			}
			continue
		}
		v, err := dec.DecodeRaw()
		if err != nil {
			return nil, err
		}
		switch sm.(type) {
		case nil:
			dst = append(dst, v...)
		case DeleteEntry:
		default:
			dst, err = sm.descend(dst, v, o, readDict)
			if err != nil {
				return nil, err
			}
		}
	}
	for i := len(m.Changes); elements > i; i++ {
		v, err := dec.DecodeRaw()
		if err != nil {
			return nil, err
		}
		dst = append(dst, v...)
	}
	return dst, nil
}

func (m Each) descend(dst, data []byte, o fastmsgpack.EncodeOptions, readDict *fastmsgpack.Dict) ([]byte, error) {
	dec := fastmsgpack.NewDecoder(data, fastmsgpack.WithDict(readDict))
	var elements int
	var isMap bool
	var err error
	switch t := dec.PeekType(); t {
	case fastmsgpack.TypeMap:
		isMap = true
		elements, err = dec.DecodeMapLen()
	case fastmsgpack.TypeArray:
		elements, err = dec.DecodeArrayLen()
	default:
		return nil, fmt.Errorf("encountered msgpack type %q while expecting a map or array", t.String())
	}
	if err != nil {
		return nil, err
	}
	switch m.Change.(type) {
	case nil:
		return nil, errors.New("fastmsgpack/mpmerge: this Each merger has nil Merger")
	case DeleteEntry:
		elements = 0
	}
	if isMap {
		dst, err = internal.AppendMapLen(dst, elements)
	} else {
		dst, err = internal.AppendArrayLen(dst, elements)
	}
	if err != nil {
		return nil, err
	}
	for i := 0; elements > i; i++ {
		if isMap {
			v, err := dec.DecodeRaw()
			if err != nil {
				return nil, err
			}
			dst = append(dst, v...)
		}
		v, err := dec.DecodeRaw()
		if err != nil {
			return nil, err
		}
		dst, err = m.Change.descend(dst, v, o, readDict)
		if err != nil {
			return nil, err
		}
	}
	return dst, nil
}

func (m Value) descend(dst, data []byte, o fastmsgpack.EncodeOptions, readDict *fastmsgpack.Dict) ([]byte, error) {
	return o.Encode(dst, m.Value)
}

func (m EncodedValue) descend(dst, data []byte, o fastmsgpack.EncodeOptions, readDict *fastmsgpack.Dict) ([]byte, error) {
	return append(dst, m...), nil
}

func (DeleteEntry) descend(dst, data []byte, o fastmsgpack.EncodeOptions, readDict *fastmsgpack.Dict) ([]byte, error) {
	return nil, errors.New("fastmsgpack/mpmerge: DeleteEntry must be used as a child of a map or array")
}
