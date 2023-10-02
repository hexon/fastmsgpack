package msgpack_test

import (
	"bytes"
	"encoding/hex"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/hexon/fastmsgpack"
	"github.com/stretchr/testify/require"
	"github.com/vmihailenco/msgpack/v5"
)

type decoderTest struct {
	wanted any
	data   string
}

var decoderTests = []decoderTest{
	{nil, "c0"},

	{[]byte{1, 2, 3}, "c403010203"},

	{time.Unix(0, 0), "d6ff00000000"},
	{time.Unix(1, 1), "d7ff0000000400000001"},
	{time.Time{}.In(time.Local), "c70cff00000000fffffff1886e0900"},

	{[]any{}, "90"},

	{
		map[string]any{"a": "", "b": "", "c": "", "d": "", "e": ""},
		"85a161a0a162a0a163a0a164a0a165a0",
	},

	{float32(3.0), "ca40400000"},
	{float64(3.0), "cb4008000000000000"},
	{float64(-3.0), "cbc008000000000000"},
	{math.Inf(1), "cb7ff0000000000000"},
}

func TestDecoder(t *testing.T) {
	for i, test := range decoderTests {
		b, _ := hex.DecodeString(test.data)
		v, err := fastmsgpack.Decode(b, nil)
		if err != nil {
			t.Errorf("Failed to decode: %v", err)
			continue
		}
		require.Equal(t, test.wanted, v, "#%d (%s)", i, test.data)
	}
}

func TestStringsBin(t *testing.T) {
	tests := []struct {
		in     string
		wanted string
	}{
		{"", "a0"},
		{"a", "a161"},
		{"hello", "a568656c6c6f"},
		{
			strings.Repeat("x", 31),
			"bf" + strings.Repeat("78", 31),
		},
		{
			strings.Repeat("x", 32),
			"d920" + strings.Repeat("78", 32),
		},
		{
			strings.Repeat("x", 255),
			"d9ff" + strings.Repeat("78", 255),
		},
		{
			strings.Repeat("x", 256),
			"da0100" + strings.Repeat("78", 256),
		},
		{
			strings.Repeat("x", 65535),
			"daffff" + strings.Repeat("78", 65535),
		},
		{
			strings.Repeat("x", 65536),
			"db00010000" + strings.Repeat("78", 65536),
		},
	}

	for _, test := range tests {
		b, err := msgpack.Marshal(test.in)
		require.Nil(t, err)
		s := hex.EncodeToString(b)
		require.Equal(t, s, test.wanted)

		out, err := fastmsgpack.Decode(b, nil)
		require.Nil(t, err)
		require.Equal(t, out, test.in)
	}
}

func TestBin(t *testing.T) {
	tests := []struct {
		in     []byte
		wanted string
	}{
		{[]byte{}, "c400"},
		{[]byte{0}, "c40100"},
		{
			bytes.Repeat([]byte{'x'}, 31),
			"c41f" + strings.Repeat("78", 31),
		},
		{
			bytes.Repeat([]byte{'x'}, 32),
			"c420" + strings.Repeat("78", 32),
		},
		{
			bytes.Repeat([]byte{'x'}, 255),
			"c4ff" + strings.Repeat("78", 255),
		},
		{
			bytes.Repeat([]byte{'x'}, 256),
			"c50100" + strings.Repeat("78", 256),
		},
		{
			bytes.Repeat([]byte{'x'}, 65535),
			"c5ffff" + strings.Repeat("78", 65535),
		},
		{
			bytes.Repeat([]byte{'x'}, 65536),
			"c600010000" + strings.Repeat("78", 65536),
		},
	}

	for _, test := range tests {
		b, err := msgpack.Marshal(test.in)
		if err != nil {
			t.Fatal(err)
		}
		s := hex.EncodeToString(b)
		if s != test.wanted {
			t.Fatalf("%.32s != %.32s", s, test.wanted)
		}

		v, err := fastmsgpack.Decode(b, nil)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(v.([]byte), test.in) {
			t.Fatalf("%x != %x", v, test.in)
		}
	}
}

func TestUint64(t *testing.T) {
	tests := []struct {
		in     uint64
		wanted string
	}{
		{0, "00"},
		{1, "01"},
		{math.MaxInt8 - 1, "7e"},
		{math.MaxInt8, "7f"},
		{math.MaxInt8 + 1, "cc80"},
		{math.MaxUint8 - 1, "ccfe"},
		{math.MaxUint8, "ccff"},
		{math.MaxUint8 + 1, "cd0100"},
		{math.MaxUint16 - 1, "cdfffe"},
		{math.MaxUint16, "cdffff"},
		{math.MaxUint16 + 1, "ce00010000"},
		{math.MaxUint32 - 1, "cefffffffe"},
		{math.MaxUint32, "ceffffffff"},
		{math.MaxUint32 + 1, "cf0000000100000000"},
		{math.MaxInt64 - 1, "cf7ffffffffffffffe"},
		{math.MaxInt64, "cf7fffffffffffffff"},
	}

	var buf bytes.Buffer
	enc := msgpack.NewEncoder(&buf)
	enc.UseCompactInts(true)

	for _, test := range tests {
		buf.Reset()
		err := enc.Encode(test.in)
		if err != nil {
			t.Fatal(err)
		}
		s := hex.EncodeToString(buf.Bytes())
		if s != test.wanted {
			t.Fatalf("%.32s != %.32s", s, test.wanted)
		}

		out, err := fastmsgpack.Decode(buf.Bytes(), nil)
		if err != nil {
			t.Fatal(err)
		}
		if out.(int) != int(test.in) {
			t.Fatalf("%d != %d", out.(int), test.in)
		}
	}
}

