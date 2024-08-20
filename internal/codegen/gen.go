package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/format"
	"log"
	"os"
	"regexp"
	"strings"
)

var (
	internalOut = flag.String("internal-out", "../../internal/generated.go", "")
	publicOut   = flag.String("public-out", "../../generated.go", "")
	jsonOut     = flag.String("json-out", "../../msgpackconverter/generated.go", "")
)

func main() {
	flag.Parse()
	{
		var buf bytes.Buffer
		fmt.Fprintf(&buf, "// Code generated by internal/codegen. DO NOT EDIT.\n")
		fmt.Fprintf(&buf, "\n")
		fmt.Fprintf(&buf, "package internal\n")
		fmt.Fprintf(&buf, "\n")
		fmt.Fprintf(&buf, "import (\n")
		fmt.Fprintf(&buf, "	%q\n", "encoding/binary")
		fmt.Fprintf(&buf, "	%q\n", "errors")
		fmt.Fprintf(&buf, "	%q\n", "fmt")
		fmt.Fprintf(&buf, "	%q\n", "math")
		fmt.Fprintf(&buf, "	%q\n", "time")
		fmt.Fprintf(&buf, ")\n")
		generate(&buf, "void", "ValueLength")
		generate(&buf, "_desc", "DescribeValue")
		generate(&buf, "string", "DecodeString")
		generate(&buf, "int", "DecodeInt")
		generate(&buf, "float32", "DecodeFloat32")
		generate(&buf, "float64", "DecodeFloat64")
		generate(&buf, "bool", "DecodeBool")
		generate(&buf, "time", "DecodeTime")

		b := bytes.ReplaceAll(buf.Bytes(), []byte("internal."), nil)
		formatted, err := format.Source(b)
		if err != nil {
			formatted = buf.Bytes()
		}
		if err := os.WriteFile(*internalOut, formatted, 0644); err != nil {
			log.Fatal(err)
		}
	}

	{
		var buf bytes.Buffer
		fmt.Fprintf(&buf, "// Code generated by internal/codegen. DO NOT EDIT.\n")
		fmt.Fprintf(&buf, "\n")
		fmt.Fprintf(&buf, "package fastmsgpack\n")
		fmt.Fprintf(&buf, "\n")
		fmt.Fprintf(&buf, "import (\n")
		fmt.Fprintf(&buf, "	%q\n", "encoding/binary")
		fmt.Fprintf(&buf, "	%q\n", "errors")
		fmt.Fprintf(&buf, "	%q\n", "math")
		fmt.Fprintf(&buf, "\n")
		fmt.Fprintf(&buf, "	%q\n", "github.com/hexon/fastmsgpack/internal")
		fmt.Fprintf(&buf, ")\n")
		generate(&buf, "any", "decodeValue")
		generate(&buf, "type", "DecodeType")

		formatted, err := format.Source(buf.Bytes())
		if err != nil {
			formatted = buf.Bytes()
		}
		if err := os.WriteFile(*publicOut, formatted, 0644); err != nil {
			log.Fatal(err)
		}
	}

	{
		var buf bytes.Buffer
		fmt.Fprintf(&buf, "// Code generated by internal/codegen. DO NOT EDIT.\n")
		fmt.Fprintf(&buf, "\n")
		fmt.Fprintf(&buf, "package msgpackconverter\n")
		fmt.Fprintf(&buf, "\n")
		fmt.Fprintf(&buf, "import (\n")
		fmt.Fprintf(&buf, "	%q\n", "encoding/binary")
		fmt.Fprintf(&buf, "	%q\n", "errors")
		fmt.Fprintf(&buf, "	%q\n", "math")
		fmt.Fprintf(&buf, "\n")
		fmt.Fprintf(&buf, "	%q\n", "github.com/hexon/fastmsgpack/internal")
		fmt.Fprintf(&buf, ")\n")
		generate(&buf, "json", "convertValue")

		formatted, err := format.Source(buf.Bytes())
		if err != nil {
			formatted = buf.Bytes()
		}
		if err := os.WriteFile(*jsonOut, formatted, 0644); err != nil {
			log.Fatal(err)
		}
	}
}

