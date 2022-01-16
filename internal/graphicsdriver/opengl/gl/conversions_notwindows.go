// SPDX-License-Identifier: MIT

//go:build !windows
// +build !windows

package gl

import (
	"unsafe"
)

// #include <stdlib.h>
import "C"

// GoStr takes a null-terminated string returned by OpenGL and constructs a
// corresponding Go string.
func GoStr(cstr *uint8) string {
	return C.GoString((*C.char)(unsafe.Pointer(cstr)))
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

	css := make([]*uint8, 0, len(strs))
	for _, str := range strs {
		cs := C.CString(str)
		css = append(css, (*uint8)(unsafe.Pointer(cs)))
	}

	return (**uint8)(&css[0]), func() {
		for _, cs := range css {
			C.free(unsafe.Pointer(cs))
		}
	}
}
