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
	"errors"

	"github.com/hajimehoshi/ebiten/v2/internal/graphics"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/opengl/gl"
)

type Image struct {
	id          graphicsdriver.ImageID
	graphics    *Graphics
	texture     textureNative
	stencil     renderbufferNative
	framebuffer *framebuffer
	width       int
	height      int
	screen      bool
}

// framebuffer is a wrapper of OpenGL's framebuffer.
type framebuffer struct {
	graphics *Graphics
	native   framebufferNative
	width    int
	height   int
}

func (i *Image) ID() graphicsdriver.ImageID {
	return i.id
}

func (i *Image) IsInvalidated() bool {
	return !i.graphics.context.ctx.IsTexture(uint32(i.texture))
}

func (i *Image) Dispose() {
	if i.framebuffer != nil {
		i.graphics.context.deleteFramebuffer(i.framebuffer.native)
	}
	if i.texture != 0 {
		i.graphics.context.deleteTexture(i.texture)
	}
	if i.stencil != 0 {
		i.graphics.context.deleteRenderbuffer(i.stencil)
	}

	i.graphics.removeImage(i)
}

func (i *Image) setViewport() error {
	if err := i.ensureFramebuffer(); err != nil {
		return err
	}
	i.graphics.context.setViewport(i.framebuffer)
	return nil
}

func (i *Image) ReadPixels(buf []byte, x, y, width, height int) error {
	if err := i.ensureFramebuffer(); err != nil {
		return err
	}
	if err := i.graphics.context.framebufferPixels(buf, i.framebuffer, x, y, width, height); err != nil {
		return err
	}
	return nil
}

func (i *Image) framebufferSize() (int, int) {
	if i.screen {
		// The (default) framebuffer size can't be converted to a power of 2.
		// On browsers, i.width and i.height are used as viewport size and
		// Edge can't treat a bigger viewport than the drawing area (#71).
		return i.width, i.height
	}
	return graphics.InternalImageSize(i.width), graphics.InternalImageSize(i.height)
}

func (i *Image) ensureFramebuffer() error {
	if i.framebuffer != nil {
		return nil
	}

	w, h := i.framebufferSize()
	if i.screen {
		i.framebuffer = i.graphics.context.newScreenFramebuffer(w, h)
		return nil
	}
	f, err := i.graphics.context.newFramebuffer(i.texture, w, h)
	if err != nil {
		return err
	}
	i.framebuffer = f
	return nil
}

func (i *Image) ensureStencilBuffer() error {
	if i.stencil != 0 {
		return nil
	}

	if err := i.ensureFramebuffer(); err != nil {
		return err
	}

	r, err := i.graphics.context.newRenderbuffer(i.framebufferSize())
	if err != nil {
		return err
	}
	i.stencil = r

	if err := i.graphics.context.bindStencilBuffer(i.framebuffer.native, i.stencil); err != nil {
		return err
	}
	return nil
}

func (i *Image) WritePixels(args []*graphicsdriver.WritePixelsArgs) error {
	if i.screen {
		return errors.New("opengl: WritePixels cannot be called on the screen")
	}
	if len(args) == 0 {
		return nil
	}

	// glFlush is necessary on Android.
	// glTexSubImage2D didn't work without this hack at least on Nexus 5x and NuAns NEO [Reloaded] (#211).
	if i.graphics.drawCalled {
		i.graphics.context.ctx.Flush()
	}
	i.graphics.drawCalled = false

	i.graphics.context.bindTexture(i.texture)
	for _, a := range args {
		i.graphics.context.ctx.TexSubImage2D(gl.TEXTURE_2D, 0, int32(a.X), int32(a.Y), int32(a.Width), int32(a.Height), gl.RGBA, gl.UNSIGNED_BYTE, a.Pixels)
	}

	return nil
}
