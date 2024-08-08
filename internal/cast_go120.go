//go:build go1.20

package internal

import "unsafe"

func UnsafeStringCast(data []byte) string {
	return unsafe.String(unsafe.SliceData(data), len(data))
}
