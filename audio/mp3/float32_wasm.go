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

// +build js,wasm

package mp3

import (
	"reflect"
	"syscall/js"
	"unsafe"
)

func float32ArrayToSlice(arr js.Value) []float32 {
	bytes := make([]byte, arr.Length()*4)
	buf := arr.Get("buffer").Call("slice", arr.Get("byteOffset"), arr.Get("byteOffset").Int()+arr.Get("byteLength").Int())
	js.ValueOf(bytes).Call("set", js.Global().Get("Uint8Array").New(buf))

	bh := (*reflect.SliceHeader)(unsafe.Pointer(&bytes))
	var f []float32
	fh := (*reflect.SliceHeader)(unsafe.Pointer(&f))

	fh.Data = bh.Data
	fh.Len = bh.Len / 4
	fh.Cap = bh.Cap / 4
	return f
}
