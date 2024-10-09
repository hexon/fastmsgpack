package fastmsgpack

import (
	"bytes"
	"encoding/binary"

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

func (f FlavorBuilder) MarshalMsgpack() ([]byte, error) {
	if len(f.dataChunks) == 1 {
		return f.dataChunks[0], nil
	}

	var jumpOffset int
	chunkOffsets := make([]uint64, len(f.dataChunks))
	for i, b := range f.dataChunks {
		chunkOffsets[i] = uint64(jumpOffset)
		jumpOffset += len(b)
	}

	buf := make([]byte, 0, binary.MaxVarintLen64*(2+2*len(f.cases))+jumpOffset)
	buf = binary.AppendUvarint(buf, uint64(f.field))
	numCases := len(f.cases) << 1
	if f.elseCase != nil {
		numCases |= 1
	}
	buf = binary.AppendUvarint(buf, uint64(numCases))
	staticHeader := buf

	headerLen := len(buf)
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
		if len(buf) <= headerLen {
			clear(buf[len(buf):headerLen])
			buf = buf[:headerLen]
			break
		}
		headerLen = len(buf)
	}
	for _, b := range f.dataChunks {
		buf = append(buf, b...)
	}

	return Extension{Type: 18, Data: buf}.MarshalMsgpack()
}
