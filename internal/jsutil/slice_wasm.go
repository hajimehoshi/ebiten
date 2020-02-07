// +build js,wasm

package jsutil

import (
	"fmt"
	"reflect"
	"runtime"
	"unsafe"
)

func sliceToByteSlice(s interface{}) (bs []byte) {
	switch s := s.(type) {
	case []int8:
		h := (*reflect.SliceHeader)(unsafe.Pointer(&s))
		bs = *(*[]byte)(unsafe.Pointer(h))
		runtime.KeepAlive(s)
	case []int16:
		h := (*reflect.SliceHeader)(unsafe.Pointer(&s))
		h.Len *= 2
		h.Cap *= 2
		bs = *(*[]byte)(unsafe.Pointer(h))
		runtime.KeepAlive(s)
	case []int32:
		h := (*reflect.SliceHeader)(unsafe.Pointer(&s))
		h.Len *= 4
		h.Cap *= 4
		bs = *(*[]byte)(unsafe.Pointer(h))
		runtime.KeepAlive(s)
	case []int64:
		h := (*reflect.SliceHeader)(unsafe.Pointer(&s))
		h.Len *= 8
		h.Cap *= 8
		bs = *(*[]byte)(unsafe.Pointer(h))
		runtime.KeepAlive(s)
	case []uint8:
		return s
	case []uint16:
		h := (*reflect.SliceHeader)(unsafe.Pointer(&s))
		h.Len *= 2
		h.Cap *= 2
		bs = *(*[]byte)(unsafe.Pointer(h))
		runtime.KeepAlive(s)
	case []uint32:
		h := (*reflect.SliceHeader)(unsafe.Pointer(&s))
		h.Len *= 4
		h.Cap *= 4
		bs = *(*[]byte)(unsafe.Pointer(h))
		runtime.KeepAlive(s)
	case []uint64:
		h := (*reflect.SliceHeader)(unsafe.Pointer(&s))
		h.Len *= 8
		h.Cap *= 8
		bs = *(*[]byte)(unsafe.Pointer(h))
		runtime.KeepAlive(s)
	case []float32:
		h := (*reflect.SliceHeader)(unsafe.Pointer(&s))
		h.Len *= 4
		h.Cap *= 4
		bs = *(*[]byte)(unsafe.Pointer(h))
		runtime.KeepAlive(s)
	case []float64:
		h := (*reflect.SliceHeader)(unsafe.Pointer(&s))
		h.Len *= 8
		h.Cap *= 8
		bs = *(*[]byte)(unsafe.Pointer(h))
		runtime.KeepAlive(s)
	default:
		panic(fmt.Sprintf("jsutil: unexpected value at sliceToBytesSlice: %T", s))
	}
	return
}
