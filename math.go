// Copyright 2016 The Ebiten Authors
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

// +build !js

package ebiten

import (
	"unsafe"

	"github.com/hajimehoshi/ebiten/internal/endian"
)

func floatBytes(xs ...float64) []uint8 {
	bits := make([]uint8, 0, len(xs)*4)
	for _, x := range xs {
		x32 := float32(x)
		n := *(*uint32)(unsafe.Pointer(&x32))
		if endian.IsLittle() {
			bits = append(bits,
				uint8(n),
				uint8(n>>8),
				uint8(n>>16),
				uint8(n>>24))
		} else {
			bits = append(bits,
				uint8(n>>24),
				uint8(n>>16),
				uint8(n>>8),
				uint8(n))
		}
	}
	return bits
}
