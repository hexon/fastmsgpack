package msgpackconverter

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"strconv"
	"time"
	"unicode/utf8"

	"github.com/hexon/fastmsgpack/internal"
)

const hex = "0123456789abcdef"

var jsonSafeSet = [utf8.RuneSelf]bool{
	' ':      true,
	'!':      true,
	'"':      false,
	'#':      true,
	'$':      true,
	'%':      true,
	'&':      true,
	'\'':     true,
	'(':      true,
	')':      true,
	'*':      true,
	'+':      true,
	',':      true,
	'-':      true,
	'.':      true,
	'/':      true,
	'0':      true,
	'1':      true,
	'2':      true,
	'3':      true,
	'4':      true,
	'5':      true,
	'6':      true,
	'7':      true,
	'8':      true,
	'9':      true,
	':':      true,
	';':      true,
	'<':      true,
	'=':      true,
	'>':      true,
	'?':      true,
	'@':      true,
	'A':      true,
	'B':      true,
	'C':      true,
	'D':      true,
	'E':      true,
	'F':      true,
	'G':      true,
	'H':      true,
	'I':      true,
	'J':      true,
	'K':      true,
	'L':      true,
	'M':      true,
	'N':      true,
	'O':      true,
	'P':      true,
	'Q':      true,
	'R':      true,
	'S':      true,
	'T':      true,
	'U':      true,
	'V':      true,
	'W':      true,
	'X':      true,
	'Y':      true,
	'Z':      true,
	'[':      true,
	'\\':     false,
	']':      true,
	'^':      true,
	'_':      true,
	'`':      true,
	'a':      true,
	'b':      true,
	'c':      true,
	'd':      true,
	'e':      true,
	'f':      true,
	'g':      true,
	'h':      true,
	'i':      true,
	'j':      true,
	'k':      true,
	'l':      true,
	'm':      true,
	'n':      true,
	'o':      true,
	'p':      true,
	'q':      true,
	'r':      true,
	's':      true,
	't':      true,
	'u':      true,
	'v':      true,
	'w':      true,
	'x':      true,
	'y':      true,
	'z':      true,
	'{':      true,
	'|':      true,
	'}':      true,
	'~':      true,
	'\u007f': true,
}

type JSONConverter struct {
	encodedDict [][]byte
}

func NewJSONConverter(dict []string) JSONConverter {
	encodedDict := make([][]byte, len(dict))
	for i, s := range dict {
		encodedDict[i] = encodeJSONString(nil, []byte(s))
	}
	return JSONConverter{encodedDict}
}

type converter struct {
	data   []byte
	offset int
	w      *bufio.Writer
	JSONConverter
}

// Convert the given msgpack to JSON efficiently.
func (c JSONConverter) Convert(dst io.Writer, data []byte) error {
	cc := converter{
		data:          data,
		w:             bufio.NewWriter(dst),
		JSONConverter: c,
	}
	if err := cc.convertValue(); err != nil {
		return err
	}
	return cc.w.Flush()
}

func (c *converter) convertValue() error {
	buf := c.w.AvailableBuffer()
	b := c.data[c.offset]
	c.offset++
	switch b & 0b11100000 {
	case 0b00000000, 0b00100000, 0b01000000, 0b01100000:
		buf = strconv.AppendInt(buf, int64(b), 10)
	case 0b11100000:
		buf = strconv.AppendInt(buf, int64(int8(b)), 10)
	case 0b10100000:
		l := int(b & 0b00011111)
		c.offset += l
		buf = encodeJSONString(buf, c.data[c.offset-l:c.offset])
	case 0b10000000:
		if b&0b11110000 == 0b10010000 {
			return c.convertArray(int(b & 0b00001111))
		} else {
			return c.convertMap(int(b & 0b00001111))
		}

	default:
		switch b {
		case 0xc0:
			buf = append(buf, "null"...)
		case 0xc2:
			buf = append(buf, "false"...)
		case 0xc3:
			buf = append(buf, "true"...)
		case 0xcc:
			buf = strconv.AppendInt(buf, int64(c.readUint8()), 10)
		case 0xcd:
			buf = strconv.AppendInt(buf, int64(c.readUint16()), 10)
		case 0xce:
			buf = strconv.AppendInt(buf, int64(c.readUint32()), 10)
		case 0xcf:
			buf = strconv.AppendInt(buf, int64(c.readUint64()), 10)
		case 0xd0:
			buf = strconv.AppendInt(buf, int64(int8(c.readUint8())), 10)
		case 0xd1:
			buf = strconv.AppendInt(buf, int64(int16(c.readUint16())), 10)
		case 0xd2:
			buf = strconv.AppendInt(buf, int64(int32(c.readUint32())), 10)
		case 0xd3:
			buf = strconv.AppendInt(buf, int64(c.readUint64()), 10)
		case 0xca:
			buf = strconv.AppendFloat(buf, float64(math.Float32frombits(c.readUint32())), 'f', -1, 32)
		case 0xcb:
			buf = strconv.AppendFloat(buf, math.Float64frombits(c.readUint64()), 'f', -1, 64)
		case 0xd9, 0xc4:
			l := int(c.readUint8())
			buf = encodeJSONString(buf, c.readBytes(l))
		case 0xda, 0xc5:
			l := int(c.readUint16())
			buf = encodeJSONString(buf, c.readBytes(l))
		case 0xdb, 0xc6:
			l := int(c.readUint32())
			buf = encodeJSONString(buf, c.readBytes(l))
		case 0xdc:
			return c.convertArray(int(c.readUint16()))
		case 0xdd:
			return c.convertArray(int(c.readUint32()))
		case 0xde:
			return c.convertMap(int(c.readUint16()))
		case 0xdf:
			return c.convertMap(int(c.readUint32()))
		case 0xd4:
			c.offset += 2
			return c.convertExtension(c.data[c.offset-2], c.data[c.offset-1:c.offset])
		case 0xd5:
			c.offset += 3
			return c.convertExtension(c.data[c.offset-3], c.data[c.offset-2:c.offset])
		case 0xd6:
			c.offset += 5
			return c.convertExtension(c.data[c.offset-5], c.data[c.offset-4:c.offset])
		case 0xd7:
			c.offset += 9
			return c.convertExtension(c.data[c.offset-9], c.data[c.offset-8:c.offset])
		case 0xd8:
			c.offset += 17
			return c.convertExtension(c.data[c.offset-17], c.data[c.offset-16:c.offset])
		case 0xc7:
			l := int(c.readUint8())
			c.offset += 1 + l
			return c.convertExtension(c.data[c.offset-l-1], c.data[c.offset-l:c.offset])
		case 0xc8:
			l := int(c.readUint16())
			c.offset += 1 + l
			return c.convertExtension(c.data[c.offset-l-1], c.data[c.offset-l:c.offset])
		case 0xc9:
			l := int(c.readUint32())
			c.offset += 1 + l
			return c.convertExtension(c.data[c.offset-l-1], c.data[c.offset-l:c.offset])
		default:
			c.offset--
			return fmt.Errorf("unexpected msgpack byte while decoding: %c", b)
		}
	}
	_, err := c.w.Write(buf)
	return err
}

