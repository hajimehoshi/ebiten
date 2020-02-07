// Copyright 2019 The Ebiten Authors
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

// +build go1.13

package jsutil

import (
	"fmt"
	"syscall/js"
)

func Uint8ArrayToSlice(value js.Value) []byte {
	s := make([]byte, value.Get("byteLength").Int())
	js.CopyBytesToGo(s, value)
	return s
}

func ArrayBufferToSlice(value js.Value) []byte {
	return Uint8ArrayToSlice(js.Global().Get("Uint8Array").New(value))
}

func CopySliceToJS(dst js.Value, src interface{}) {
	switch s := src.(type) {
	case []uint8:
		js.CopyBytesToJS(dst, s)
	case []int8, []int16, []int32, []uint16, []uint32, []float32, []float64:
		a := js.Global().Get("Uint8Array").New(dst.Get("buffer"), dst.Get("byteOffset"), dst.Get("byteLength"))
		js.CopyBytesToJS(a, sliceToByteSlice(s))
	default:
		panic(fmt.Sprintf("jsutil: unexpected value at CopySliceToJS: %T", s))
	}
}
