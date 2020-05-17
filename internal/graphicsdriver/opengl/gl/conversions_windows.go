// SPDX-License-Identifier: MIT

package gl

import (
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"unsafe"
)

// Ptr takes a slice or pointer (to a singular scalar value or the first
// element of an array or slice) and returns its GL-compatible address.
//
// For example:
//
// 	var data []uint8
// 	...
// 	gl.TexImage2D(gl.TEXTURE_2D, ..., gl.UNSIGNED_BYTE, gl.Ptr(&data[0]))
func Ptr(data interface{}) unsafe.Pointer {
	if data == nil {
		return unsafe.Pointer(nil)
	}
	var addr unsafe.Pointer
	switch v := data.(type) {
	case *uint8:
		addr = unsafe.Pointer(v)
	case *uint16:
		addr = unsafe.Pointer(v)
	case *float32:
		addr = unsafe.Pointer(v)
	case []uint8:
		addr = unsafe.Pointer(&v[0])
	case []uint16:
		addr = unsafe.Pointer(&v[0])
	case []float32:
		addr = unsafe.Pointer(&v[0])
	default:
		panic(fmt.Errorf("unsupported type %T; must be a slice or pointer to a singular scalar value or the first element of an array or slice", v))
	}
	return addr
}

// Str takes a null-terminated Go string and returns its GL-compatible address.
// This function reaches into Go string storage in an unsafe way so the caller
// must ensure the string is not garbage collected.
func Str(str string) *uint8 {
	if !strings.HasSuffix(str, "\x00") {
		panic("str argument missing null terminator: " + str)
	}
	header := (*reflect.StringHeader)(unsafe.Pointer(&str))
	return (*uint8)(unsafe.Pointer(header.Data))
}

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