func generate(w *bytes.Buffer, retType, name string) {
	fmt.Fprintf(w, "\n")
	var typeRestricted bool
	switch retType {
	case "void":
		fmt.Fprintf(w, "func %s(data []byte) (int, error) {\n", name)
	case "json":
		fmt.Fprintf(w, "func (c *converter) %s(data []byte) (int, error) {\n", name)
	case "any":
		fmt.Fprintf(w, "func %s(data []byte, opt internal.DecodeOptions) (any, int, error) {\n", name)
	case "_desc":
		fmt.Fprintf(w, "func %s(data []byte) string {\n", name)
	case "type":
		fmt.Fprintf(w, "func %s(data []byte) ValueType {\n", name)
	case "time":
		fmt.Fprintf(w, "func %s(data []byte, opt internal.DecodeOptions) (time.Time, int, error) {\n", name)
		typeRestricted = true
	default:
		fmt.Fprintf(w, "func %s(data []byte, opt internal.DecodeOptions) (%s, int, error) {\n", name, retType)
		typeRestricted = true
	}
	guaranteedLength := 1
	switch retType {
	case "time":
		guaranteedLength = 6
		emitLengthCheck(w, retType, MsgpackType{}, fmt.Sprint(guaranteedLength), "internal.ErrShortInputForTime")
	default:
		emitLengthCheck(w, retType, MsgpackType{}, fmt.Sprint(guaranteedLength), "internal.ErrShortInput")
	}
	if retType == "string" {
		fmt.Fprintf(w, "	if data[0] & 0b11100000 == 0b10100000 {\n")
		generateDecodeType(w, retType, name, guaranteedLength, fixstr)
		fmt.Fprintf(w, "	}\n")
	} else if retType == "int" {
		fmt.Fprintf(w, "	if data[0] <= 0x7f {\n")
		generateDecodeType(w, retType, name, guaranteedLength, positiveFixint)
		fmt.Fprintf(w, "	}\n")
	} else if !typeRestricted {
		fmt.Fprintf(w, "	if data[0] < 0xc0 {\n")
		fmt.Fprintf(w, "		if data[0] <= 0x7f {\n")
		generateDecodeType(w, retType, name, guaranteedLength, positiveFixint)
		fmt.Fprintf(w, "		}\n")
		fmt.Fprintf(w, "		if data[0] <= 0x8f {\n")
		generateDecodeType(w, retType, name, guaranteedLength, fixmap)
		fmt.Fprintf(w, "		}\n")
		fmt.Fprintf(w, "		if data[0] <= 0x9f {\n")
		generateDecodeType(w, retType, name, guaranteedLength, fixarray)
		fmt.Fprintf(w, "		}\n")
		generateDecodeType(w, retType, name, guaranteedLength, fixstr)
		fmt.Fprintf(w, "	}\n")
	}
	if !typeRestricted || retType == "int" {
		fmt.Fprintf(w, "	if data[0] >= 0xe0 {\n")
		generateDecodeType(w, retType, name, guaranteedLength, negativeFixint)
		fmt.Fprintf(w, "	}\n")
	}
	if typeRestricted && retType != "time" {
		fmt.Fprintf(w, "	switch data[0] {\n")
		for _, t := range types {
			if t.ByteEnd != 0 || (t.DataType != retType && !(isNumericType(t.DataType) && isNumericType(retType)) && !(retType == "string" && t.DataType == "[]byte")) {
				continue
			}
			fmt.Fprintf(w, "	case 0x%02x:\n", t.Byte)
			generateDecodeType(w, retType, name, guaranteedLength, t)
		}
		fmt.Fprintf(w, "	}\n")
		fmt.Fprintf(w, "\n")
		fmt.Fprintf(w, "	// Try extension decoding in case of a length-prefixed entry (#17)\n")
	}
	fmt.Fprintf(w, "	switch data[0] {\n")
	for _, t := range types {
		if t.ByteEnd != 0 {
			continue
		}
		if typeRestricted && t.DataType != "ext" {
			continue
		}
		fmt.Fprintf(w, "	case 0x%02x:\n", t.Byte)
		generateDecodeType(w, retType, name, guaranteedLength, t)
	}
	fmt.Fprintf(w, "	}\n")
	switch retType {
	case "void", "json":
		fmt.Fprintf(w, "	return 0, errors.New(%q)\n", "unexpected 0xc1")
	case "_desc":
		fmt.Fprintf(w, "	return %q\n", "0xc1")
	case "type":
		fmt.Fprintf(w, "	return TypeInvalid\n")
	default:
		fmt.Fprintf(w, "	return %s, 0, errors.New(%q + internal.DescribeValue(data) + %q)\n", produceZero(retType), "unexpected ", " when expecting "+retType)
	}
	fmt.Fprintf(w, "}\n")
}

