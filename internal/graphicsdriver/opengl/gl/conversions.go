// SPDX-License-Identifier: MIT

//go:build !android && !ios && !js && !opengles

package gl

import (
	"fmt"
	"unsafe"
)

// Ptr takes a slice or pointer (to a singular scalar value or the first
// element of an array or slice) and returns its GL-compatible address.
//
// For example:
//
//	var data []byte
//	...
//	gl.TexImage2D(gl.TEXTURE_2D, ..., gl.UNSIGNED_BYTE, gl.Ptr(&data[0]))
func Ptr(data any) unsafe.Pointer {
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
	case []uint32:
		addr = unsafe.Pointer(&v[0])
	case []float32:
		addr = unsafe.Pointer(&v[0])
	default:
		panic(fmt.Errorf("unsupported type %T; must be a slice or pointer to a singular scalar value or the first element of an array or slice", v))
	}
	return addr
}
