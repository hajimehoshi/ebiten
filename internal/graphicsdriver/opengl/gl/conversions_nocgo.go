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
			str := make([]byte, i)
			copy(str, x[:i])
			return string(str)
		}
	}
	return ""
}

// Strs takes a list of Go strings (with or without null-termination) and
// returns their C counterpart.
//
// The returned free function must be called once you are done using the strings
// in order to free the memory.
//
// If no strings are provided as a parameter this function will panic.
func Strs(strs ...string) (cstrs **byte, free func()) {
	if len(strs) == 0 {
		panic("gl: expected at least 1 string at Strs")
	}

	pinned := make([][]byte, 0, len(strs))
	ptrs := make([]*byte, 0, len(strs))
	for _, str := range strs {
		bs := []byte(str)
		if len(bs) == 0 || bs[len(bs)-1] != 0 {
			bs = append(bs, 0)
		}
		pinned = append(pinned, bs)
		ptrs = append(ptrs, &bs[0])
	}

	return &ptrs[0], func() {
		runtime.KeepAlive(pinned)
		pinned = nil
	}
}
