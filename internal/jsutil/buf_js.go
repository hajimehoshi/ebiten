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

// temporaryArrayBuffer is a temporary buffer used at gl.readPixels or gl.texSubImage2D.
// The read data is converted to Go's byte slice as soon as possible.
// To avoid often allocating ArrayBuffer, reuse the buffer whenever possible.
var temporaryArrayBuffer = js.Global().Get("ArrayBuffer").New(16)

var temporaryFloat32Array = js.Global().Get("Float32Array").New(temporaryArrayBuffer)

func ensureTemporaryArrayBufferSize(byteLength int) {
	if bufl := temporaryArrayBuffer.Get("byteLength").Int(); bufl < byteLength {
		for bufl < byteLength {
			bufl *= 2
		}
		temporaryArrayBuffer = js.Global().Get("ArrayBuffer").New(bufl)
	}
}

func ensureTemporaryFloat32ArraySize(length int) {
	ensureTemporaryArrayBufferSize(length * 4)
	if temporaryFloat32Array.Get("byteLength").Int() < temporaryArrayBuffer.Get("byteLength").Int() {
		temporaryFloat32Array = js.Global().Get("Float32Array").New(temporaryArrayBuffer)
	}
}

func TemporaryUint8Array(byteLength int) js.Value {
	ensureTemporaryArrayBufferSize(byteLength)
	return uint8Array(temporaryArrayBuffer, 0, byteLength)
}

var uint8ArrayObj js.Value

func uint8Array(buffer js.Value, byteOffset, byteLength int) js.Value {
	if isTypedArrayWritable {
		if Equal(uint8ArrayObj, js.Undefined()) {
			uint8ArrayObj = js.Global().Get("Uint8Array").New()
		}
		uint8ArrayObj.Set("buffer", buffer)
		uint8ArrayObj.Set("byteOffset", byteOffset)
		uint8ArrayObj.Set("byteLength", byteLength)
		return uint8ArrayObj
	}
	return js.Global().Get("Uint8Array").New(buffer, byteOffset, byteLength)
}

// TemporaryFloat32Array returns a Float32Array whose length is at least minLength.
// Be careful that the length can exceed the given minLength.
func TemporaryFloat32Array(minLength int) js.Value {
	ensureTemporaryFloat32ArraySize(minLength * 4)
	return temporaryFloat32Array
}
