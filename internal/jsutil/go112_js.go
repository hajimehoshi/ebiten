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

// +build !go1.13 !wasm

package jsutil

import (
	"syscall/js"
)

func Uint8ArrayToSlice(value js.Value) []byte {
	// Note that TypedArrayOf cannot work correcly on Wasm.
	// See https://github.com/golang/go/issues/31980

	s := make([]byte, value.Get("byteLength").Int())
	a := js.TypedArrayOf(s)
	a.Call("set", value)
	a.Release()
	return s
}

func ArrayBufferToSlice(value js.Value) []byte {
	return Uint8ArrayToSlice(js.Global().Get("Uint8Array").New(value))
}

func CopySliceToJS(dst js.Value, src interface{}) {
	// Note that TypedArrayOf cannot work correcly on Wasm.
	// See https://github.com/golang/go/issues/31980

	a := js.TypedArrayOf(src)
	dst.Call("set", a)
	a.Release()
}
