// Copyright 2020 The Ebiten Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build android || ios
// +build android ios

package gles

// #include <stdlib.h>
import "C"

import (
	"unsafe"
)

func cString(str string) (uintptr, func()) {
	ptr := C.CString(str)
	return uintptr(unsafe.Pointer(ptr)), func() { C.free(unsafe.Pointer(ptr)) }
}

func cStringPtr(str string) (uintptr, func()) {
	s, free := cString(str)
	ptr := C.malloc(C.size_t(unsafe.Sizeof((*int)(nil))))
	*(*uintptr)(ptr) = s
	return uintptr(ptr), func() {
		free()
		C.free(ptr)
	}
}
