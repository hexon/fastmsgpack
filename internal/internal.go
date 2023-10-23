package internal

import "encoding/binary"

func DecodeBytesToUint(data []byte) (uint, bool) {
	switch len(data) {
	case 1:
		return uint(data[0]), true
	case 2:
		return uint(binary.BigEndian.Uint16(data)), true
	case 4:
		return uint(binary.BigEndian.Uint32(data)), true
	case 8:
		return uint(binary.BigEndian.Uint64(data)), true
	default:
		return 0, false
	}
}
