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

package jsutil

import (
	"syscall/js"
)

// isTypedArrayWritable represents whether TypedArray is writable or not.
// TypedArray's properties are not writable in the Web standard, but are writable with go2cpp.
// This enables to avoid unnecessary allocations of js.Value.
var isTypedArrayWritable = js.Global().Get("go2cpp").Truthy()

// temporaryBuffer is a temporary buffer used at gl.readPixels or gl.texSubImage2D.
// The read data is converted to Go's byte slice as soon as possible.
// To avoid often allocating ArrayBuffer, reuse the buffer whenever possible.
var temporaryBuffer = js.Global().Get("ArrayBuffer").New(16)

var uint8ArrayObj js.Value

func TemporaryUint8Array(byteLength int) js.Value {
	if bufl := temporaryBuffer.Get("byteLength").Int(); bufl < byteLength {
		for bufl < byteLength {
			bufl *= 2
		}
		temporaryBuffer = js.Global().Get("ArrayBuffer").New(bufl)
	}
	if isTypedArrayWritable {
		if uint8ArrayObj.IsUndefined() {
			uint8ArrayObj = js.Global().Get("Uint8Array").New()
		}
		uint8ArrayObj.Set("buffer", temporaryBuffer)
		uint8ArrayObj.Set("byteOffset", 0)
		uint8ArrayObj.Set("byteLength", byteLength)
		return uint8ArrayObj
	}
	return js.Global().Get("Uint8Array").New(temporaryBuffer, 0, byteLength)
}
