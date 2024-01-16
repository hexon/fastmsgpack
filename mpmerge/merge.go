package mpmerge

import (
	"errors"
	"fmt"

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
	data = data[internal.DecodeLengthPrefixExtension(data):]
	elements, consume, ok := internal.DecodeMapLen(data)
	if !ok {
		return nil, fmt.Errorf("encountered msgpack byte %02x while expecting a map", data[0])
	}
	data = data[consume:]
	newSize := len(m.Changes)
	for _, sm := range m.Changes {
		if _, ok := sm.(DeleteEntry); ok {
			newSize--
		}
	}
	keys := make([]string, elements)
	values := make([][]byte, elements)
	for i := 0; elements > i; i++ {
		sz, err := fastmsgpack.Size(data)
		if err != nil {
			return nil, err
		}
		k, err := fastmsgpack.Decode(data[:sz], readDict)
		if err != nil {
			return nil, err
		}
		data = data[sz:]
		keys[i] = k.(string)
		if _, def := m.Changes[k.(string)]; !def {
			newSize++
		}
		sz, err = fastmsgpack.Size(data)
		if err != nil {
			return nil, err
		}
		values[i] = data[:sz]
		data = data[sz:]
	}
	dst, err := internal.AppendMapLen(dst, newSize)
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
outer:
	for k, sm := range m.Changes {
		for _, ek := range keys {
			if k == ek {
				continue outer
			}
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
		default:
			return nil, fmt.Errorf("fastmsgpack/mpmerge: tried to apply %T to non-existent map key %q", sm, k)
		}
	}
	return dst, nil
}

func (m Array) descend(dst, data []byte, o fastmsgpack.EncodeOptions, readDict *fastmsgpack.Dict) ([]byte, error) {
	data = data[internal.DecodeLengthPrefixExtension(data):]
	elements, consume, ok := internal.DecodeArrayLen(data)
	if !ok {
		return nil, fmt.Errorf("encountered msgpack byte %02x while expecting an array", data[0])
	}
	data = data[consume:]
	newSize := len(m.Changes)
	if elements > len(m.Changes) {
		newSize += elements - len(m.Changes)
	}
	for _, sm := range m.Changes {
		if _, ok := sm.(DeleteEntry); ok {
			newSize--
		}
	}
	dst, err := internal.AppendArrayLen(dst, newSize)
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
			default:
				return nil, fmt.Errorf("fastmsgpack/mpmerge: tried to apply %T to out of bounds array entry %d", sm, i)
			}
		}
		sz, err := fastmsgpack.Size(data)
		if err != nil {
			return nil, err
		}
		switch sm.(type) {
		case nil:
			dst = append(dst, data[:sz]...)
		case DeleteEntry:
		default:
			dst, err = sm.descend(dst, data[:sz], o, readDict)
			if err != nil {
				return nil, err
			}
		}
		data = data[sz:]
	}
	for i := len(m.Changes); elements > i; i++ {
		sz, err := fastmsgpack.Size(data)
		if err != nil {
			return nil, err
		}
		dst = append(dst, data[:sz]...)
		data = data[sz:]
	}
	return dst, nil
}

func (m Each) descend(dst, data []byte, o fastmsgpack.EncodeOptions, readDict *fastmsgpack.Dict) ([]byte, error) {
	data = data[internal.DecodeLengthPrefixExtension(data):]
	elements, consume, isMap := internal.DecodeMapLen(data)
	if !isMap {
		var ok bool
		elements, consume, ok = internal.DecodeArrayLen(data)
		if !ok {
			return nil, fmt.Errorf("encountered msgpack byte %02x while expecting a map or array", data[0])
		}
	}
	data = data[consume:]
	switch m.Change.(type) {
	case nil:
		return nil, errors.New("fastmsgpack/mpmerge: this Each merger has nil Merger")
	case DeleteEntry:
		elements = 0
	}
	var err error
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
			sz, err := fastmsgpack.Size(data)
			if err != nil {
				return nil, err
			}
			dst = append(dst, data[:sz]...)
			data = data[sz:]
		}
		sz, err := fastmsgpack.Size(data)
		if err != nil {
			return nil, err
		}
		dst, err = m.Change.descend(dst, data[:sz], o, readDict)
		if err != nil {
			return nil, err
		}
		data = data[sz:]
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
