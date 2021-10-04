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
	"fmt"
	"syscall/js"
)

func Uint8ArrayToSlice(value js.Value, length int) []byte {
	if l := value.Get("byteLength").Int(); length > l {
		length = l
	}
	s := make([]byte, length)
	js.CopyBytesToGo(s, value)
	return s
}

func copySliceToTemporaryArrayBuffer(src interface{}) {
	switch s := src.(type) {
	case []uint8:
		js.CopyBytesToJS(temporaryUint8Array, s)
	case []int8, []int16, []int32, []uint16, []uint32, []float32, []float64:
		js.CopyBytesToJS(temporaryUint8Array, sliceToByteSlice(s))
	default:
		panic(fmt.Sprintf("jsutil: unexpected value at CopySliceToJS: %T", s))
	}
}
