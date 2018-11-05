// Copyright 2018 The Ebiten Authors
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

package opengl

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/internal/math"
)

func checkSize(width, height int) {
	if width < 1 {
		panic(fmt.Sprintf("opengl: width (%d) must be equal or more than 1.", width))
	}
	if height < 1 {
		panic(fmt.Sprintf("opengl: height (%d) must be equal or more than 1.", height))
	}
	m := theContext.getMaxTextureSize()
	if width > m {
		panic(fmt.Sprintf("opengl: width (%d) must be less than or equal to %d", width, m))
	}
	if height > m {
		panic(fmt.Sprintf("opengl: height (%d) must be less than or equal to %d", height, m))
	}
}

type Image struct {
	textureNative textureNative
	framebuffer   *framebuffer
	width         int
	height        int
}

func (i *Image) IsInvalidated() bool {
	return !theContext.isTexture(i.textureNative)
}

func (i *Image) Delete() {
	if i.framebuffer != nil {
		i.framebuffer.delete()
	}
	if i.textureNative != *new(textureNative) {
		theContext.deleteTexture(i.textureNative)
	}
}

func (i *Image) SetAsDestination() {
	theOpenGLState.destination = i
}

func (i *Image) setViewport() error {
	if err := i.ensureFramebuffer(); err != nil {
		return err
	}
	theContext.setViewport(i.framebuffer)
	return nil
}

func (i *Image) Pixels() ([]byte, error) {
	if err := i.ensureFramebuffer(); err != nil {
		return nil, err
	}
	p, err := theContext.framebufferPixels(i.framebuffer, i.width, i.height)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (i *Image) projectionMatrix() []float32 {
	if i.framebuffer == nil {
		panic("not reached")
	}
	return i.framebuffer.projectionMatrix()
}

func (i *Image) ensureFramebuffer() error {
	if i.framebuffer != nil {
		return nil
	}
	w, h := i.width, i.height
	f, err := newFramebufferFromTexture(i.textureNative, math.NextPowerOf2Int(w), math.NextPowerOf2Int(h))
	if err != nil {
		return err
	}
	i.framebuffer = f
	return nil
}

func (i *Image) TexSubImage2D(p []byte, x, y, width, height int) {
	theContext.texSubImage2D(i.textureNative, p, x, y, width, height)
}

func (i *Image) SetAsSource() {
	theOpenGLState.source = i
}
