package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/format"
	"log"
	"os"
	"strings"
)

var (
	internalOut = flag.String("internal-out", "../../internal/generated.go", "")
	publicOut   = flag.String("public-out", "../../generated.go", "")
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
		fmt.Fprintf(&buf, "	%q\n", "math")
		fmt.Fprintf(&buf, "	%q\n", "time")
		fmt.Fprintf(&buf, ")\n")
		generate(&buf, "void", "ValueLength")
		generate(&buf, "_desc", "DescribeValue")
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
		generate(&buf, "string", "decodeString")

		formatted, err := format.Source(buf.Bytes())
		if err != nil {
			formatted = buf.Bytes()
		}
		if err := os.WriteFile(*publicOut, formatted, 0644); err != nil {
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
	case "any":
		fmt.Fprintf(w, "func (d *Decoder) %s(data []byte) (any, int, error) {\n", name)
	case "_desc":
		fmt.Fprintf(w, "func %s(data []byte) string {\n", name)
	case "time":
		fmt.Fprintf(w, "func %s(data []byte) (time.Time, int, error) {\n", name)
		typeRestricted = true
	case "string":
		fmt.Fprintf(w, "func (d *Decoder) %s(data []byte) (%s, int, error) {\n", name, retType)
		typeRestricted = true
	default:
		fmt.Fprintf(w, "func %s(data []byte) (%s, int, error) {\n", name, retType)
		typeRestricted = true
	}
	guaranteedLength := 1
	switch retType {
	case "float32", "float64":
		guaranteedLength = 5
		emitLengthCheck(w, retType, fmt.Sprint(guaranteedLength), "internal.ErrShortInputForFloat")
	case "time":
		guaranteedLength = 6
		emitLengthCheck(w, retType, fmt.Sprint(guaranteedLength), "internal.ErrShortInputForTime")
	default:
		emitLengthCheck(w, retType, fmt.Sprint(guaranteedLength), "internal.ErrShortInput")
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
			if t.ByteEnd != 0 || (t.DataType != retType && !(strings.HasPrefix(t.DataType, "float") && strings.HasPrefix(retType, "float"))) {
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
	case "void":
		fmt.Fprintf(w, "	return 0, errors.New(%q)\n", "unexpected 0xc1")
	case "_desc":
		fmt.Fprintf(w, "	return %q\n", "0xc1")
	default:
		fmt.Fprintf(w, "	return %s, 0, errors.New(%q + internal.DescribeValue(data) + %q)\n", produceZero(retType), "unexpected ", " when expecting "+retType)
	}
	fmt.Fprintf(w, "}\n")
}

func generateDecodeType(w *bytes.Buffer, retType, thisFunc string, guaranteedLength int, t MsgpackType) {
	if retType == "_desc" {
		fmt.Fprintf(w, "		return %q\n", t.Name)
		return
	}
	minLen, preamble, val, lencalc := getDecoder(t)
	if minLen > guaranteedLength && (preamble != "" || retType != "void") {
		emitLengthCheck(w, retType, fmt.Sprint(minLen), "internal.ErrShortInput")
	}
	if preamble != "" {
		fmt.Fprintf(w, "		%s\n", preamble)
	}
	if fmt.Sprint(minLen) != lencalc && retType != "void" {
		emitLengthCheck(w, retType, lencalc, "internal.ErrShortInput")
	}
	switch t.DataType {
	case "array":
		switch retType {
		case "void":
			fmt.Fprintf(w, "			return internal.SkipMultiple(data, %s, %s)\n", lencalc, val)
		case "any":
			fmt.Fprintf(w, "			return d.%s_array(data, %s, %s)\n", lcfirst(thisFunc), lencalc, val)
		default:
			fmt.Fprintf(w, "			return %s, 0, errors.New(%q)\n", produceZero(retType), "unexpected array when expecting "+retType)
		}
	case "map":
		switch retType {
		case "void":
			fmt.Fprintf(w, "			return internal.SkipMultiple(data, %s, 2*(%s))\n", lencalc, val)
		case "any":
			fmt.Fprintf(w, "			return d.%s_map(data, %s, %s)\n", lcfirst(thisFunc), lencalc, val)
		default:
			fmt.Fprintf(w, "			return %s, 0, errors.New(%q)\n", produceZero(retType), "unexpected map when expecting "+retType)
		}
	case "ext":
		switch retType {
		case "void":
			fmt.Fprintf(w, "		return %s, nil\n", lencalc)
		case "any", "string":
			fmt.Fprintf(w, "		ret, err := d.%s_ext(%s, int8(data[%d]))\n", lcfirst(thisFunc), val, t.ExtTypeAt)
			fmt.Fprintf(w, "		return ret, %s, err\n", lencalc)
		default:
			fmt.Fprintf(w, "		ret, err := %s_ext(%s, int8(data[%d]))\n", lcfirst(thisFunc), val, t.ExtTypeAt)
			fmt.Fprintf(w, "		return ret, %s, err\n", lencalc)
		}
	default:
		if retType == "float64" && t.DataType == "float32" {
			fmt.Fprintf(w, "		return float64(%s), %s, nil\n", val, lencalc)
			break
		}
		if retType == "float32" && t.DataType == "float64" {
			fmt.Fprintf(w, "		return float32(%s), %s, nil\n", val, lencalc)
			break
		}
		switch retType {
		case "void":
			fmt.Fprintf(w, "		return %s, nil\n", lencalc)
		case "any", t.DataType:
			fmt.Fprintf(w, "		return %s, %s, nil\n", val, lencalc)
		default:
			fmt.Fprintf(w, "			return %s, 0, errors.New(%q)\n", produceZero(retType), "unexpected "+t.DataType+" when expecting "+retType)
		}
	}
}

func emitLengthCheck(w *bytes.Buffer, retType, minLen string, errName string) {
	fmt.Fprintf(w, "		if len(data) < %s {\n", minLen)
	switch retType {
	case "void":
		fmt.Fprintf(w, "			return 0, %s\n", errName)
	case "_desc":
		fmt.Fprintf(w, "			return %q\n", "empty input")
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
