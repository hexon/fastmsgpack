package fastmsgpack

type ValueType uint8

const (
	TypeInvalid ValueType = iota
	TypeNil
	TypeBool
	TypeInt
	TypeFloat32
	TypeFloat64
	TypeString
	TypeBinary
	TypeArray
	TypeMap
	TypeTimestamp
	TypeUnknownExtension
)

func decodeType_ext(data []byte, extType int8) ValueType {
	switch extType {
	case -1: // Timestamp
		switch len(data) {
		case 4, 8, 12:
			return TypeTimestamp
		default:
			return TypeInvalid
		}

	case -128: // Interned string
		return TypeString

	case 17: // Length-prefixed entry
		return DecodeType(data)

	default:
		return TypeUnknownExtension
	}
}
