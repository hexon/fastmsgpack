package fastmsgpack

import (
	"bytes"
	"encoding/binary"
	"errors"
	"math"
	"slices"
	"sync"

	"github.com/hexon/fastmsgpack/internal"
)

var canonicalizerPool sync.Pool // 4kb []byte

func canonicalizerGetBuf() []byte {
	b := canonicalizerPool.Get()
	if b == nil {
		return make([]byte, 0, 4096)
	}
	return b.([]byte)[:0]
}

func Canonical(dst, data []byte, eo EncodeOptions, opts ...DecodeOption) ([]byte, error) {
	if cap(dst) < len(data) {
		d := make([]byte, 0, len(dst)+len(data))
		copy(d, dst)
		dst = d
	}
	c := canonicalizer{
		ret:           dst,
		encodeOptions: eo,
	}
	for _, o := range opts {
		o(&c.decodeOptions)
	}
	if _, err := c.canonicalize(data); err != nil {
		return nil, err
	}
	return c.ret, nil
}

type canonicalizer struct {
	ret           []byte
	decodeOptions internal.DecodeOptions
	encodeOptions EncodeOptions
}

func (c *canonicalizer) write(b []byte) error {
	c.ret = append(c.ret, b...)
	return nil
}

func (c *canonicalizer) probablyAppended(b []byte, err error) error {
	c.ret = b
	return err
}

func (c *canonicalizer) appendBytes(b []byte) error {
	return c.probablyAppended(encodeBytes(c.ret, b))
}

func (c *canonicalizer) appendString(s string) error {
	return c.probablyAppended(c.encodeOptions.encodeString(c.ret, s))
}

func (c *canonicalizer) appendInt(raw []byte, n int) error {
	if c.encodeOptions.CompactInts {
		c.ret = appendCompactInt(c.ret, n)
	} else {
		c.ret = append(c.ret, raw...)
	}
	return nil
}

func (c *canonicalizer) canonicalize_array(data []byte, offset, elements int) (int, error) {
	processed := make([][]byte, 0, elements)
	var neededLen int
	for i := 0; elements > i; i++ {
		sc := canonicalizer{
			ret:           canonicalizerGetBuf(),
			encodeOptions: c.encodeOptions,
			decodeOptions: c.decodeOptions,
		}
		defer canonicalizerPool.Put(sc.ret)
		consume, err := sc.canonicalize(data[offset:])
		if err != nil {
			return 0, err
		}
		offset += consume
		if bytes.Equal(sc.ret, canonicalVoidExtension) {
			continue
		}
		processed = append(processed, sc.ret)
		neededLen += len(sc.ret)
	}
	var err error
	c.ret, err = internal.AppendArrayLen(c.ret, len(processed))
	if err != nil {
		return 0, err
	}
	c.ret = slices.Grow(c.ret, neededLen)
	for _, b := range processed {
		c.ret = append(c.ret, b...)
	}
	return offset, nil
}

func (c *canonicalizer) canonicalize_map(data []byte, offset, elements int) (int, error) {
	reindex := make([]int, 0, elements)
	keys := make([][]byte, 0, elements)
	values := make([][]byte, 0, elements)
	var neededLen int
	for i := 0; elements > i; i++ {
		sc := canonicalizer{
			ret:           canonicalizerGetBuf(),
			encodeOptions: c.encodeOptions,
			decodeOptions: c.decodeOptions,
		}
		defer canonicalizerPool.Put(sc.ret)
		consume, err := sc.canonicalize(data[offset:])
		if err != nil {
			return 0, err
		}
		offset += consume
		key := sc.ret
		if bytes.Equal(key, canonicalVoidExtension) {
			consume, err := Size(data[consume:])
			if err != nil {
				return 0, err
			}
			offset += consume
			continue
		}
		sc = canonicalizer{
			ret:           canonicalizerGetBuf(),
			encodeOptions: c.encodeOptions,
			decodeOptions: c.decodeOptions,
		}
		defer canonicalizerPool.Put(sc.ret)
		consume, err = sc.canonicalize(data[offset:])
		if err != nil {
			return 0, err
		}
		offset += consume
		if bytes.Equal(sc.ret, canonicalVoidExtension) {
			continue
		}
		keys = append(keys, key)
		values = append(values, sc.ret)
		reindex = append(reindex, len(reindex))
		neededLen += len(key) + len(sc.ret)
	}
	slices.SortFunc(reindex, func(i, j int) int {
		return bytes.Compare(keys[i], keys[j])
	})
	var err error
	c.ret, err = internal.AppendMapLen(c.ret, len(keys))
	if err != nil {
		return 0, err
	}
	c.ret = slices.Grow(c.ret, neededLen)
	for _, i := range reindex {
		c.ret = append(c.ret, keys[i]...)
		c.ret = append(c.ret, values[i]...)
	}
	return offset, nil
}

