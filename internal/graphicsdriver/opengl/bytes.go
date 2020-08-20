// Copyright 2018 The Ebiten Authors
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

// +build android ios

package opengl

import (
	"reflect"
	"unsafe"
)

func float32sToBytes(v []float32) []byte {
	f32h := (*reflect.SliceHeader)(unsafe.Pointer(&v))

	var b []byte
	bh := (*reflect.SliceHeader)(unsafe.Pointer(&b))
	bh.Data = f32h.Data
	bh.Len = len(v) * 4
	bh.Cap = len(v) * 4
	return b
}

func uint16sToBytes(v []uint16) []byte {
	u16h := (*reflect.SliceHeader)(unsafe.Pointer(&v))

	var b []byte
	bh := (*reflect.SliceHeader)(unsafe.Pointer(&b))
	bh.Data = u16h.Data
	bh.Len = len(v) * 2
	bh.Cap = len(v) * 2
	return b
}