func generateDecodeType(w *bytes.Buffer, retType, thisFunc string, guaranteedLength int, t MsgpackType) {
	if retType == "type" && t.DataType != "ext" {
		switch t.DataType {
		case "[]byte":
			fmt.Fprintf(w, "		return TypeBinary\n")
			return
		default:
			fmt.Fprintf(w, "		return Type%s\n", ucfirst(t.DataType))
			return
		case "ext":
			// continue
		}
	}
	minLen, preamble, val, lencalc := getDecoder(t)
	if minLen > guaranteedLength {
		emitLengthCheck(w, retType, t, fmt.Sprint(minLen), "internal.ErrShortInput")
	}
	if preamble != "" {
		fmt.Fprintf(w, "		%s\n", preamble)
	}
	if fmt.Sprint(minLen) != lencalc {
		emitLengthCheck(w, retType, t, lencalc, "internal.ErrShortInput")
	}
	switch retType {
	case "_desc":
		switch t.DataType {
		case "array":
			fmt.Fprintf(w, "		return fmt.Sprintf(%q, %s)\n", t.Name+" (%d entries)", val)
		case "map":
			fmt.Fprintf(w, "		return fmt.Sprintf(%q, %s)\n", t.Name+" (%d entries)", val)
		case "ext":
			fmt.Fprintf(w, "		return fmt.Sprintf(%q, int8(data[%d]), len(%s))\n", t.Name+" (type %d, %d bytes)", t.ExtTypeAt, val)
		case "int":
			fmt.Fprintf(w, "		return fmt.Sprintf(%q, %s)\n", t.Name+" (%d)", val)
		case "float32", "float64":
			fmt.Fprintf(w, "		return fmt.Sprintf(%q, %s)\n", t.Name+" (%f)", val)
		case "string":
			fmt.Fprintf(w, "		return fmt.Sprintf(%q, %s)\n", t.Name+" (%q)", val)
		default:
			fmt.Fprintf(w, "		return %q\n", t.Name)
		}
		return
	case "json":
		switch t.DataType {
		case "array":
			fmt.Fprintf(w, "		return c.%s_array(data, %s, %s)\n", lcfirst(thisFunc), lencalc, val)
		case "map":
			fmt.Fprintf(w, "		return c.%s_map(data, %s, %s)\n", lcfirst(thisFunc), lencalc, val)
		case "ext":
			fmt.Fprintf(w, "		return %s, c.%s_ext(%s, int8(data[%d]))\n", lencalc, lcfirst(thisFunc), val, t.ExtTypeAt)
		case "[]byte":
			fmt.Fprintf(w, "		return %s, c.appendBytes(%s)\n", lencalc, val)
		case "string":
			fmt.Fprintf(w, "		return %s, c.appendBytes(%s)\n", lencalc, regexp.MustCompile(`^internal\.UnsafeStringCast\((.+)\)$`).ReplaceAllString(val, "$1"))
		case "nil":
			fmt.Fprintf(w, "		return %s, c.appendRaw(%q)\n", lencalc, "null")
		case "bool":
			fmt.Fprintf(w, "		return %s, c.appendRaw(%q)\n", lencalc, val)
		case "float64":
			fmt.Fprintf(w, "		return %s, c.appendFloat(%s)\n", lencalc, val)
		case "float32":
			fmt.Fprintf(w, "		return %s, c.appendFloat(float64(%s))\n", lencalc, val)
		default:
			fmt.Fprintf(w, "		return %s, c.append%s(%s)\n", lencalc, ucfirst(t.DataType), val)
		}
		return
	case "void":
		switch t.DataType {
		case "array":
			fmt.Fprintf(w, "		return internal.SkipMultiple(data, %s, %s)\n", lencalc, val)
		case "map":
			fmt.Fprintf(w, "		return internal.SkipMultiple(data, %s, 2*(%s))\n", lencalc, val)
		default:
			fmt.Fprintf(w, "		return %s, nil\n", lencalc)
		}
		return
	case "type":
		fmt.Fprintf(w, "		return %s_ext(%s, int8(data[%d]))\n", lcfirst(thisFunc), val, t.ExtTypeAt)
		return
	}
	switch t.DataType {
	case "array":
		switch retType {
		case "any":
			fmt.Fprintf(w, "			return %s_array(data, %s, %s, opt)\n", lcfirst(thisFunc), lencalc, val)
		default:
			fmt.Fprintf(w, "			return %s, 0, errors.New(%q)\n", produceZero(retType), "unexpected array when expecting "+retType)
		}
	case "map":
		switch retType {
		case "any":
			fmt.Fprintf(w, "			return %s_map(data, %s, %s, opt)\n", lcfirst(thisFunc), lencalc, val)
		default:
			fmt.Fprintf(w, "			return %s, 0, errors.New(%q)\n", produceZero(retType), "unexpected map when expecting "+retType)
		}
	case "ext":
		fmt.Fprintf(w, "		ret, err := %s_ext(%s, int8(data[%d]), opt)\n", lcfirst(thisFunc), val, t.ExtTypeAt)
		fmt.Fprintf(w, "		return ret, %s, err\n", lencalc)
	default:
		if retType != t.DataType && isNumericType(retType) && isNumericType(t.DataType) {
			fmt.Fprintf(w, "		return %s(%s), %s, nil\n", retType, val, lencalc)
			break
		}
		if retType == "string" && t.DataType == "[]byte" {
			fmt.Fprintf(w, "		return internal.UnsafeStringCast(%s), %s, nil\n", val, lencalc)
			break
		}
		switch retType {
		case "any", t.DataType:
			fmt.Fprintf(w, "		return %s, %s, nil\n", val, lencalc)
		default:
			fmt.Fprintf(w, "			return %s, 0, errors.New(%q)\n", produceZero(retType), "unexpected "+t.DataType+" when expecting "+retType)
		}
	}
}

