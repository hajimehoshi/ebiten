// SPDX-License-Identifier: MIT

package gl

import (
	"runtime"
	"strings"
	"unsafe"
)

// GoStr takes a null-terminated string returned by OpenGL and constructs a
// corresponding Go string.
func GoStr(cstr *uint8) string {
	str := ""
	for {
		if *cstr == 0 {
			break
		}
		str += string(*cstr)
		cstr = (*uint8)(unsafe.Pointer(uintptr(unsafe.Pointer(cstr)) + 1))
	}
	return str
}

// Strs takes a list of Go strings (with or without null-termination) and
// returns their C counterpart.
//
// The returned free function must be called once you are done using the strings
// in order to free the memory.
//
// If no strings are provided as a parameter this function will panic.
func Strs(strs ...string) (cstrs **uint8, free func()) {
	if len(strs) == 0 {
		panic("Strs: expected at least 1 string")
	}

	var pinned []string
	var ptrs []*uint8
	for _, str := range strs {
		if !strings.HasSuffix(str, "\x00") {
			str += "\x00"
		}
		pinned = append(pinned, str)
		ptrs = append(ptrs, Str(str))
	}

	return &ptrs[0], func() {
		runtime.KeepAlive(pinned)
		pinned = nil
	}
}
