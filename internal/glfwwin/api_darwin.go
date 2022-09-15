package glfwwin

import (
	"reflect"
	"unsafe"
)

func GoString(p uintptr) string {
	if p == 0 {
		return ""
	}
	var length int
	for {
		// use unsafe.Add once we reach 1.17
		if *(*byte)(unsafe.Pointer(uintptr(unsafe.Pointer(p)) + uintptr(length))) == '\x00' {
			break
		}
		length++
	}
	// use unsafe.Slice once we reach 1.17
	s := make([]byte, length)
	var src []byte
	h := (*reflect.SliceHeader)(unsafe.Pointer(&src))
	h.Data = uintptr(unsafe.Pointer(p))
	h.Len = length
	h.Cap = length
	copy(s, src)
	return string(s)
}
