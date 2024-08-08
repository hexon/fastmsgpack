package main

type MsgpackType struct {
	Name               string
	Byte               byte
	ByteEnd            byte
	DynamicLengthStart int
	DynamicLengthLen   int
	DataStart          int
	DataLen            int
	ExtTypeAt          int
	DataCast           string
	DataType           string
}

var (
	positiveFixint = MsgpackType{
		Name:      "positive fixint",
		Byte:      0x00,
		ByteEnd:   0x7f,
		DataStart: 0,
		DataLen:   1,
		DataCast:  "int($[0])",
		DataType:  "int",
	}
	fixmap = MsgpackType{
		Name:      "fixmap",
		Byte:      0x80,
		ByteEnd:   0x8f,
		DataStart: 0,
		DataLen:   1,
		DataCast:  "int($[0] & 0b00001111)",
		DataType:  "map",
	}
	fixarray = MsgpackType{
		Name:      "fixarray",
		Byte:      0x90,
		ByteEnd:   0x9f,
		DataStart: 0,
		DataLen:   1,
		DataCast:  "int($[0] & 0b00001111)",
		DataType:  "array",
	}
	fixstr = MsgpackType{
		Name:     "fixstr",
		Byte:     0xa0,
		ByteEnd:  0xbf,
		DataCast: "internal.UnsafeStringCast($)",
		DataType: "string",
	}
	negativeFixint = MsgpackType{
		Name:      "negative fixint",
		Byte:      0xe0,
		ByteEnd:   0xff,
		DataStart: 0,
		DataLen:   1,
		DataCast:  "int(int8($[0]))",
		DataType:  "int",
	}
	float_32 = MsgpackType{
		Name:      "float 32",
		Byte:      0xca,
		DataStart: 1,
		DataLen:   4,
		DataCast:  "math.Float32frombits(binary.BigEndian.Uint32($))",
		DataType:  "float32",
	}
	float_64 = MsgpackType{
		Name:      "float 64",
		Byte:      0xcb,
		DataStart: 1,
		DataLen:   8,
		DataCast:  "math.Float64frombits(binary.BigEndian.Uint64($))",
		DataType:  "float64",
	}
)

