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

// +build darwin freebsd linux windows
// +build !js
// +build !android
// +build !ios

package opengl

import (
	"reflect"
	"unsafe"

	"github.com/hajimehoshi/ebiten/internal/driver"
)

const canUsePBO = true

func drawPixelsWithPBO(img *Image, args []*driver.ReplacePixelsArgs) {
	w, h := img.width, img.height
	if img.pbo == *new(buffer) {
		img.pbo = img.driver.context.newPixelBufferObject(w, h)
	}
	if img.pbo == *new(buffer) {
		panic("opengl: newPixelBufferObject failed")
	}

	mappedPBO := img.driver.context.mapPixelBuffer(img.pbo)
	if mappedPBO == 0 {
		panic("opengl: mapPixelBuffer failed")
	}

	var mapped []byte
	sh := (*reflect.SliceHeader)(unsafe.Pointer(&mapped))
	sh.Data = mappedPBO
	sh.Len = 4 * w * h
	sh.Cap = 4 * w * h

	for _, a := range args {
		stride := 4 * w
		offset := 4 * (a.Y*w + a.X)
		for j := 0; j < a.Height; j++ {
			copy(mapped[offset+stride*j:offset+stride*j+4*a.Width], a.Pixels[4*a.Width*j:4*a.Width*(j+1)])
		}
	}
	img.driver.context.unmapPixelBuffer(img.pbo, img.textureNative, w, h)
}
