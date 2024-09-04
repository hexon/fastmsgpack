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
	TypeFlavorSelector
	TypeVoid
	TypeInjection
	TypeUnknownExtension
)

func (t ValueType) String() string {
	switch t {
	case TypeInvalid:
		return "invalid"
	case TypeNil:
		return "nil"
	case TypeBool:
		return "bool"
	case TypeInt:
		return "int"
	case TypeFloat32:
		return "float32"
	case TypeFloat64:
		return "float64"
	case TypeString:
		return "string"
	case TypeBinary:
		return "binary"
	case TypeArray:
		return "array"
	case TypeMap:
		return "map"
	case TypeTimestamp:
		return "timestamp"
	case TypeFlavorSelector:
		return "flavor selector"
	case TypeVoid:
		return "void"
	case TypeInjection:
		return "injection"
	case TypeUnknownExtension:
		return "unknown extension"
	default:
		return "unknown ValueType constant"
	}
}

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

	case 18:
		return TypeFlavorSelector

	case 19:
		return TypeVoid

	case 20:
		return TypeInjection

	default:
		return TypeUnknownExtension
	}
}
