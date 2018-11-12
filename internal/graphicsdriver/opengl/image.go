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
	"github.com/hajimehoshi/ebiten/internal/math"
)

type Image struct {
	driver        *Driver
	textureNative textureNative
	framebuffer   *framebuffer
	width         int
	height        int
}

func (i *Image) IsInvalidated() bool {
	return !i.driver.context.isTexture(i.textureNative)
}

func (i *Image) Dispose() {
	if i.framebuffer != nil {
		i.framebuffer.delete(&i.driver.context)
	}
	if i.textureNative != *new(textureNative) {
		i.driver.context.deleteTexture(i.textureNative)
	}
}

func (i *Image) SetAsDestination() {
	i.driver.state.destination = i
}

func (i *Image) setViewport() error {
	if err := i.ensureFramebuffer(); err != nil {
		return err
	}
	i.driver.context.setViewport(i.framebuffer)
	return nil
}

func (i *Image) Pixels() ([]byte, error) {
	if err := i.ensureFramebuffer(); err != nil {
		return nil, err
	}
	p, err := i.driver.context.framebufferPixels(i.framebuffer, i.width, i.height)
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
	f, err := newFramebufferFromTexture(&i.driver.context, i.textureNative, math.NextPowerOf2Int(w), math.NextPowerOf2Int(h))
	if err != nil {
		return err
	}
	i.framebuffer = f
	return nil
}

func (i *Image) ReplacePixels(p []byte, x, y, width, height int) {
	// glFlush is necessary on Android.
	// glTexSubImage2D didn't work without this hack at least on Nexus 5x and NuAns NEO [Reloaded] (#211).
	i.driver.context.flush()
	i.driver.context.texSubImage2D(i.textureNative, p, x, y, width, height)
}

func (i *Image) SetAsSource() {
	i.driver.state.source = i
}
