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

var go2cpp = js.Global().Get("go2cpp").Truthy()

var (
	object       = js.Global().Get("Object")
	arrayBuffer  = js.Global().Get("ArrayBuffer")
	uint8Array   = js.Global().Get("Uint8Array")
	float32Array = js.Global().Get("Float32Array")
)

var (
	// temporaryArrayBuffer is a temporary buffer used at gl.readPixels or gl.texSubImage2D.
	// The read data is converted to Go's byte slice as soon as possible.
	// To avoid often allocating ArrayBuffer, reuse the buffer whenever possible.
	temporaryArrayBuffer = arrayBuffer.New(16)

	// temporaryUint8Array is a Uint8ArrayBuffer whose underlying buffer is always temporaryArrayBuffer.
	temporaryUint8Array = uint8Array.New(temporaryArrayBuffer)

	// temporaryFloat32Array is a Float32ArrayBuffer whose underlying buffer is always temporaryArrayBuffer.
	temporaryFloat32Array = float32Array.New(temporaryArrayBuffer)
)

var (
	temporaryArrayBufferByteLengthFunc  js.Value
	temporaryUint8ArrayByteLengthFunc   js.Value
	temporaryFloat32ArrayByteLengthFunc js.Value
)

func init() {
	if go2cpp {
		return
	}
	temporaryArrayBufferByteLengthFunc = object.Call("getOwnPropertyDescriptor", arrayBuffer.Get("prototype"), "byteLength").Get("get").Call("bind", temporaryArrayBuffer)
	temporaryUint8ArrayByteLengthFunc = object.Call("getOwnPropertyDescriptor", object.Call("getPrototypeOf", uint8Array).Get("prototype"), "byteLength").Get("get").Call("bind", temporaryUint8Array)
	temporaryFloat32ArrayByteLengthFunc = object.Call("getOwnPropertyDescriptor", object.Call("getPrototypeOf", float32Array).Get("prototype"), "byteLength").Get("get").Call("bind", temporaryFloat32Array)
}

func temporaryArrayBufferByteLength() int {
	if go2cpp {
		return temporaryArrayBuffer.Get("byteLength").Int()
	}
	return temporaryArrayBufferByteLengthFunc.Invoke().Int()
}

func temporaryUint8ArrayByteLength() int {
	if go2cpp {
		return temporaryUint8Array.Get("byteLength").Int()
	}
	return temporaryUint8ArrayByteLengthFunc.Invoke().Int()
}

func temporaryFloat32ArrayByteLength() int {
	if go2cpp {
		return temporaryFloat32Array.Get("byteLength").Int()
	}
	return temporaryFloat32ArrayByteLengthFunc.Invoke().Int()
}

func ensureTemporaryArrayBufferSize(byteLength int) {
	bufl := temporaryArrayBufferByteLength()
	if bufl < byteLength {
		for bufl < byteLength {
			bufl *= 2
		}
		temporaryArrayBuffer = arrayBuffer.New(bufl)
	}
	if temporaryUint8ArrayByteLength() < bufl {
		temporaryUint8Array = uint8Array.New(temporaryArrayBuffer)
	}
	if temporaryFloat32ArrayByteLength() < bufl {
		temporaryFloat32Array = float32Array.New(temporaryArrayBuffer)
	}
}

// TemporaryUint8Array returns a Uint8Array whose length is at least minLength.
// Be careful that the length can exceed the given minLength.
// data must be a slice of a numeric type for initialization, or nil if you don't need initialization.
func TemporaryUint8Array(minLength int, data interface{}) js.Value {
	ensureTemporaryArrayBufferSize(minLength)
	if data != nil {
		copySliceToTemporaryArrayBuffer(data)
	}
	return temporaryUint8Array
}

// TemporaryFloat32Array returns a Float32Array whose length is at least minLength.
// Be careful that the length can exceed the given minLength.
// data must be a slice of a numeric type for initialization, or nil if you don't need initialization.
func TemporaryFloat32Array(minLength int, data interface{}) js.Value {
	ensureTemporaryArrayBufferSize(minLength * 4)
	if data != nil {
		copySliceToTemporaryArrayBuffer(data)
	}
	return temporaryFloat32Array
}
