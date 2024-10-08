package debug

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"slices"
	"strings"

	"github.com/hexon/fastmsgpack"
	"github.com/hexon/fastmsgpack/internal"
)

type printer struct {
	w       *bufio.Writer
	options internal.DecodeOptions
	indent  int
}

func Fdump(dst io.Writer, data []byte, opts ...fastmsgpack.DecodeOption) error {
	p := printer{
		w: bufio.NewWriter(dst),
	}
	for _, o := range opts {
		o(&p.options)
	}
	if _, err := p.debugValue(data); err != nil {
		_ = p.w.Flush()
		return err
	}
	return p.w.Flush()
}

func (p *printer) debugValue_array(data []byte, offset, elements int) (int, error) {
	p.printf("[%02x] array (%d elements)", data[:offset], elements)
	p.indent++
	for i := 0; elements > i; i++ {
		n, err := p.debugValue(data[offset:])
		if err != nil {
			return 0, err
		}
		offset += n
	}
	p.indent--
	p.printf("(end of array)")
	return offset, nil
}

func (p *printer) debugValue_map(data []byte, offset, elements int) (int, error) {
	p.printf("[%02x] map (%d elements)", data[:offset], elements)
	p.indent++
	for i := 0; elements > i; i++ {
		n, err := p.debugValue(data[offset:])
		if err != nil {
			return 0, err
		}
		offset += n
		p.indent++
		n, err = p.debugValue(data[offset:])
		if err != nil {
			return 0, err
		}
		offset += n
		p.indent--
	}
	p.indent--
	p.printf("(end of map)")
	return offset, nil
}

func (p *printer) printf(f string, args ...any) error {
	p.w.WriteString(strings.Repeat("\t", p.indent))
	_, err := fmt.Fprintf(p.w, f, args...)
	p.w.WriteByte('\n')
	return err
}

func (p *printer) appendBytes(header, b []byte) error {
	return p.printf("[%02x] bytes: %q", header, b)
}

func (p *printer) appendString(header []byte, s string) error {
	return p.printf("[%02x] string: %q", header, s)
}

func (p *printer) appendNil(header []byte, _ any) error {
	return p.printf("[%02x] nil", header)
}

func (p *printer) appendBool(header []byte, b bool) error {
	return p.printf("[%02x] bool %v", header, b)
}

func (p *printer) appendInt(header []byte, i int) error {
	return p.printf("[%02x] int: %d", header, i)
}

func (p *printer) appendFloat32(header []byte, f float32) error {
	return p.printf("[%02x] float32: %f", header, f)
}

func (p *printer) appendFloat64(header []byte, f float64) error {
	return p.printf("[%02x] float64: %f", header, f)
}

func (p *printer) debugValue_ext(header, data []byte, extType int8) error {
	switch extType {
	case -1:
		ts, err := internal.DecodeTimestamp(data)
		if err != nil {
			return err
		}
		return p.printf("[%02x %02x] time: %s", header, data, ts)

	case int8(math.MinInt8): // Interned string
		n, ok := internal.DecodeBytesToUint(data)
		if !ok {
			return errors.New("failed to decode index number of interned string")
		}
		s, err := p.options.Dict.LookupString(n)
		if err != nil {
			return p.printf("[%02x %02x] interned string %d: [broken] %v", header, data, n, err)
		}
		return p.printf("[%02x %02x] interned string %d: %s", header, data, n, s)

	case 17: // Length-prefixed entry
		p.printf("[%02x] length prefixed (%d bytes)", header, len(data))
		p.indent++
		_, err := p.debugValue(data)
		p.indent--
		return err

	case 18: // Flavor pick
		return p.debugFlavor(header, data)

	case 19: // Void
		return p.printf("[%02x] VOID", header)

	case 20: // Injection
		n, ok := internal.DecodeBytesToUint(data)
		if !ok {
			return errors.New("failed to decode index number of injection")
		}
		return p.printf("[%02x %02x] injection %d", header, data, n)

	default:
		return p.printf("[%02x] unknown extension: %02x", header, data)
	}
}

func (p *printer) debugFlavor(header, data []byte) error {
	full := data
	selector, sz := binary.Uvarint(data)
	if sz <= 0 {
		return internal.ErrCorruptedFlavorData
	}
	data = data[sz:]
	numCases, sz := binary.Uvarint(data)
	if sz <= 0 {
		return internal.ErrCorruptedFlavorData
	}
	data = data[sz:]
	hasElse := numCases&1 == 1
	numCases >>= 1
	p.printf("[%02x %02x] flavor selector (field: %d, cases: %d, else: %v)", header, full[:cap(full)-cap(data)], selector, numCases, hasElse)
	p.indent++
	var jumpTargets []uint64
	for numCases > 0 {
		n, sz := binary.Uvarint(data)
		if sz <= 0 {
			return internal.ErrCorruptedFlavorData
		}
		data = data[sz:]

		j, sz := binary.Uvarint(data)
		if sz <= 0 {
			return internal.ErrCorruptedFlavorData
		}
		data = data[sz:]
		p.printf("case %d: jump %d", n, j)
		jumpTargets = append(jumpTargets, j)
		numCases--
	}
	if hasElse {
		j, sz := binary.Uvarint(data)
		if sz <= 0 {
			return internal.ErrCorruptedFlavorData
		}
		p.printf("else: jump %d", j)
		jumpTargets = append(jumpTargets, j)
	}
	slices.Sort(jumpTargets)
	jumpTargets = slices.Compact(jumpTargets)
	for _, j := range jumpTargets {
		p.printf("offset %d:", j)
		p.indent++
		_, err := p.debugValue(full[j:])
		p.indent--
		if err != nil {
			return err
		}
	}
	p.indent--
	return nil
}
