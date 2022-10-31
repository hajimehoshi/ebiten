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
	"runtime"
	"syscall/js"
	"unsafe"
)

func copyUint8SliceToTemporaryArrayBuffer(src []uint8) {
	if len(src) == 0 {
		return
	}
	js.CopyBytesToJS(temporaryUint8Array, src)
}

func copyUint16SliceToTemporaryArrayBuffer(src []uint16) {
	if len(src) == 0 {
		return
	}
	js.CopyBytesToJS(temporaryUint8Array, unsafe.Slice((*byte)(unsafe.Pointer(&src[0])), len(src)*2))
	runtime.KeepAlive(src)
}

func copyFloat32SliceToTemporaryArrayBuffer(src []float32) {
	if len(src) == 0 {
		return
	}
	js.CopyBytesToJS(temporaryUint8Array, unsafe.Slice((*byte)(unsafe.Pointer(&src[0])), len(src)*4))
	runtime.KeepAlive(src)
}
