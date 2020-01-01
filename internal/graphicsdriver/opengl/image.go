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
	"github.com/hajimehoshi/ebiten/internal/driver"
	"github.com/hajimehoshi/ebiten/internal/graphics"
)

type Image struct {
	driver        *Driver
	textureNative textureNative
	framebuffer   *framebuffer
	pbo           buffer
	width         int
	height        int
	screen        bool
}

func (i *Image) IsInvalidated() bool {
	return !i.driver.context.isTexture(i.textureNative)
}

func (i *Image) Dispose() {
	if !i.pbo.equal(*new(buffer)) {
		i.driver.context.deleteBuffer(i.pbo)
	}
	if i.framebuffer != nil {
		i.framebuffer.delete(&i.driver.context)
	}
	if !i.textureNative.equal(*new(textureNative)) {
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

func (i *Image) ensureFramebuffer() error {
	if i.framebuffer != nil {
		return nil
	}

	if i.screen {
		// The (default) framebuffer size can't be converted to a power of 2.
		// On browsers, c.width and c.height are used as viewport size and
		// Edge can't treat a bigger viewport than the drawing area (#71).
		i.framebuffer = newScreenFramebuffer(&i.driver.context, i.width, i.height)
		return nil
	}

	w, h := graphics.InternalImageSize(i.width), graphics.InternalImageSize(i.height)
	f, err := newFramebufferFromTexture(&i.driver.context, i.textureNative, w, h)
	if err != nil {
		return err
	}
	i.framebuffer = f
	return nil
}

func (i *Image) ReplacePixels(args []*driver.ReplacePixelsArgs) {
	if i.screen {
		panic("opengl: ReplacePixels cannot be called on the screen, that doesn't have a texture")
	}
	if len(args) == 0 {
		return
	}

	// glFlush is necessary on Android.
	// glTexSubImage2D didn't work without this hack at least on Nexus 5x and NuAns NEO [Reloaded] (#211).
	if i.driver.drawCalled {
		i.driver.context.flush()
	}
	i.driver.drawCalled = false

	w, h := i.width, i.height
	if !i.driver.context.canUsePBO() {
		i.driver.context.texSubImage2D(i.textureNative, w, h, args)
		return
	}
	if i.pbo.equal(*new(buffer)) {
		i.pbo = i.driver.context.newPixelBufferObject(w, h)
	}
	if i.pbo.equal(*new(buffer)) {
		panic("opengl: newPixelBufferObject failed")
	}

	i.driver.context.replacePixelsWithPBO(i.pbo, i.textureNative, w, h, args)
}

func (i *Image) SetAsSource() {
	i.driver.state.source = i
}
