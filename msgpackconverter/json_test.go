package msgpackconverter

import (
	"bytes"
	"testing"

	"github.com/hexon/fastmsgpack"
)

func TestJSON(t *testing.T) {
	tests := []struct {
		data      any
		hideNulls bool
		want      string
	}{
		{
			data: 5,
			want: "5",
		},
		{
			data: "str",
			want: "\"str\"",
		},
		{
			data: true,
			want: "true",
		},
		{
			data: nil,
			want: "null",
		},
		{
			data: []any{1, 2, 3},
			want: "[1,2,3]",
		},
		{
			data:      []any{1, nil, 3},
			hideNulls: true,
			want:      "[1,3]",
		},
		{
			data: []any{1, 2, 3},
			want: "[1,2,3]",
		},
		{
			data:      map[string]any{"x": 5, "y": nil},
			hideNulls: true,
			want:      "{\"x\":5}",
		},
	}
	for _, tc := range tests {
		t.Run(tc.want, func(t *testing.T) {
			msgp, err := fastmsgpack.Encode(nil, tc.data)
			if err != nil {
				t.Fatalf("Can't convert data to msgpack: %v", err)
			}
			var opts []fastmsgpack.DecodeOption
			if tc.hideNulls {
				opts = append(opts, WithHideNulls())
			}
			j := NewJSONConverter(opts...)
			var got bytes.Buffer
			if err := j.Convert(&got, msgp); err != nil {
				t.Fatalf("Failed to convert msgpack to JSON: %v", err)
			}
			if got.String() != tc.want {
				t.Errorf("Didn't get expected result: %s; want %s", got.String(), tc.want)
			}
		})
	}
}
