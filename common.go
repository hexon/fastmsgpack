package fastmsgpack

import (
	"fmt"
	"math"
)

var thisLibraryRequires64Bits int = math.MaxInt64

type Extension struct {
	Data []byte
	Type int8
}

func (e Extension) AppendMsgpack(dst []byte) ([]byte, error) {
	switch len(e.Data) {
	case 1:
		return append(dst, 0xd4, byte(e.Type), e.Data[0]), nil
	case 2:
		return append(dst, 0xd5, byte(e.Type), e.Data[0], e.Data[1]), nil
	case 4:
		return append(dst, 0xd6, byte(e.Type), e.Data[0], e.Data[1], e.Data[2], e.Data[3]), nil
	case 8:
		dst = append(dst, 0xd7, byte(e.Type))
	case 16:
		dst = append(dst, 0xd8, byte(e.Type))
	default:
		if len(e.Data) <= math.MaxUint8 {
			dst = append(dst, 0xc7, byte(len(e.Data)), byte(e.Type))
		} else if len(e.Data) <= math.MaxUint16 {
			dst = append(dst, 0xc8, byte(len(e.Data)>>8), byte(len(e.Data)), byte(e.Type))
		} else if len(e.Data) <= math.MaxUint32 {
			dst = append(dst, 0xc9, byte(len(e.Data)>>24), byte(len(e.Data)>>16), byte(len(e.Data)>>8), byte(len(e.Data)), byte(e.Type))
		} else {
			return nil, fmt.Errorf("fastmsgpack.Encode: extension data too long to encode (len %d)", len(e.Data))
		}
	}
	return append(dst, e.Data...), nil
}

func (e Extension) MarshalMsgpack() ([]byte, error) {
	return e.AppendMsgpack(nil)
}
