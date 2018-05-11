// Copyright 2014 Hajime Hoshi
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

package graphics

import (
	"github.com/hajimehoshi/ebiten/internal/opengl"
	"github.com/hajimehoshi/ebiten/internal/web"
)

// orthoProjectionMatrix returns an orthogonal projection matrix for OpenGL.
//
// The matrix converts the coodinates (left, bottom) - (right, top) to the normalized device coodinates (-1, -1) - (1, 1).
func orthoProjectionMatrix(left, right, bottom, top int) []float32 {
	e11 := 2 / float32(right-left)
	e22 := 2 / float32(top-bottom)
	e14 := -1 * float32(right+left) / float32(right-left)
	e24 := -1 * float32(top+bottom) / float32(top-bottom)

	return []float32{
		e11, 0, 0, 0,
		0, e22, 0, 0,
		0, 0, 1, 0,
		e14, e24, 0, 1,
	}
}

// framebuffer is a wrapper of OpenGL's framebuffer.
type framebuffer struct {
	native    opengl.Framebuffer
	proMatrix []float32
	width     int
	height    int
}

// newFramebufferFromTexture creates a framebuffer from the given texture.
func newFramebufferFromTexture(texture *texture, width, height int) (*framebuffer, error) {
	native, err := opengl.GetContext().NewFramebuffer(opengl.Texture(texture.native))
	if err != nil {
		return nil, err
	}
	return &framebuffer{
		native: native,
		width:  width,
		height: height,
	}, nil
}

// newScreenFramebuffer creates a framebuffer for the screen.
func newScreenFramebuffer(width, height int) *framebuffer {
	return &framebuffer{
		native: opengl.GetContext().ScreenFramebuffer(),
		width:  width,
		height: height,
	}
}

// viewportSize returns the viewport size of the framebuffer.
func (f *framebuffer) viewportSize() (int, int) {
	// On some browsers, viewport size must be within the framebuffer size.
	// e.g. Edge (#71), Chrome on GPD Pocket (#420)
	if web.IsBrowser() {
		return f.width, f.height
	}

	// If possible, always use the same viewport size to reduce draw calls.
	m := MaxImageSize()
	return m, m
}

// setAsViewport sets the framebuffer as the current viewport.
func (f *framebuffer) setAsViewport() {
	w, h := f.viewportSize()
	opengl.GetContext().SetViewport(f.native, w, h)
}

// projectionMatrix returns a projection matrix of the framebuffer.
//
// A projection matrix converts the coodinates on the framebuffer
// (0, 0) - (viewport width, viewport height)
// to the normalized device coodinates (-1, -1) - (1, 1) with adjustment.
func (f *framebuffer) projectionMatrix() []float32 {
	if f.proMatrix != nil {
		return f.proMatrix
	}
	w, h := f.viewportSize()
	f.proMatrix = orthoProjectionMatrix(0, w, 0, h)
	return f.proMatrix
}
