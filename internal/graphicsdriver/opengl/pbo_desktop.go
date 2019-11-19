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

	"github.com/hajimehoshi/ebiten/internal/graphics"
)

const canUsePBO = true

type pboState struct {
	image     *Image
	mappedPBO uintptr
	mapped    []byte
}

var thePBOState pboState

func (s *pboState) mapPBO(img *Image) {
	w, h := graphics.InternalImageSize(img.width), graphics.InternalImageSize(img.height)
	if img.pbo == *new(buffer) {
		img.pbo = img.driver.context.newPixelBufferObject(w, h)
	}
	s.image = img
	s.mappedPBO = img.driver.context.mapPixelBuffer(img.pbo, img.textureNative)

	if s.mappedPBO == 0 {
		panic("opengl: mapPixelBuffer failed")
	}

	var mapped []byte
	sh := (*reflect.SliceHeader)(unsafe.Pointer(&mapped))
	sh.Data = s.mappedPBO
	sh.Len = 4 * w * h
	sh.Cap = 4 * w * h
	s.mapped = mapped
}

func (s *pboState) draw(pix []byte, x, y, width, height int) {
	w := graphics.InternalImageSize(s.image.width)
	stride := 4 * w
	offset := 4 * (y*w + x)
	for j := 0; j < height; j++ {
		copy(s.mapped[offset+stride*j:offset+stride*j+4*width], pix[4*width*j:4*width*(j+1)])
	}
}

func (s *pboState) unmapPBO() {
	i := s.image
	w, h := graphics.InternalImageSize(i.width), graphics.InternalImageSize(i.height)
	i.driver.context.unmapPixelBuffer(i.pbo, w, h)

	s.image = nil
	s.mappedPBO = 0
	s.mapped = nil
}
