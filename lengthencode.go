package fastmsgpack

import (
	"fmt"
	"math"
	"sync"

	"github.com/hexon/fastmsgpack/internal"
)

var lengthEncoderPool = sync.Pool{New: func() any { return make([]lengthEncoderAction, 128) }}

// LengthEncode injects a length-encoding extension before every map and array to make skipping over it faster.
// The result is appended to dst and returned. dst can be nil.
func LengthEncode(dst, data []byte) ([]byte, error) {
	le := lengthEncoder{
		data:         data,
		currentChunk: lengthEncoderPool.Get().([]lengthEncoderAction)[:0],
	}
	l, err := le.parseValue()
	le.listOfChunks = append(le.listOfChunks, le.currentChunk)
	if err != nil {
		for _, chunks := range le.listOfChunks {
			lengthEncoderPool.Put(chunks)
		}
		return nil, err
	}
	if cap(dst) < l {
		d := make([]byte, 0, len(dst)+l)
		copy(d, dst)
		dst = d
	}
	offset := 0
	for _, chunks := range le.listOfChunks {
		for _, c := range chunks {
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
		lengthEncoderPool.Put(chunks)
	}
	return dst, nil
}

type lengthEncoder struct {
	data         []byte
	listOfChunks [][]lengthEncoderAction
	currentChunk []lengthEncoderAction
	offset       int
}

type lengthEncoderAction interface{}
type lengthEncoderCopy int
type lengthEncoderSkip int
type lengthEncoderHeader *int

func (le *lengthEncoder) parseValue() (int, error) {
	if l := internal.DecodeLengthPrefixExtension(le.data[le.offset:]); l > 0 {
		le.appendAction(lengthEncoderSkip(l))
		le.offset += l
	}
	elements, consume, isMap := internal.DecodeUnwrappedMapLen(le.data[le.offset:])
	if !isMap {
		var ok bool
		elements, consume, ok = internal.DecodeUnwrappedArrayLen(le.data[le.offset:])
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
	le.appendAction(h)
	le.appendAction(lengthEncoderCopy(consume))
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
	if len(le.currentChunk) > 0 {
		if l, ok := le.currentChunk[len(le.currentChunk)-1].(lengthEncoderCopy); ok {
			le.currentChunk[len(le.currentChunk)-1] = l + lengthEncoderCopy(sz)
			return
		}
	}
	le.appendAction(lengthEncoderCopy(sz))
}

func (le *lengthEncoder) appendAction(a lengthEncoderAction) {
	if len(le.currentChunk) == cap(le.currentChunk) {
		le.listOfChunks = append(le.listOfChunks, le.currentChunk)
		le.currentChunk = lengthEncoderPool.Get().([]lengthEncoderAction)[:0]
	}
	le.currentChunk = append(le.currentChunk, a)
}

func sizeOfLengthHeader(wrapped int) int {
	if wrapped <= math.MaxUint8 {
		switch wrapped {
		case 1, 2, 4, 8, 16:
			return 2
		}
		return 3
	} else if wrapped <= math.MaxUint16 {
		return 4
	}
	return 6
}

func appendLengthHeader(dst []byte, wrapped int) []byte {
	if wrapped <= math.MaxUint8 {
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
		return append(dst, 0xc7, byte(wrapped), 17)
	} else if wrapped <= math.MaxUint16 {
		return append(dst, 0xc8, byte(wrapped>>8), byte(wrapped), 17)
	}
	return append(dst, 0xc9, byte(wrapped>>24), byte(wrapped>>16), byte(wrapped>>8), byte(wrapped), 17)
}
