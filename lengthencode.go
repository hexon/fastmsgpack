package fastmsgpack

import (
	"fmt"
	"math"

	"github.com/hexon/fastmsgpack/internal"
)

// LengthEncode injects a length-encoding extension before every map and array to make skipping over it faster.
func LengthEncode(dst, data []byte) ([]byte, error) {
	le := lengthEncoder{
		data: data,
	}
	l, err := le.parseValue()
	if err != nil {
		return nil, err
	}
	if cap(dst) < l {
		dst = make([]byte, 0, l)
	}
	offset := 0
	for _, c := range le.chunks {
		switch c := c.(type) {
		case lengthEncoderCopy:
			dst = append(dst, data[offset:offset+int(c)]...)
			offset += int(c)
		case lengthEncoderSkip:
			offset += int(c)
		case lengthEncoderHeader:
			dst = appendLengthHeader(dst, *c)
		}
	}
	return dst, nil
}

type lengthEncoder struct {
	data   []byte
	chunks []lengthEncoderAction
	offset int
}

type lengthEncoderAction interface{}

type lengthEncoderCopy int
type lengthEncoderSkip int
type lengthEncoderHeader *int

func (le *lengthEncoder) parseValue() (int, error) {
	if l := internal.DecodeLengthPrefixExtension(le.data[le.offset:]); l > 0 {
		le.chunks = append(le.chunks, lengthEncoderSkip(l))
		le.offset += l
	}
	elements, consume, isMap := internal.DecodeMapLen(le.data[le.offset:])
	if !isMap {
		var ok bool
		elements, consume, ok = internal.DecodeArrayLen(le.data[le.offset:])
		if !ok {
			sz, err := Size(le.data[le.offset:])
			if err != nil {
				return 0, err
			}
			le.offset += sz
			le.appendCopy(sz)
			return sz, nil
		}
	}
	le.offset += consume
	h := lengthEncoderHeader(new(int))
	*h = consume
	le.chunks = append(le.chunks, h, lengthEncoderCopy(consume))
	if isMap {
		elements *= 2
	}
	for i := 0; elements > i; i++ {
		sz, err := le.parseValue()
		if err != nil {
			return 0, err
		}
		*h += sz
	}
	if *h > math.MaxUint32 {
		return 0, fmt.Errorf("fastmsgpack.LengthEncode: array/map data too long to encode (len %d)", *h)
	}
	hdrSize := sizeOfLengthHeader(*h)
	return hdrSize + *h, nil
}

func (le *lengthEncoder) appendCopy(sz int) {
	if len(le.chunks) > 0 {
		if l, ok := le.chunks[len(le.chunks)-1].(lengthEncoderCopy); ok {
			le.chunks[len(le.chunks)-1] = l + lengthEncoderCopy(sz)
			return
		}
	}
	le.chunks = append(le.chunks, lengthEncoderCopy(sz))
}

func sizeOfLengthHeader(wrapped int) int {
	switch wrapped {
	case 1, 2, 4, 8, 16:
		return 2
	}
	if wrapped <= math.MaxUint8 {
		return 3
	} else if wrapped <= math.MaxUint16 {
		return 4
	}
	return 6
}

func appendLengthHeader(dst []byte, wrapped int) []byte {
	switch wrapped {
	case 1:
		return append(dst, 0xd4, 17)
	case 2:
		return append(dst, 0xd5, 17)
	case 4:
		return append(dst, 0xd6, 17)
	case 8:
		return append(dst, 0xd7, 17)
	case 16:
		return append(dst, 0xd8, 17)
	}
	if wrapped <= math.MaxUint8 {
		return append(dst, 0xc7, byte(wrapped), 17)
	} else if wrapped <= math.MaxUint16 {
		return append(dst, 0xc8, byte(wrapped>>8), byte(wrapped), 17)
	}
	return append(dst, 0xc9, byte(wrapped>>24), byte(wrapped>>16), byte(wrapped>>8), byte(wrapped), 17)
}