func (c *converter) convertArray(elements int) error {
	c.w.WriteByte('[')
	for i := 0; elements > i; i++ {
		if i > 0 {
			c.w.WriteByte(',')
		}
		if err := c.convertValue(); err != nil {
			return err
		}
	}
	c.w.WriteByte(']')
	return nil
}

func (c *converter) convertMap(elements int) error {
	c.w.WriteByte('{')
	for i := 0; elements > i; i++ {
		if i > 0 {
			c.w.WriteByte(',')
		}
		if err := c.convertValue(); err != nil {
			return err
		}
		c.w.WriteByte(':')
		if err := c.convertValue(); err != nil {
			return err
		}
	}
	return c.w.WriteByte('}')
}

func (c *converter) readUint8() uint8 {
	c.offset++
	return uint8(c.data[c.offset-1])
}

func (c *converter) readUint16() uint16 {
	c.offset += 2
	return binary.BigEndian.Uint16(c.data[c.offset-2:])
}

func (c *converter) readUint32() uint32 {
	c.offset += 4
	return binary.BigEndian.Uint32(c.data[c.offset-4:])
}

func (c *converter) readUint64() uint64 {
	c.offset += 8
	return binary.BigEndian.Uint64(c.data[c.offset-8:])
}

func (c *converter) readBytes(n int) []byte {
	c.offset += n
	return c.data[c.offset-n : c.offset]
}

func (c *converter) convertExtension(extType uint8, data []byte) error {
	switch int8(extType) {
	case -1:
		var ts time.Time
		switch len(data) {
		case 4:
			ts = time.Unix(int64(binary.BigEndian.Uint32(data)), 0)
		case 8:
			n := binary.BigEndian.Uint64(data)
			ts = time.Unix(int64(n&0x00000003ffffffff), int64(n>>34))
		case 12:
			nsec := binary.BigEndian.Uint32(data[:4])
			sec := binary.BigEndian.Uint64(data[4:])
			ts = time.Unix(int64(sec), int64(nsec))
		default:
			return fmt.Errorf("failed to timestamp of %d bytes", len(data))
		}
		buf := c.w.AvailableBuffer()
		buf = append(buf, '"')
		if ts.Nanosecond() == 0 {
			buf = ts.AppendFormat(buf, time.RFC3339)
		} else {
			buf = ts.AppendFormat(buf, time.RFC3339Nano)
		}
		buf = append(buf, '"')
		_, err := c.w.Write(buf)
		return err

	case int8(math.MinInt8): // Interned string
		n, ok := internal.DecodeBytesToUint(data)
		if !ok {
			return errors.New("failed to decode index number of interned string")
		}
		if n >= uint(len(c.encodedDict)) {
			return fmt.Errorf("interned string %d is out of bounds for the dict (%d entries)", n, len(c.encodedDict))
		}
		_, err := c.w.Write(c.encodedDict[n])
		return err

	case 17: // Length-prefixed entry
		c.offset -= len(data)
		return c.convertValue()

	default:
		return fmt.Errorf("don't know how to encode Extension type %d", int8(extType))
	}
}

func encodeJSONString(dst, s []byte) []byte {
	if len(dst) <= len(s)+2 {
		dst = make([]byte, 0, len(s)*10/8)
	}
	dst = append(dst, '"')
	for i := 0; len(s) > i; {
		if b := s[i]; b < utf8.RuneSelf {
			if jsonSafeSet[b] {
				dst = append(dst, b)
				i++
				continue
			}
			switch b {
			case '\\', '"':
				dst = append(dst, '\\', b)
			case '\n':
				dst = append(dst, '\\', 'n')
			case '\r':
				dst = append(dst, '\\', 'r')
			case '\t':
				dst = append(dst, '\\', 't')
			default:
				dst = append(dst, '\\', 'u', '0', '0', hex[b>>4], hex[b&0xf])
			}
			i++
			continue
		}
		r, n := utf8.DecodeRune(s[i:])
		switch r {
		case utf8.RuneError:
			dst = append(dst, `\ufffd`...)
		case '\u2028':
			dst = append(dst, `\u2028`...)
		case '\u2029':
			dst = append(dst, `\u2029`...)
		default:
			dst = append(dst, s[i:i+n]...)
		}
		i += n
	}
	dst = append(dst, '"')
	return dst
}
