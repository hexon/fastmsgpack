package fastmsgpack

import "math"

var thisLibraryRequires64Bits int = math.MaxInt64

type Extension struct {
	Data []byte
	Type int8
}