var types = []MsgpackType{
	positiveFixint,
	fixmap,
	fixarray,
	fixstr,
	{
		Name:     "nil",
		Byte:     0xc0,
		DataCast: "nil",
		DataType: "nil",
	},
	{
		Name:     "false",
		Byte:     0xc2,
		DataCast: "false",
		DataType: "bool",
	},
	{
		Name:     "true",
		Byte:     0xc3,
		DataCast: "true",
		DataType: "bool",
	},
	{
		Name:               "bin 8",
		Byte:               0xc4,
		DynamicLengthStart: 1,
		DynamicLengthLen:   1,
		DataCast:           "$",
		DataType:           "[]byte",
	},
	{
		Name:               "bin 16",
		Byte:               0xc5,
		DynamicLengthStart: 1,
		DynamicLengthLen:   2,
		DataCast:           "$",
		DataType:           "[]byte",
	},
	{
		Name:               "bin 32",
		Byte:               0xc6,
		DynamicLengthStart: 1,
		DynamicLengthLen:   4,
		DataCast:           "$",
		DataType:           "[]byte",
	},
	{
		Name:               "ext 8",
		Byte:               0xc7,
		ExtTypeAt:          2,
		DynamicLengthStart: 1,
		DynamicLengthLen:   1,
		DataCast:           "$",
		DataType:           "ext",
	},
	{
		Name:               "ext 16",
		Byte:               0xc8,
		ExtTypeAt:          3,
		DynamicLengthStart: 1,
		DynamicLengthLen:   2,
		DataCast:           "$",
		DataType:           "ext",
	},
	{
		Name:               "ext 32",
		Byte:               0xc9,
		ExtTypeAt:          5,
		DynamicLengthStart: 1,
		DynamicLengthLen:   4,
		DataCast:           "$",
		DataType:           "ext",
	},
	float_32,
	float_64,
	{
		Name:      "uint 8",
		Byte:      0xcc,
		DataStart: 1,
		DataLen:   1,
		DataCast:  "int($[0])",
		DataType:  "int",
	},
	{
		Name:      "uint 16",
		Byte:      0xcd,
		DataStart: 1,
		DataLen:   2,
		DataCast:  "int(binary.BigEndian.Uint16($))",
		DataType:  "int",
	},
	{
		Name:      "uint 32",
		Byte:      0xce,
		DataStart: 1,
		DataLen:   4,
		DataCast:  "int(binary.BigEndian.Uint32($))",
		DataType:  "int",
	},
	{
		Name:      "uint 64",
		Byte:      0xcf,
		DataStart: 1,
		DataLen:   8,
		DataCast:  "int(binary.BigEndian.Uint64($))",
		DataType:  "int",
	},
	{
		Name:      "int 8",
		Byte:      0xd0,
		DataStart: 1,
		DataLen:   1,
		DataCast:  "int(int8($[0]))",
		DataType:  "int",
	},
	{
		Name:      "int 16",
		Byte:      0xd1,
		DataStart: 1,
		DataLen:   2,
		DataCast:  "int(int16(binary.BigEndian.Uint16($)))",
		DataType:  "int",
	},
	{
		Name:      "int 32",
		Byte:      0xd2,
		DataStart: 1,
		DataLen:   4,
		DataCast:  "int(int32(binary.BigEndian.Uint32($)))",
		DataType:  "int",
	},
	{
		Name:      "int 64",
		Byte:      0xd3,
		DataStart: 1,
		DataLen:   8,
		DataCast:  "int(int64(binary.BigEndian.Uint64($)))",
		DataType:  "int",
	},
	{
		Name:      "fixext 1",
		Byte:      0xd4,
		ExtTypeAt: 1,
		DataStart: 2,
		DataLen:   1,
		DataCast:  "$",
		DataType:  "ext",
	},
	{
		Name:      "fixext 2",
		Byte:      0xd5,
		ExtTypeAt: 1,
		DataStart: 2,
		DataLen:   2,
		DataCast:  "$",
		DataType:  "ext",
	},
	{
		Name:      "fixext 4",
		Byte:      0xd6,
		ExtTypeAt: 1,
		DataStart: 2,
		DataLen:   4,
		DataCast:  "$",
		DataType:  "ext",
	},
	{
		Name:      "fixext 8",
		Byte:      0xd7,
		ExtTypeAt: 1,
		DataStart: 2,
		DataLen:   8,
		DataCast:  "$",
		DataType:  "ext",
	},
	{
		Name:      "fixext 16",
		Byte:      0xd8,
		ExtTypeAt: 1,
		DataStart: 2,
		DataLen:   16,
		DataCast:  "$",
		DataType:  "ext",
	},
	{
		Name:               "str 8",
		Byte:               0xd9,
		DynamicLengthStart: 1,
		DynamicLengthLen:   1,
		DataCast:           "internal.UnsafeStringCast($)",
		DataType:           "string",
	},
	{
		Name:               "str 16",
		Byte:               0xda,
		DynamicLengthStart: 1,
		DynamicLengthLen:   2,
		DataCast:           "internal.UnsafeStringCast($)",
		DataType:           "string",
	},
	{
		Name:               "str 32",
		Byte:               0xdb,
		DynamicLengthStart: 1,
		DynamicLengthLen:   4,
		DataCast:           "internal.UnsafeStringCast($)",
		DataType:           "string",
	},
	{
		Name:      "array 16",
		Byte:      0xdc,
		DataStart: 1,
		DataLen:   2,
		DataCast:  "int(binary.BigEndian.Uint16($))",
		DataType:  "array",
	},
	{
		Name:      "array 32",
		Byte:      0xdd,
		DataStart: 1,
		DataLen:   4,
		DataCast:  "int(binary.BigEndian.Uint32($))",
		DataType:  "array",
	},
	{
		Name:      "map 16",
		Byte:      0xde,
		DataStart: 1,
		DataLen:   2,
		DataCast:  "int(binary.BigEndian.Uint16($))",
		DataType:  "map",
	},
	{
		Name:      "map 32",
		Byte:      0xdf,
		DataStart: 1,
		DataLen:   4,
		DataCast:  "int(binary.BigEndian.Uint32($))",
		DataType:  "map",
	},
	negativeFixint,
}