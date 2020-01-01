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

// temporaryBuffer is a temporary buffer used at gl.readPixels or gl.texSubImage2D.
// The read data is converted to Go's byte slice as soon as possible.
// To avoid often allocating ArrayBuffer, reuse the buffer whenever possible.
var temporaryBuffer = js.Global().Get("ArrayBuffer").New(16)

func TemporaryUint8Array(byteLength int) js.Value {
	if bufl := temporaryBuffer.Get("byteLength").Int(); bufl < byteLength {
		for bufl < byteLength {
			bufl *= 2
		}
		temporaryBuffer = js.Global().Get("ArrayBuffer").New(bufl)
	}
	return js.Global().Get("Uint8Array").New(temporaryBuffer, 0, byteLength)
}
