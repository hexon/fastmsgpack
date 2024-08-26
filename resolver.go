// Package fastmsgpack is a msgpack decoder. See the README.
package fastmsgpack

import (
	"errors"
	"strings"

	"github.com/Jille/genericz/slicez"
	"github.com/hexon/fastmsgpack/internal"
)

// Decode the given data (with the optional given dictionary).
// Any []byte and string in the return value might point into memory from the given data. Don't modify the input data until you're done with the return value.
// The dictionary is optional and can be nil.
func Decode(data []byte, opts ...DecodeOption) (any, error) {
	var opt internal.DecodeOptions
	for _, o := range opts {
		o(&opt)
	}
	v, _, err := decodeValue(data, opt)
	return v, err
}

// NewResolver prepares a new resolver. It can be reused for multiple Resolve calls.
// You can't query the same field twice. You can't even query a child of something else you request (e.g. both "person.properties" and "person.properties.age"). This is the only reason NewResolver might return an error.
// The dictionary is optional and can be nil.
func NewResolver(fields []string, opts ...DecodeOption) (*Resolver, error) {
	interests := map[string]any{}
	r := &Resolver{interests, opts, len(fields)}
	for n, f := range fields {
		if err := r.addField(f, n); err != nil {
			return nil, err
		}
	}
	return r, nil
}

type subresolver struct {
	interests   map[string]any
	destination int
	numFields   int
}

type Resolver struct {
	interests     map[string]any
	decodeOptions []DecodeOption
	numFields     int
}

// AddArrayResolver allows resolving inside array fields. For example like this pseudocode: `r.AddArrayResolve("person.addresses", NewResolver(["street"]))`.
// It returns the offset in the return value from Resolve(), which will be of type [][]any.
// AddArrayResolver can not be called concurrently with itself or Resolve.
// The dict that was given to the subresolver is not used.
//
//	r, err := NewResolver([]string{"person.properties.age"}, nil)
//	sub, err := NewResolver([]string{"street", "number"}, nil)
//	addrOffset, err := r.AddArrayResolver("person.addresses", sub)
//	found, err := r.Resolve(msgpackData)
//	age := found[0] // e.g. 5
//	addresses := found[addrOffset] // e.g. [][]any{[]any{"Main Street", 1}, []any{"Second Street", 2}}
func (r *Resolver) AddArrayResolver(field string, sub *Resolver) (int, error) {
	dst := r.numFields
	if err := r.addField(field, subresolver{sub.interests, dst, sub.numFields}); err != nil {
		return -1, err
	}
	r.numFields++
	return dst, nil
}

func (r *Resolver) addField(field string, what any) error {
	sp := strings.Split(field, ".")
	dst := r.interests
	for len(sp) > 1 {
		v := dst[sp[0]]
		m, ok := v.(map[string]any)
		if !ok {
			if v != nil {
				return errors.New("NewResolver: conflicting fields requested")
			}
			m = map[string]any{}
			dst[sp[0]] = m
		}
		dst = m
		sp = sp[1:]
	}
	if dst[sp[0]] != nil {
		return errors.New("NewResolver: conflicting fields requested: " + field)
	}
	dst[sp[0]] = what
	return nil
}

type SubresolverDescription struct {
	Fields       []string
	Subresolvers map[string]SubresolverDescription
	Index        int
}

// Describe returns which fields and subresolvers were registered to this Resolver.
// The returned values should not be modified.
func (r *Resolver) Describe() ([]string, map[string]SubresolverDescription) {
	fields := make([]string, r.numFields)
	subs := map[string]SubresolverDescription{}
	recurseInterests(fields, subs, r.interests, "")
	fields = fields[:len(fields)-len(subs)]
	return fields, subs
}

func recurseInterests(fields []string, subs map[string]SubresolverDescription, i any, prefix string) {
	switch i := i.(type) {
	case int:
		fields[i] = prefix
	case map[string]any:
		if prefix != "" {
			prefix += "."
		}
		for k, v := range i {
			recurseInterests(fields, subs, v, prefix+k)
		}
	case subresolver:
		sd := SubresolverDescription{
			Index:        i.destination,
			Fields:       make([]string, i.numFields),
			Subresolvers: map[string]SubresolverDescription{},
		}
		recurseInterests(sd.Fields, sd.Subresolvers, i.interests, "")
		sd.Fields = sd.Fields[:len(sd.Fields)-len(sd.Subresolvers)]
		subs[prefix] = sd
	}
}

// Resolve scans through the given data and returns an array with the fields you've requested from this Resolver.
// Any []byte and string in the return value might point into memory from the given data. Don't modify the input data until you're done with the return value.
func (r *Resolver) Resolve(data []byte, opts ...DecodeOption) (foundFields []any, retErr error) {
	rc := resolveCall{
		decoder: NewDecoder(data, slicez.Concat(r.decodeOptions, opts)...),
		result:  make([]any, r.numFields),
	}
	if err := rc.recurseMap(r.interests, false); err != nil {
		return nil, err
	}
	return rc.result, nil
}

type resolveCall struct {
	decoder *Decoder
	result  []any
}

func (rc *resolveCall) recurseMap(interests map[string]any, mustSkip bool) error {
	elements, err := rc.decoder.DecodeMapLen()
	if err != nil {
		if err == ErrVoid {
			if err := rc.decoder.Skip(); err != nil {
				return err
			}
		}
		return err
	}

	sought := len(interests)
	for elements > 0 {
		elements--
		k, err := rc.decoder.DecodeString()
		if err != nil {
			if err == ErrVoid {
				if err := rc.decoder.Skip(); err != nil {
					return err
				}
				if err := rc.decoder.Skip(); err != nil {
					return err
				}
				continue
			}
			return err
		}
		switch x := interests[k].(type) {
		case int:
			rc.result[x], err = rc.decoder.DecodeValue()
			sought--
		case map[string]any:
			sought--
			err = rc.recurseMap(x, mustSkip || sought > 0)
		case subresolver:
			sought--
			err = rc.recurseArray(x, mustSkip || sought > 0)
		default:
			err = rc.decoder.Skip()
		}
		if err == ErrVoid {
			err = rc.decoder.Skip()
		}
		if err != nil {
			return err
		}
		if elements == 0 {
			break
		}
		if sought == 0 {
			if mustSkip {
				return rc.decoder.Break()
			}
			return nil
		}
	}
	return nil
}

func (rc *resolveCall) recurseArray(sub subresolver, mustSkip bool) error {
	elements, err := rc.decoder.DecodeArrayLen()
	if err != nil {
		if err == ErrVoid {
			if err := rc.decoder.Skip(); err != nil {
				return err
			}
		}
		return err
	}
	parentResults := rc.result
	results := make([][]any, elements)
	var voided int
	for i := 0; elements > i; i++ {
		rc.result = make([]any, sub.numFields)
		if err := rc.recurseMap(sub.interests, mustSkip || i < elements-1); err != nil {
			if err == ErrVoid {
				voided++
				continue
			}
			return err
		}
		results[i-voided] = rc.result
	}
	rc.result = parentResults
	rc.result[sub.destination] = results[:len(results)-voided]
	return nil
}
