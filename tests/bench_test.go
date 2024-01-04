package msgpack_test

import (
	"bytes"
	"encoding/json"
	"math"
	"testing"
	"time"

	"github.com/hexon/fastmsgpack"
	"github.com/vmihailenco/msgpack/v5"
)

func benchmarkEncodeDecode(b *testing.B, src, dst interface{}) {
	b.Run("fastmsgpack", func(b *testing.B) {
		var buf []byte
		enc := fastmsgpack.EncodeOptions{}

		b.ResetTimer()

		var err error
		for i := 0; i < b.N; i++ {
			buf, err = enc.Encode(buf[:0], src)
			if err != nil {
				b.Fatal(err)
			}
			if _, err := fastmsgpack.Decode(buf, nil); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("vmihailenco/msgpack/v5", func(b *testing.B) {
		var buf bytes.Buffer
		enc := msgpack.NewEncoder(&buf)

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			buf.Reset()
			if err := enc.Encode(src); err != nil {
				b.Fatal(err)
			}
			if _, err := fastmsgpack.Decode(buf.Bytes(), nil); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("json", func(b *testing.B) {
		var buf bytes.Buffer
		enc := json.NewEncoder(&buf)
		dec := json.NewDecoder(&buf)

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			if err := enc.Encode(src); err != nil {
				b.Fatal(err)
			}
			if err := dec.Decode(dst); err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkBool(b *testing.B) {
	var dst bool
	benchmarkEncodeDecode(b, true, &dst)
}

func BenchmarkInt0(b *testing.B) {
	var dst int
	benchmarkEncodeDecode(b, 1, &dst)
}

func BenchmarkInt1(b *testing.B) {
	var dst int
	benchmarkEncodeDecode(b, -33, &dst)
}

func BenchmarkInt2(b *testing.B) {
	var dst int
	benchmarkEncodeDecode(b, 128, &dst)
}

func BenchmarkInt4(b *testing.B) {
	var dst int
	benchmarkEncodeDecode(b, 32768, &dst)
}

func BenchmarkInt8(b *testing.B) {
	var dst int
	benchmarkEncodeDecode(b, int64(2147483648), &dst)
}

func BenchmarkInt32(b *testing.B) {
	var dst int32
	benchmarkEncodeDecode(b, int32(0), &dst)
}

func BenchmarkFloat32(b *testing.B) {
	var dst float32
	benchmarkEncodeDecode(b, float32(0), &dst)
}

func BenchmarkFloat32_Max(b *testing.B) {
	var dst float32
	benchmarkEncodeDecode(b, float32(math.MaxFloat32), &dst)
}

func BenchmarkFloat64(b *testing.B) {
	var dst float64
	benchmarkEncodeDecode(b, float64(0), &dst)
}

func BenchmarkFloat64_Max(b *testing.B) {
	var dst float64
	benchmarkEncodeDecode(b, float64(math.MaxFloat64), &dst)
}

func BenchmarkTime(b *testing.B) {
	var dst time.Time
	benchmarkEncodeDecode(b, time.Now(), &dst)
}

func BenchmarkByteSlice(b *testing.B) {
	src := make([]byte, 1024)
	var dst []byte
	benchmarkEncodeDecode(b, src, &dst)
}

func BenchmarkByteArray(b *testing.B) {
	var src [1024]byte
	var dst [1024]byte
	benchmarkEncodeDecode(b, src, &dst)
}

func BenchmarkByteArrayPtr(b *testing.B) {
	var src [1024]byte
	var dst [1024]byte
	benchmarkEncodeDecode(b, &src, &dst)
}

func BenchmarkMapStringString(b *testing.B) {
	src := map[string]string{
		"hello": "world",
		"foo":   "bar",
	}
	var dst map[string]string
	benchmarkEncodeDecode(b, src, &dst)
}

func BenchmarkMapStringStringPtr(b *testing.B) {
	src := map[string]string{
		"hello": "world",
		"foo":   "bar",
	}
	dst := new(map[string]string)
	benchmarkEncodeDecode(b, src, &dst)
}

func BenchmarkMapStringInterface(b *testing.B) {
	src := map[string]interface{}{
		"hello": "world",
		"foo":   "bar",
		"one":   1111111,
		"two":   2222222,
	}
	var dst map[string]interface{}
	benchmarkEncodeDecode(b, src, &dst)
}

func BenchmarkStringSlice(b *testing.B) {
	src := []string{"hello", "world"}
	var dst []string
	benchmarkEncodeDecode(b, src, &dst)
}

func BenchmarkStringSlicePtr(b *testing.B) {
	src := []string{"hello", "world"}
	var dst []string
	dstptr := &dst
	benchmarkEncodeDecode(b, src, &dstptr)
}

/*
func BenchmarkVmihailencoQuery(b *testing.B) {
	var records []map[string]interface{}
	for i := 0; i < 1000; i++ {
		record := map[string]interface{}{
			"id":    int64(i),
			"attrs": map[string]interface{}{"phone": int64(i)},
		}
		records = append(records, record)
	}

	bs, err := msgpack.Marshal(records)
	if err != nil {
		b.Fatal(err)
	}

	dec := msgpack.NewDecoder(bytes.NewBuffer(bs))

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		dec.Reset(bytes.NewBuffer(bs))

		values, err := dec.Query("10.attrs.phone")
		if err != nil {
			b.Fatal(err)
		}
		if values[0].(int64) != 10 {
			b.Fatalf("%v != %d", values[0], 10)
		}
	}
}

func BenchmarkResolve(b *testing.B) {
	var records []map[string]interface{}
	for i := 0; i < 1000; i++ {
		record := map[string]interface{}{
			"id":    int64(i),
			"attrs": map[string]interface{}{"phone": int64(i)},
		}
		records = append(records, record)
	}

	bs, err := msgpack.Marshal(records)
	if err != nil {
		b.Fatal(err)
	}

	dec := msgpack.NewDecoder(bytes.NewBuffer(bs))

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		dec.Reset(bytes.NewBuffer(bs))

		values, err := dec.Query("10.attrs.phone")
		if err != nil {
			b.Fatal(err)
		}
		if values[0].(int64) != 10 {
			b.Fatalf("%v != %d", values[0], 10)
		}
	}
}
*/