func emitLengthCheck(w *bytes.Buffer, retType string, t MsgpackType, minLen string, errName string) {
	fmt.Fprintf(w, "		if len(data) < %s {\n", minLen)
	switch retType {
	case "void", "json":
		fmt.Fprintf(w, "			return 0, %s\n", errName)
	case "_desc":
		if minLen == "1" {
			fmt.Fprintf(w, "			return %q\n", "empty input")
		} else {
			fmt.Fprintf(w, "			return %q\n", "truncated "+t.Name)
		}
	case "type":
		fmt.Fprintf(w, "			return TypeInvalid\n")
	default:
		fmt.Fprintf(w, "			return %s, 0, %s\n", produceZero(retType), errName)
	}
	fmt.Fprintf(w, "		}\n")
}

func getDecoder(t MsgpackType) (int, string, string, string) {
	var extraLen int
	if t.ExtTypeAt > 0 {
		extraLen = 1
	}
	minLen := max(1, t.ExtTypeAt+extraLen, t.DataStart+t.DataLen, t.DynamicLengthStart+t.DynamicLengthLen+extraLen)
	var preamble, sel, lencalc string
	if t.DataLen == 1 && strings.Contains(t.DataCast, "$[0]") {
		return minLen, "", strings.ReplaceAll(t.DataCast, "$[0]", fmt.Sprintf("data[%d]", t.DataStart)), fmt.Sprint(1 + t.DataStart + extraLen)
	} else if t.DataLen > 0 {
		sel = fmt.Sprintf("data[%d:%d]", t.DataStart, t.DataStart+t.DataLen)
		lencalc = fmt.Sprint(t.DataStart + t.DataLen)
	} else if t.DynamicLengthLen > 0 {
		if t.DynamicLengthLen == 1 {
			preamble = fmt.Sprintf("s := int(data[%d]) + %d", t.DynamicLengthStart, t.DynamicLengthStart+t.DynamicLengthLen+extraLen)
		} else {
			preamble = fmt.Sprintf("s := int(binary.BigEndian.Uint%d(data[%d:%d])) + %d", 8*t.DynamicLengthLen, t.DynamicLengthStart, t.DynamicLengthStart+t.DynamicLengthLen, t.DynamicLengthStart+t.DynamicLengthLen+extraLen)
		}
		sel = fmt.Sprintf("data[%d:s]", t.DynamicLengthStart+t.DynamicLengthLen+extraLen)
		lencalc = "s"
	} else if t.Name == "fixstr" {
		preamble = "s := int(data[0]&0b00011111) + 1"
		sel = "data[1:s]"
		lencalc = "s"
	} else {
		return minLen, "", t.DataCast, fmt.Sprint(1 + extraLen)
	}
	return minLen, preamble, strings.ReplaceAll(t.DataCast, "$", sel), lencalc
}

func produceZero(retType string) string {
	switch retType {
	case "string":
		return `""`
	case "int", "float32", "float64":
		return "0"
	case "time":
		return "time.Time{}"
	case "bool":
		return "false"
	default:
		return "nil"
	}
}

func lcfirst(s string) string {
	return strings.ToLower(s[:1]) + s[1:]
}

func ucfirst(s string) string {
	return strings.ToUpper(s[:1]) + s[1:]
}

func isNumericType(t string) bool {
	switch t {
	case "int", "float32", "float64":
		return true
	default:
		return false
	}
}
