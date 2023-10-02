package msgpack_test

import (
	"bytes"
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/hexon/fastmsgpack"
	"github.com/stretchr/testify/suite"
	"github.com/vmihailenco/msgpack/v5"
)

func TestMsgpackTestSuite(t *testing.T) {
	suite.Run(t, new(MsgpackTest))
}

type MsgpackTest struct {
	suite.Suite

	buf *bytes.Buffer
	enc *msgpack.Encoder
}

func (t *MsgpackTest) SetupTest() {
	t.buf = &bytes.Buffer{}
	t.enc = msgpack.NewEncoder(t.buf)
}

func (t *MsgpackTest) decode() any {
	ret, err := fastmsgpack.Decode(t.buf.Bytes(), nil)
	if err != nil {
		t.T().Errorf("Failed to decode: %v", err)
		return nil
	}
	t.buf.Reset()
	return ret
}

func (t *MsgpackTest) TestTime() {
	in := time.Now()

	t.Nil(t.enc.Encode(in))
	out := t.decode().(time.Time)
	t.True(out.Equal(in))

	var zero time.Time
	t.Nil(t.enc.Encode(zero))
	out = t.decode().(time.Time)
	t.True(out.Equal(zero))
	t.True(out.IsZero())

}

func (t *MsgpackTest) TestLargeBytes() {
	N := int(1e6)

	src := bytes.Repeat([]byte{'1'}, N)
	t.Nil(t.enc.Encode(src))
	dst := t.decode().([]byte)
	t.Equal(dst, src)
}

func (t *MsgpackTest) TestNaN() {
	t.Nil(t.enc.Encode(float64(math.NaN())))
	out := t.decode().(float64)
	t.True(math.IsNaN(out))
}

func (t *MsgpackTest) TestLargeString() {
	N := int(1e6)

	src := string(bytes.Repeat([]byte{'1'}, N))
	t.Nil(t.enc.Encode(src))
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
		t.Nil(t.enc.Encode(i.m))
		t.Equal(t.buf.Bytes(), i.b, fmt.Errorf("err encoding %v", i.m))
		m := t.decode().(map[string]any)
		t.Equal(m, i.m)
	}
}