var canonicalVoidExtension = []byte{0xc7, 0, 19}

func (c *canonicalizer) canonicalize_ext(data []byte, extType int8) error {
	switch extType {
	case -1:
		// Re-encode to see if we can encode it smaller without losing information.
		ts, err := internal.DecodeTimestamp(data)
		if err != nil {
			return err
		}
		return c.probablyAppended(encodeTime(c.ret, ts))

	case int8(math.MinInt8): // Interned string
		n, ok := internal.DecodeBytesToUint(data)
		if !ok {
			return errors.New("failed to decode index number of interned string")
		}
		if c.decodeOptions.Dict != nil && len(c.decodeOptions.Dict.Strings) > int(n) {
			return c.appendString(c.decodeOptions.Dict.Strings[n])
		}

	case 17: // Length-prefixed entry
		// Recurse and drop this extension
		_, err := c.canonicalize(data)
		return err

	case 18: // Flavor pick
		if j, err := internal.DecodeFlavorPick(data, c.decodeOptions); err == nil { // == nil
			_, err = c.canonicalize(data[j:])
			return err
		}
		return c.canonicalize_flavor(data)

	case 19: // Void
		c.ret = append(c.ret, canonicalVoidExtension...)
		return nil

	case 20: // Injection
		if b, err := internal.DecodeInjectionExtension(data, c.decodeOptions); err == nil { // == nil
			_, err = c.canonicalize(b)
			return err
		}

	default:
		// We don't know this extension, so just leave it unchanged.
	}
	return c.probablyAppended(Extension{Data: data, Type: extType}.AppendMsgpack(c.ret))
}

func (c *canonicalizer) canonicalize_flavor(data []byte) error {
	full := data
	selector, sz := binary.Uvarint(data)
	if sz <= 0 {
		return internal.ErrCorruptedFlavorData
	}
	data = data[sz:]
	numCases, sz := binary.Uvarint(data)
	if sz <= 0 {
		return internal.ErrCorruptedFlavorData
	}
	data = data[sz:]
	hasElse := numCases&1 == 1
	numCases >>= 1
	var cases []uint64
	var jumpTargets []uint64
	var reindex []int
	for numCases > 0 {
		n, sz := binary.Uvarint(data)
		if sz <= 0 {
			return internal.ErrCorruptedFlavorData
		}
		data = data[sz:]
		cases = append(cases, n)

		j, sz := binary.Uvarint(data)
		if sz <= 0 {
			return internal.ErrCorruptedFlavorData
		}
		data = data[sz:]
		jumpTargets = append(jumpTargets, j)
		reindex = append(reindex, len(reindex))
		numCases--
	}
	if hasElse {
		j, sz := binary.Uvarint(data)
		if sz <= 0 {
			return internal.ErrCorruptedFlavorData
		}
		jumpTargets = append(jumpTargets, j)
	}
	uniqueJumpTargets := slices.Clone(jumpTargets)
	slices.Sort(uniqueJumpTargets)
	uniqueJumpTargets = slices.Compact(uniqueJumpTargets)
	canon := make([][]byte, len(uniqueJumpTargets))
	for i, j := range uniqueJumpTargets {
		sc := canonicalizer{
			ret:           canonicalizerGetBuf(),
			encodeOptions: c.encodeOptions,
			decodeOptions: c.decodeOptions,
		}
		defer canonicalizerPool.Put(sc.ret)
		if _, err := sc.canonicalize(full[j:]); err != nil {
			return err
		}
		canon[i] = sc.ret
	}
	fb := NewFlavorBuilder(uint(selector))
	slices.SortFunc(reindex, func(i, j int) int {
		return int(cases[i]) - int(cases[j])
	})
	for _, i := range reindex {
		fb.AddCase(uint(cases[i]), canon[slices.Index(uniqueJumpTargets, jumpTargets[i])])
	}
	if len(jumpTargets) > len(cases) {
		fb.SetElse(canon[slices.Index(uniqueJumpTargets, jumpTargets[len(jumpTargets)-1])])
	}
	enc, err := fb.MarshalMsgpack()
	if err != nil {
		return err
	}
	c.ret = append(c.ret, enc...)
	return nil
}
