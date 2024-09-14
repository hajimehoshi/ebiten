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

package gl

import (
	"syscall/js"
)

var (
	object       = js.Global().Get("Object")
	arrayBuffer  = js.Global().Get("ArrayBuffer")
	uint8Array   = js.Global().Get("Uint8Array")
	float32Array = js.Global().Get("Float32Array")
	int32Array   = js.Global().Get("Int32Array")
)

var (
	tmpArrayBufferByteLength = 16

	// tmpArrayBuffer is a temporary buffer used at gl.readPixels or gl.texSubImage2D.
	// The read data is converted to Go's byte slice as soon as possible.
	// To avoid often allocating ArrayBuffer, reuse the buffer whenever possible.
	tmpArrayBuffer = arrayBuffer.New(tmpArrayBufferByteLength)

	// tmpUint8Array is a Uint8ArrayBuffer whose underlying buffer is always temporaryArrayBuffer.
	tmpUint8Array = uint8Array.New(tmpArrayBuffer)

	// tmpFloat32Array is a Float32ArrayBuffer whose underlying buffer is always temporaryArrayBuffer.
	tmpFloat32Array = float32Array.New(tmpArrayBuffer)

	// tmpInt32Array is a Float32ArrayBuffer whose underlying buffer is always temporaryArrayBuffer.
	tmpInt32Array = int32Array.New(tmpArrayBuffer)
)

func ensureTemporaryArrayBufferSize(byteLength int) {
	if bufl := tmpArrayBufferByteLength; bufl < byteLength {
		for bufl < byteLength {
			bufl *= 2
		}
		tmpArrayBufferByteLength = bufl
		tmpArrayBuffer = arrayBuffer.New(bufl)
		tmpUint8Array = uint8Array.New(tmpArrayBuffer)
		tmpFloat32Array = float32Array.New(tmpArrayBuffer)
		tmpInt32Array = int32Array.New(tmpArrayBuffer)
	}
}

// tmpUint8ArrayFromUint8Slice returns a Uint8Array whose length is at least minLength from an uint8 slice.
// Be careful that the length can exceed the given minLength.
// data must be a slice of a numeric type for initialization, or nil if you don't need initialization.
func tmpUint8ArrayFromUint8Slice(minLength int, data []uint8) js.Value {
	ensureTemporaryArrayBufferSize(minLength)
	copyUint8SliceToTemporaryArrayBuffer(data)
	return tmpUint8Array
}

// tmpUint8ArrayFromUint16Slice returns a Uint8Array whose length is at least minLength from an uint16 slice.
// Be careful that the length can exceed the given minLength.
// data must be a slice of a numeric type for initialization, or nil if you don't need initialization.
func tmpUint8ArrayFromUint16Slice(minLength int, data []uint16) js.Value {
	ensureTemporaryArrayBufferSize(minLength * 2)
	copySliceToTemporaryArrayBuffer(data)
	return tmpUint8Array
}

// tmpUint8ArrayFromFloat32Slice returns a Uint8Array whose length is at least minLength from a float32 slice.
// Be careful that the length can exceed the given minLength.
// data must be a slice of a numeric type for initialization, or nil if you don't need initialization.
func tmpUint8ArrayFromFloat32Slice(minLength int, data []float32) js.Value {
	ensureTemporaryArrayBufferSize(minLength * 4)
	copySliceToTemporaryArrayBuffer(data)
	return tmpUint8Array
}

// tmpFloat32ArrayFromFloat32Slice returns a Float32Array whose length is at least minLength.
// Be careful that the length can exceed the given minLength.
// data must be a slice of a numeric type for initialization, or nil if you don't need initialization.
func tmpFloat32ArrayFromFloat32Slice(minLength int, data []float32) js.Value {
	ensureTemporaryArrayBufferSize(minLength * 4)
	copySliceToTemporaryArrayBuffer(data)
	return tmpFloat32Array
}

// tmpInt32ArrayFromInt32Slice returns a Int32Array whose length is at least minLength.
// Be careful that the length can exceed the given minLength.
// data must be a slice of a numeric type for initialization, or nil if you don't need initialization.
func tmpInt32ArrayFromInt32Slice(minLength int, data []int32) js.Value {
	ensureTemporaryArrayBufferSize(minLength * 4)
	copySliceToTemporaryArrayBuffer(data)
	return tmpInt32Array
}
