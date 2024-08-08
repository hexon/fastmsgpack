package msgpackconverter

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"math"
	"strconv"
	"time"
	"unicode/utf8"

	"github.com/hexon/fastmsgpack"
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

func NewJSONConverter(dict *fastmsgpack.Dict) JSONConverter {
	encodedDict := make([][]byte, len(dict.Strings))
	for i, s := range dict.Strings {
		encodedDict[i] = encodeJSONString(nil, []byte(s))
	}
	return JSONConverter{encodedDict}
}

type converter struct {
	w *bufio.Writer
	JSONConverter
}

// Convert the given msgpack to JSON efficiently.
func (c JSONConverter) Convert(dst io.Writer, data []byte) error {
	cc := converter{
		w:             bufio.NewWriter(dst),
		JSONConverter: c,
	}
	if _, err := cc.convertValue(data); err != nil {
		return err
	}
	return cc.w.Flush()
}

func (c *converter) convertValue_array(data []byte, offset, elements int) (int, error) {
	if err := c.w.WriteByte('['); err != nil {
		return 0, err
	}
	for i := 0; elements > i; i++ {
		if i > 0 {
			c.w.WriteByte(',')
		}
		c, err := c.convertValue(data[offset:])
		if err != nil {
			return 0, err
		}
		offset += c
	}
	return offset, c.w.WriteByte(']')
}

func (c *converter) convertValue_map(data []byte, offset, elements int) (int, error) {
	if err := c.w.WriteByte('{'); err != nil {
		return 0, err
	}
	for i := 0; elements > i; i++ {
		if i > 0 {
			c.w.WriteByte(',')
		}
		n, err := c.convertValue(data[offset:])
		if err != nil {
			return 0, err
		}
		offset += n
		c.w.WriteByte(':')
		n, err = c.convertValue(data[offset:])
		if err != nil {
			return 0, err
		}
		offset += n
	}
	return offset, c.w.WriteByte('}')
}

func (c *converter) appendRaw(s string) error {
	_, err := c.w.WriteString(s)
	return err
}

func (c *converter) appendBytes(b []byte) error {
	_, err := c.w.Write(encodeJSONString(c.w.AvailableBuffer(), b))
	return err
}

func (c *converter) appendInt(i int) error {
	_, err := c.w.Write(strconv.AppendInt(c.w.AvailableBuffer(), int64(i), 10))
	return err
}

func (c *converter) appendFloat(f float64) error {
	_, err := c.w.Write(strconv.AppendFloat(c.w.AvailableBuffer(), f, 'f', -1, 32))
	return err
}

func (c *converter) convertValue_ext(data []byte, extType int8) error {
	switch extType {
	case -1:
		ts, err := internal.DecodeTimestamp(data)
		if err != nil {
			return err
		}
		buf := c.w.AvailableBuffer()
		buf = append(buf, '"')
		if ts.Nanosecond() == 0 {
			buf = ts.AppendFormat(buf, time.RFC3339)
		} else {
			buf = ts.AppendFormat(buf, time.RFC3339Nano)
		}
		buf = append(buf, '"')
		_, err = c.w.Write(buf)
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
		_, err := c.convertValue(data)
		return err

	default:
		return errors.New("don't know how to encode Extension type " + strconv.FormatInt(int64(extType), 10))
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
