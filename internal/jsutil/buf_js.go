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

var (
	object       = js.Global().Get("Object")
	arrayBuffer  = js.Global().Get("ArrayBuffer")
	uint8Array   = js.Global().Get("Uint8Array")
	float32Array = js.Global().Get("Float32Array")
)

var (
	temporaryArrayBufferByteLength = 16

	// temporaryArrayBuffer is a temporary buffer used at gl.readPixels or gl.texSubImage2D.
	// The read data is converted to Go's byte slice as soon as possible.
	// To avoid often allocating ArrayBuffer, reuse the buffer whenever possible.
	temporaryArrayBuffer = arrayBuffer.New(temporaryArrayBufferByteLength)

	// temporaryUint8Array is a Uint8ArrayBuffer whose underlying buffer is always temporaryArrayBuffer.
	temporaryUint8Array = uint8Array.New(temporaryArrayBuffer)

	// temporaryFloat32Array is a Float32ArrayBuffer whose underlying buffer is always temporaryArrayBuffer.
	temporaryFloat32Array = float32Array.New(temporaryArrayBuffer)
)

func ensureTemporaryArrayBufferSize(byteLength int) {
	if bufl := temporaryArrayBufferByteLength; bufl < byteLength {
		for bufl < byteLength {
			bufl *= 2
		}
		temporaryArrayBufferByteLength = bufl
		temporaryArrayBuffer = arrayBuffer.New(bufl)
		temporaryUint8Array = uint8Array.New(temporaryArrayBuffer)
		temporaryFloat32Array = float32Array.New(temporaryArrayBuffer)
	}
}

// TemporaryUint8ArrayFromUint8Slice returns a Uint8Array whose length is at least minLength from a uint8 slice.
// Be careful that the length can exceed the given minLength.
// data must be a slice of a numeric type for initialization, or nil if you don't need initialization.
func TemporaryUint8ArrayFromUint8Slice(minLength int, data []uint8) js.Value {
	ensureTemporaryArrayBufferSize(minLength)
	copyUint8SliceToTemporaryArrayBuffer(data)
	return temporaryUint8Array
}

// TemporaryUint8ArrayFromUint16Slice returns a Uint8Array whose length is at least minLength from a uint16 slice.
// Be careful that the length can exceed the given minLength.
// data must be a slice of a numeric type for initialization, or nil if you don't need initialization.
func TemporaryUint8ArrayFromUint16Slice(minLength int, data []uint16) js.Value {
	ensureTemporaryArrayBufferSize(minLength * 2)
	copyUint16SliceToTemporaryArrayBuffer(data)
	return temporaryUint8Array
}

// TemporaryUint8ArrayFromFloat32Slice returns a Uint8Array whose length is at least minLength from a float32 slice.
// Be careful that the length can exceed the given minLength.
// data must be a slice of a numeric type for initialization, or nil if you don't need initialization.
func TemporaryUint8ArrayFromFloat32Slice(minLength int, data []float32) js.Value {
	ensureTemporaryArrayBufferSize(minLength * 4)
	copyFloat32SliceToTemporaryArrayBuffer(data)
	return temporaryUint8Array
}

// TemporaryFloat32Array returns a Float32Array whose length is at least minLength.
// Be careful that the length can exceed the given minLength.
// data must be a slice of a numeric type for initialization, or nil if you don't need initialization.
func TemporaryFloat32Array(minLength int, data []float32) js.Value {
	ensureTemporaryArrayBufferSize(minLength * 4)
	copyFloat32SliceToTemporaryArrayBuffer(data)
	return temporaryFloat32Array
}
