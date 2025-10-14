package fastmsgpack

import (
	"bytes"
	"encoding/binary"
	"errors"
	"math"
	"slices"

	"github.com/dennwc/varint"
)

// FlavorBuilder helps create an encoded extension 18.
// The flavor extension is like a switch statement inside your data.
//
// When decoding with WithFlavorSelector(1, 2) we will return x if it decodes the result of NewFlavorBuilder(1).AddCase(2, x).
//
// You are expected to cover all possible cases when building. Using WithFlavorSelector(1, 5) without having called AddCase(5, x) or SetElse() is undefined behavior.
type FlavorBuilder struct {
	cases      []flavorCase
	dataChunks [][]byte
	elseCase   *int
	field      uint
}

type flavorCase struct {
	match uint
	chunk int
}

func NewFlavorBuilder(field uint) FlavorBuilder {
	return FlavorBuilder{field: field}
}

func (f *FlavorBuilder) AddCase(match uint, b []byte) {
	f.cases = append(f.cases, flavorCase{match, f.dataChunk(b)})
}

func (f *FlavorBuilder) CopyCase(existing uint, newRefs ...uint) {
	chunk := -1
	for _, c := range f.cases {
		if c.match == existing {
			chunk = c.chunk
			break
		}
	}
	if chunk == -1 {
		panic("fastmsgpack.FlavorBuilder.CopyCase: Case to be copied does not exist")
	}
	for _, n := range newRefs {
		f.cases = append(f.cases, flavorCase{n, chunk})
	}
}

func (f *FlavorBuilder) SetElse(b []byte) {
	e := f.dataChunk(b)
	f.elseCase = &e
}

func (f *FlavorBuilder) dataChunk(b []byte) int {
	for i, dc := range f.dataChunks {
		if bytes.Equal(b, dc) {
			return i
		}
	}
	f.dataChunks = append(f.dataChunks, b)
	return len(f.dataChunks) - 1
}

func (f FlavorBuilder) extensionHeader(size int) ([6]byte, int, error) {
	var ret [6]byte
	if size > math.MaxUint32 {
		return ret, -1, errors.New("too big for a msgpack extension to encode")
	} else if size > math.MaxUint16 {
		ret[0] = 0xc9
		binary.BigEndian.PutUint32(ret[1:5], uint32(size))
		ret[5] = 18
		return ret, 6, nil
	} else if size > math.MaxUint8 {
		ret[0] = 0xc8
		binary.BigEndian.PutUint16(ret[1:3], uint16(size))
		ret[3] = 18
		return ret, 4, nil
	} else {
		switch size {
		case 1:
			ret[0] = 0xd4
		case 2:
			ret[0] = 0xd5
		case 4:
			ret[0] = 0xd6
		case 8:
			ret[0] = 0xd7
		case 16:
			ret[0] = 0xd8
		default:
			ret[0] = 0xc7
			ret[1] = byte(size)
			ret[2] = 18
			return ret, 3, nil
		}
		ret[1] = 18
		return ret, 2, nil
	}
}

func (f FlavorBuilder) flavorHeader(buf []byte) []byte {
	var jumpOffset int
	chunkOffsets := make([]uint64, len(f.dataChunks))
	for i, b := range f.dataChunks {
		chunkOffsets[i] = uint64(jumpOffset)
		jumpOffset += len(b)
	}

	buf = slices.Grow(buf, binary.MaxVarintLen64*(2+2*len(f.cases))+jumpOffset)
	lengthAtStart := len(buf)
	buf = binary.AppendUvarint(buf, uint64(f.field))
	numCases := len(f.cases) << 1
	if f.elseCase != nil {
		numCases |= 1
	}
	buf = binary.AppendUvarint(buf, uint64(numCases))
	staticHeader := buf

	headerLen := len(buf) - lengthAtStart
	for _, c := range f.cases {
		headerLen += varint.UvarintSize(uint64(c.match))
		headerLen += varint.UvarintSize(chunkOffsets[c.chunk])
	}
	if f.elseCase != nil {
		headerLen += varint.UvarintSize(chunkOffsets[*f.elseCase])
	}
	for {
		buf = staticHeader
		for _, c := range f.cases {
			buf = binary.AppendUvarint(buf, uint64(c.match))
			buf = binary.AppendUvarint(buf, uint64(headerLen)+chunkOffsets[c.chunk])
		}
		if f.elseCase != nil {
			buf = binary.AppendUvarint(buf, uint64(headerLen)+chunkOffsets[*f.elseCase])
		}
		if len(buf) <= headerLen+lengthAtStart {
			buf = buf[:lengthAtStart+headerLen]
			break
		}
		headerLen = len(buf) - lengthAtStart
	}
	return buf
}

func (f FlavorBuilder) AppendMsgpack(dst []byte) ([]byte, error) {
	if len(f.dataChunks) == 1 {
		return append(dst, f.dataChunks[0]...), nil
	}
	dataSize := 0
	for _, b := range f.dataChunks {
		dataSize += len(b)
	}
	var estimatedHeaderSize int
	// Reserve space for the extension header.
	if dataSize > math.MaxUint16 {
		dst = append(dst, 0xc1, 0xc1, 0xc1, 0xc1, 0xc1, 0xc1)
		estimatedHeaderSize = 6
	} else {
		// Most likely it's 3 or 4 bytes.
		dst = append(dst, 0xc1, 0xc1, 0xc1, 0xc1)
		estimatedHeaderSize = 4
	}
	lengthBefore := len(dst)
	dst = f.flavorHeader(dst)
	size := len(dst) - lengthBefore + dataSize
	h, headerSize, err := f.extensionHeader(size)
	if err != nil {
		return nil, err
	}
	if headerSize > estimatedHeaderSize {
		// We estimated the extension header too small and need to move the flavor header down a little.
		dst = dst[len(dst)+headerSize-estimatedHeaderSize:]
		copy(dst[lengthBefore+headerSize-estimatedHeaderSize:], dst[lengthBefore:])
	} else if headerSize < estimatedHeaderSize {
		copy(dst[lengthBefore+headerSize-estimatedHeaderSize:], dst[lengthBefore:])
		dst = dst[:len(dst)+headerSize-estimatedHeaderSize]
	}
	copy(dst[lengthBefore-estimatedHeaderSize:], h[:headerSize])
	for _, b := range f.dataChunks {
		dst = append(dst, b...)
	}
	return dst, nil
}

var sixZeroes = make([]byte, 6, 6)

func (f FlavorBuilder) MarshalMsgpack() ([]byte, error) {
	if len(f.dataChunks) == 1 {
		return f.dataChunks[0], nil
	}
	dataSize := 0
	for _, b := range f.dataChunks {
		dataSize += len(b)
	}
	buf := f.flavorHeader(sixZeroes)
	size := len(buf) - 6 + dataSize
	h, headerSize, err := f.extensionHeader(size)
	if err != nil {
		return nil, err
	}
	buf = buf[6-headerSize:]
	copy(buf, h[:headerSize])
	for _, b := range f.dataChunks {
		buf = append(buf, b...)
	}
	return buf, nil
}
