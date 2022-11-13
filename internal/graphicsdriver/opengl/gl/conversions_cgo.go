// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2014 Eric Woroshow

//go:build !android && !darwin && !js && !windows && !opengles

package gl

import (
	"unsafe"
)

// #include <stdlib.h>
import "C"

// cStr takes a Go string (with or without null-termination)
// and returns the C counterpart.
//
// The returned free function must be called once you are done using the string
// in order to free the memory.
func cStr(str string) (cstr *byte, free func()) {
	cs := C.CString(str)
	return (*byte)(unsafe.Pointer(cs)), func() {
		C.free(unsafe.Pointer(cs))
	}
}
