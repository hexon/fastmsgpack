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

var (
	falseBytes = []byte("false")
	trueBytes  = []byte("true")
	nullBytes  = []byte("null")
)

type JSONConverter struct {
	options     internal.DecodeOptions
	encodedDict [][]byte
}

func NewJSONConverter(opts ...fastmsgpack.DecodeOption) JSONConverter {
	var ret JSONConverter
	for _, o := range opts {
		o(&ret.options)
	}
	if ret.options.Dict != nil {
		ret.encodedDict = make([][]byte, len(ret.options.Dict.Strings))
		for i, s := range ret.options.Dict.Strings {
			ret.encodedDict[i] = encodeJSONString(nil, []byte(s))
		}
	}
	return ret
}

// WithHideNulls drops all NULL values from the converted result. Array and map entries with value nil won't be silently skipped over.
func WithHideNulls() fastmsgpack.DecodeOption {
	return func(opt *internal.DecodeOptions) {
		opt.JSON_HideNulls = true
	}
}

type converter struct {
	w *bufio.Writer
	JSONConverter
	uncommitted        []byte
	transactionalState transactionalState
}

type transactionalState uint8

const (
	// transactionalStateNormal means we just write everything straight into the bufio.Writer.
	transactionalStateNormal transactionalState = iota

	// transactionalStateTentative means we buffer all writes into c.uncommitted
	transactionalStateTentative

	// transactionalStateUndecided means that if the next write is handleNil(), we discard c.uncommitted and go to transactionalStateRolledBack. Any other write will write c.uncommitted out and go to transactionalStateNormal.
	transactionalStateUndecided

	// transactionalStateRolledBack means we just discarded c.uncommitted. The state should be changed before the next write.
	transactionalStateRolledBack
)

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
	if err := c.writeByte('['); err != nil {
		return 0, err
	}
	addComma := false
	for i := 0; elements > i; i++ {
		if addComma {
			if c.options.JSON_HideNulls {
				c.transactionalState = transactionalStateTentative
			}
			if err := c.writeByte(','); err != nil {
				return 0, err
			}
		}
		if c.options.JSON_HideNulls {
			c.transactionalState = transactionalStateUndecided
		}
		n, err := c.convertValue(data[offset:])
		if err != nil {
			return 0, err
		}
		offset += n
		if c.transactionalState != transactionalStateRolledBack {
			addComma = true
		}
		c.transactionalState = transactionalStateNormal
	}
	return offset, c.writeByte(']')
}

func (c *converter) convertValue_map(data []byte, offset, elements int) (int, error) {
	if err := c.writeByte('{'); err != nil {
		return 0, err
	}
	addComma := false
	for i := 0; elements > i; i++ {
		if c.options.JSON_HideNulls {
			c.transactionalState = transactionalStateTentative
		}
		if addComma {
			if err := c.writeByte(','); err != nil {
				return 0, err
			}
		}
		n, err := c.convertValue(data[offset:])
		if err != nil {
			return 0, err
		}
		offset += n
		if err := c.writeByte(':'); err != nil {
			return 0, err
		}
		if c.options.JSON_HideNulls {
			c.transactionalState = transactionalStateUndecided
		}
		n, err = c.convertValue(data[offset:])
		if err != nil {
			return 0, err
		}
		offset += n
		if c.transactionalState != transactionalStateRolledBack {
			addComma = true
		}
		c.transactionalState = transactionalStateNormal
	}
	return offset, c.writeByte('}')
}

func (c *converter) write(b []byte) error {
	switch c.transactionalState {
	case transactionalStateNormal:
	case transactionalStateTentative:
		c.uncommitted = append(c.uncommitted, b...)
		return nil
	case transactionalStateUndecided:
		if _, err := c.w.Write(c.uncommitted); err != nil {
			return err
		}
		c.uncommitted = c.uncommitted[:0]
		c.transactionalState = transactionalStateNormal
	}
	_, err := c.w.Write(b)
	return err
}

func (c *converter) writeByte(b byte) error {
	switch c.transactionalState {
	case transactionalStateNormal:
	case transactionalStateTentative:
		c.uncommitted = append(c.uncommitted, b)
		return nil
	case transactionalStateUndecided:
		if _, err := c.w.Write(c.uncommitted); err != nil {
			return err
		}
		c.uncommitted = c.uncommitted[:0]
		c.transactionalState = transactionalStateNormal
	}
	return c.w.WriteByte(b)
}

func (c *converter) availableBuffer() []byte {
	return c.uncommitted[len(c.uncommitted):]
}

func (c *converter) handleNil() error {
	if c.transactionalState == transactionalStateUndecided {
		c.transactionalState = transactionalStateRolledBack
		c.uncommitted = c.uncommitted[:0]
		return nil
	}
	return c.write(nullBytes)
}

func (c *converter) appendBytes(b []byte) error {
	return c.write(encodeJSONString(c.availableBuffer(), b))
}

func (c *converter) appendInt(i int) error {
	return c.write(strconv.AppendInt(c.availableBuffer(), int64(i), 10))
}

func (c *converter) appendFloat(f float64) error {
	return c.write(strconv.AppendFloat(c.availableBuffer(), f, 'f', -1, 32))
}

func (c *converter) convertValue_ext(data []byte, extType int8) error {
	switch extType {
	case -1:
		ts, err := internal.DecodeTimestamp(data)
		if err != nil {
			return err
		}
		buf := c.availableBuffer()
		buf = append(buf, '"')
		if ts.Nanosecond() == 0 {
			buf = ts.AppendFormat(buf, time.RFC3339)
		} else {
			buf = ts.AppendFormat(buf, time.RFC3339Nano)
		}
		buf = append(buf, '"')
		return c.write(buf)

	case int8(math.MinInt8): // Interned string
		n, ok := internal.DecodeBytesToUint(data)
		if !ok {
			return errors.New("failed to decode index number of interned string")
		}
		if n >= uint(len(c.encodedDict)) {
			return fmt.Errorf("interned string %d is out of bounds for the dict (%d entries)", n, len(c.encodedDict))
		}
		return c.write(c.encodedDict[n])

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