func TestInt64(t *testing.T) {
	tests := []struct {
		in     int64
		wanted string
	}{
		{math.MinInt64, "d38000000000000000"},
		{math.MinInt32 - 1, "d3ffffffff7fffffff"},
		{math.MinInt32, "d280000000"},
		{math.MinInt32 + 1, "d280000001"},
		{math.MinInt16 - 1, "d2ffff7fff"},
		{math.MinInt16, "d18000"},
		{math.MinInt16 + 1, "d18001"},
		{math.MinInt8 - 1, "d1ff7f"},
		{math.MinInt8, "d080"},
		{math.MinInt8 + 1, "d081"},
		{-33, "d0df"},
		{-32, "e0"},
		{-31, "e1"},
		{-1, "ff"},
		{0, "00"},
		{1, "01"},
		{math.MaxInt8 - 1, "7e"},
		{math.MaxInt8, "7f"},
		{math.MaxInt8 + 1, "cc80"},
		{math.MaxUint8 - 1, "ccfe"},
		{math.MaxUint8, "ccff"},
		{math.MaxUint8 + 1, "cd0100"},
		{math.MaxUint16 - 1, "cdfffe"},
		{math.MaxUint16, "cdffff"},
		{math.MaxUint16 + 1, "ce00010000"},
		{math.MaxUint32 - 1, "cefffffffe"},
		{math.MaxUint32, "ceffffffff"},
		{math.MaxUint32 + 1, "cf0000000100000000"},
		{math.MaxInt64 - 1, "cf7ffffffffffffffe"},
		{math.MaxInt64, "cf7fffffffffffffff"},
	}

	var buf bytes.Buffer
	enc := msgpack.NewEncoder(&buf)
	enc.UseCompactInts(true)

	for _, test := range tests {
		buf.Reset()
		err := enc.Encode(test.in)
		if err != nil {
			t.Fatal(err)
		}
		s := hex.EncodeToString(buf.Bytes())
		if s != test.wanted {
			t.Fatalf("%.32s != %.32s", s, test.wanted)
		}

		out, err := fastmsgpack.Decode(buf.Bytes(), nil)
		if err != nil {
			t.Fatal(err)
		}
		if out.(int) != int(test.in) {
			t.Errorf("%d != %d for %s", out.(int), test.in, test.wanted)
		}
	}
}

func TestFloat32(t *testing.T) {
	tests := []struct {
		in     float32
		wanted string
	}{
		{0.1, "ca3dcccccd"},
		{0.2, "ca3e4ccccd"},
		{-0.1, "cabdcccccd"},
		{-0.2, "cabe4ccccd"},
		{float32(math.Inf(1)), "ca7f800000"},
		{float32(math.Inf(-1)), "caff800000"},
		{math.MaxFloat32, "ca7f7fffff"},
		{math.SmallestNonzeroFloat32, "ca00000001"},
	}
	for _, test := range tests {
		b, err := msgpack.Marshal(test.in)
		if err != nil {
			t.Fatal(err)
		}
		s := hex.EncodeToString(b)
		if s != test.wanted {
			t.Fatalf("%.32s != %.32s", s, test.wanted)
		}

		out, err := fastmsgpack.Decode(b, nil)
		if err != nil {
			t.Fatal(err)
		}
		if out.(float32) != test.in {
			t.Fatalf("%f != %f", out.(float32), test.in)
		}
	}
}

func TestFloat64(t *testing.T) {
	table := []struct {
		in     float64
		wanted string
	}{
		{0.1, "cb3fb999999999999a"},
		{0.2, "cb3fc999999999999a"},
		{-0.1, "cbbfb999999999999a"},
		{-0.2, "cbbfc999999999999a"},
		{math.Inf(1), "cb7ff0000000000000"},
		{math.Inf(-1), "cbfff0000000000000"},
		{math.MaxFloat64, "cb7fefffffffffffff"},
		{math.SmallestNonzeroFloat64, "cb0000000000000001"},
	}
	for _, test := range table {
		b, err := msgpack.Marshal(test.in)
		if err != nil {
			t.Fatal(err)
		}
		s := hex.EncodeToString(b)
		if s != test.wanted {
			t.Fatalf("%.32s != %.32s", s, test.wanted)
		}

		out, err := fastmsgpack.Decode(b, nil)
		if err != nil {
			t.Fatal(err)
		}
		if out.(float64) != test.in {
			t.Fatalf("%f != %f", out.(float64), test.in)
		}
	}
}
