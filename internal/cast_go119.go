//go:build !go1.20

package internal

import "github.com/alecthomas/unsafeslice"

func UnsafeStringCast(data []byte) string {
	return unsafeslice.StringFromByteSlice(data)
}
