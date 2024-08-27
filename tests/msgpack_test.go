package msgpack_test

import (
	"bytes"
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/hexon/fastmsgpack"
	"github.com/stretchr/testify/suite"
)

func TestMsgpackTestSuite(t *testing.T) {
	suite.Run(t, new(MsgpackTest))
}

type MsgpackTest struct {
	suite.Suite

	buf []byte
	enc fastmsgpack.EncodeOptions
}

func (t *MsgpackTest) encode(v any) error {
	dst, err := t.enc.Encode(t.buf[:0], v)
	t.buf = dst
	return err
}

func (t *MsgpackTest) decode() any {
	ret, err := fastmsgpack.Decode(t.buf)
	if err != nil {
		t.T().Errorf("Failed to decode: %v", err)
		return nil
	}
	return ret
}

func (t *MsgpackTest) TestTime() {
	in := time.Now()

	t.Nil(t.encode(in))
	out := t.decode().(time.Time)
	t.True(out.Equal(in))

	var zero time.Time
	t.Nil(t.encode(zero))
	out = t.decode().(time.Time)
	t.True(out.Equal(zero))
	t.True(out.IsZero())

}

func (t *MsgpackTest) TestLargeBytes() {
	N := int(1e6)

	src := bytes.Repeat([]byte{'1'}, N)
	t.Nil(t.encode(src))
	dst := t.decode().([]byte)
	t.Equal(dst, src)
}

func (t *MsgpackTest) TestNaN() {
	t.Nil(t.encode(float64(math.NaN())))
	out := t.decode().(float64)
	t.True(math.IsNaN(out))
}

func (t *MsgpackTest) TestLargeString() {
	N := int(1e6)

	src := string(bytes.Repeat([]byte{'1'}, N))
	t.Nil(t.encode(src))
	dst := t.decode().(string)
	t.Equal(dst, src)
}

func (t *MsgpackTest) TestMap() {
	for _, i := range []struct {
		m map[string]any
		b []byte
	}{
		{map[string]any{}, []byte{0x80}},
		{map[string]any{"hello": "world"}, []byte{0x81, 0xa5, 0x68, 0x65, 0x6c, 0x6c, 0x6f, 0xa5, 0x77, 0x6f, 0x72, 0x6c, 0x64}},
	} {
		t.Nil(t.encode(i.m))
		t.Equal(t.buf, i.b, fmt.Errorf("err encoding %v", i.m))
		m := t.decode().(map[string]any)
		t.Equal(m, i.m)
	}
}

func (t *MsgpackTest) TestInts() {
	for i := -70000; 70000 > i; i++ {
		t.Nil(t.encode(i))
		n := t.decode().(int)
		t.Equal(i, n)
	}
}

func (t *MsgpackTest) TestInt8s() {
	for i := math.MinInt8; math.MaxInt8 > i; i++ {
		t.Nil(t.encode(int8(i)))
		n := t.decode().(int)
		t.Equal(i, n)
	}
}

func (t *MsgpackTest) TestInt16s() {
	for i := math.MinInt16; math.MaxInt16 > i; i++ {
		t.Nil(t.encode(int16(i)))
		n := t.decode().(int)
		t.Equal(i, n)
	}
}

func (t *MsgpackTest) TestUint16s() {
	for i := 0; math.MaxUint16 > i; i++ {
		t.Nil(t.encode(uint16(i)))
		n := t.decode().(int)
		t.Equal(i, n)
	}
}
