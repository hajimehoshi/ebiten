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

	img.driver.context.replacePixelsWithPBO(img.pbo, img.textureNative, w, h, args)
}
