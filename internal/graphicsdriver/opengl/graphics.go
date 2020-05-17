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

	"github.com/hajimehoshi/ebiten/internal/affine"
	"github.com/hajimehoshi/ebiten/internal/driver"
	"github.com/hajimehoshi/ebiten/internal/graphics"
	"github.com/hajimehoshi/ebiten/internal/thread"
)

var theGraphics Graphics

func Get() *Graphics {
	return &theGraphics
}

type Graphics struct {
	state   openGLState
	context context

	// drawCalled is true just after Draw is called. This holds true until ReplacePixels is called.
	drawCalled bool
}

func (g *Graphics) SetThread(thread *thread.Thread) {
	g.context.t = thread
}

func (g *Graphics) Begin() {
	// Do nothing.
}

func (g *Graphics) End() {
	// Call glFlush to prevent black flicking (especially on Android (#226) and iOS).
	// TODO: examples/sprites worked without this. Is this really needed?
	g.context.flush()
}

func (g *Graphics) SetTransparent(transparent bool) {
	// Do nothings.
}

func (g *Graphics) checkSize(width, height int) {
	if width < 1 {
		panic(fmt.Sprintf("opengl: width (%d) must be equal or more than %d", width, 1))
	}
	if height < 1 {
		panic(fmt.Sprintf("opengl: height (%d) must be equal or more than %d", height, 1))
	}
	m := g.context.getMaxTextureSize()
	if width > m {
		panic(fmt.Sprintf("opengl: width (%d) must be less than or equal to %d", width, m))
	}
	if height > m {
		panic(fmt.Sprintf("opengl: height (%d) must be less than or equal to %d", height, m))
	}
}

func (g *Graphics) NewImage(width, height int) (driver.Image, error) {
	i := &Image{
		graphics: g,
		width:    width,
		height:   height,
	}
	w := graphics.InternalImageSize(width)
	h := graphics.InternalImageSize(height)
	g.checkSize(w, h)
	t, err := g.context.newTexture(w, h)
	if err != nil {
		return nil, err
	}
	i.textureNative = t
	return i, nil
}

func (g *Graphics) NewScreenFramebufferImage(width, height int) (driver.Image, error) {
	g.checkSize(width, height)
	i := &Image{
		graphics: g,
		width:    width,
		height:   height,
		screen:   true,
	}
	return i, nil
}

// Reset resets or initializes the current OpenGL state.
func (g *Graphics) Reset() error {
	return g.state.reset(&g.context)
}

func (g *Graphics) SetVertices(vertices []float32, indices []uint16) {
	// Note that the vertices passed to BufferSubData is not under GC management
	// in opengl package due to unsafe-way.
	// See BufferSubData in context_mobile.go.
	g.context.arrayBufferSubData(vertices)
	g.context.elementArrayBufferSubData(indices)
}

func (g *Graphics) Draw(indexLen int, indexOffset int, mode driver.CompositeMode, colorM *affine.ColorM, filter driver.Filter, address driver.Address) error {
	g.drawCalled = true

	g.context.blendFunc(mode)

	if err := g.useProgram(colorM, filter, address); err != nil {
		return err
	}
	g.context.drawElements(indexLen, indexOffset*2) // 2 is uint16 size in bytes
	// glFlush() might be necessary at least on MacBook Pro (a smilar problem at #419),
	// but basically this pass the tests (esp. TestImageTooManyFill).
	// As glFlush() causes performance problems, this should be avoided as much as possible.
	// Let's wait and see, and file a new issue when this problem is newly foung.
	return nil
}

func (g *Graphics) SetVsyncEnabled(enabled bool) {
	// Do nothing
}

func (g *Graphics) FramebufferYDirection() driver.YDirection {
	return driver.Upward
}

func (g *Graphics) NeedsRestoring() bool {
	return g.context.needsRestoring()
}

func (g *Graphics) IsGL() bool {
	return true
}

func (g *Graphics) HasHighPrecisionFloat() bool {
	return g.context.hasHighPrecisionFloat()
}

func (g *Graphics) MaxImageSize() int {
	return g.context.getMaxTextureSize()
}
