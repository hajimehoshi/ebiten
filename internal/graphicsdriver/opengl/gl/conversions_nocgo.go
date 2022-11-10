// SPDX-License-Identifier: MIT

//go:build darwin || windows

package gl

import (
	"runtime"
	"unsafe"
)

// GoStr takes a null-terminated string returned by OpenGL and constructs a
// corresponding Go string.
func GoStr(cstr *byte) string {
	if cstr == nil {
		return ""
	}
	x := unsafe.Slice(cstr, 1e9)
	for i, c := range x {
		if c == 0 {
			return string(x[:i])
		}
	}
	return ""
}

// CStr takes a Go string (with or without null-termination)
// and returns the C counterpart.
//
// The returned free function must be called once you are done using the string
// in order to free the memory.
func CStr(str string) (cstr *byte, free func()) {
	bs := []byte(str)
	if len(bs) == 0 || bs[len(bs)-1] != 0 {
		bs = append(bs, 0)
	}
	return &bs[0], func() {
		runtime.KeepAlive(bs)
		bs = nil
	}
}
