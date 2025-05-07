package fastmsgpack

import (
	"encoding/binary"
	"errors"

	"github.com/hexon/fastmsgpack/internal"
)

// DisectFlavor decodes an encoded flavor extension. It is the inverse of a FlavorBuilder.
func DisectFlavor(data []byte) (selector uint, selectors []uint, cases [][]byte, elseClause []byte, _ error) {
	extType, extData, err := internal.DecodeExtensionHeader(data)
	if err != nil {
		return 0, nil, nil, nil, err
	}
	if extType != 18 {
		return 0, nil, nil, nil, errors.New("data is not a flavor extension")
	}
	selector64, sz := binary.Uvarint(extData)
	if sz <= 0 {
		return 0, nil, nil, nil, internal.ErrCorruptedFlavorData
	}
	remainder := extData[sz:]
	numCases, sz := binary.Uvarint(remainder)
	if sz <= 0 {
		return 0, nil, nil, nil, internal.ErrCorruptedFlavorData
	}
	remainder = remainder[sz:]
	hasElse := numCases&1 == 1
	numCases >>= 1
	selectors = make([]uint, numCases)
	cases = make([][]byte, numCases)
	for i := uint64(0); numCases > i; i++ {
		n, sz := binary.Uvarint(remainder)
		if sz <= 0 {
			return 0, nil, nil, nil, internal.ErrCorruptedFlavorData
		}
		remainder = remainder[sz:]
		selectors[i] = uint(n)
		n, sz = binary.Uvarint(remainder)
		if sz <= 0 {
			return 0, nil, nil, nil, internal.ErrCorruptedFlavorData
		}
		remainder = remainder[sz:]
		d := extData[n:]
		l, err := internal.ValueLength(d)
		if err != nil {
			return 0, nil, nil, nil, err
		}
		cases[i] = d[:l]
	}
	if hasElse {
		n, sz := binary.Uvarint(remainder)
		if sz <= 0 {
			return 0, nil, nil, nil, internal.ErrCorruptedFlavorData
		}
		d := extData[n:]
		l, err := internal.ValueLength(d)
		if err != nil {
			return 0, nil, nil, nil, err
		}
		elseClause = d[:l]
	}
	return uint(selector64), selectors, cases, elseClause, nil
}
