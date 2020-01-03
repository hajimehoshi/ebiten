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

var theDriver Driver

func Get() *Driver {
	return &theDriver
}

type Driver struct {
	state   openGLState
	context context

	// drawCalled is true just after Draw is called. This holds true until ReplacePixels is called.
	drawCalled bool
}

func (d *Driver) SetThread(thread *thread.Thread) {
	d.context.t = thread
}

func (d *Driver) Begin() {
	// Do nothing.
}

func (d *Driver) End() {
	// Call glFlush to prevent black flicking (especially on Android (#226) and iOS).
	// TODO: examples/sprites worked without this. Is this really needed?
	d.context.flush()
}

func (d *Driver) SetTransparent(transparent bool) {
	// Do nothings.
}

func (d *Driver) checkSize(width, height int) {
	if width < 1 {
		panic(fmt.Sprintf("opengl: width (%d) must be equal or more than %d", width, 1))
	}
	if height < 1 {
		panic(fmt.Sprintf("opengl: height (%d) must be equal or more than %d", height, 1))
	}
	m := d.context.getMaxTextureSize()
	if width > m {
		panic(fmt.Sprintf("opengl: width (%d) must be less than or equal to %d", width, m))
	}
	if height > m {
		panic(fmt.Sprintf("opengl: height (%d) must be less than or equal to %d", height, m))
	}
}

func (d *Driver) NewImage(width, height int) (driver.Image, error) {
	i := &Image{
		driver: d,
		width:  width,
		height: height,
	}
	w := graphics.InternalImageSize(width)
	h := graphics.InternalImageSize(height)
	d.checkSize(w, h)
	t, err := d.context.newTexture(w, h)
	if err != nil {
		return nil, err
	}
	i.textureNative = t
	return i, nil
}

func (d *Driver) NewScreenFramebufferImage(width, height int) (driver.Image, error) {
	d.checkSize(width, height)
	i := &Image{
		driver: d,
		width:  width,
		height: height,
		screen: true,
	}
	return i, nil
}

// Reset resets or initializes the current OpenGL state.
func (d *Driver) Reset() error {
	return d.state.reset(&d.context)
}

func (d *Driver) SetVertices(vertices []float32, indices []uint16) {
	// Note that the vertices passed to BufferSubData is not under GC management
	// in opengl package due to unsafe-way.
	// See BufferSubData in context_mobile.go.
	d.context.arrayBufferSubData(vertices)
	d.context.elementArrayBufferSubData(indices)
}

func (d *Driver) Draw(indexLen int, indexOffset int, mode driver.CompositeMode, colorM *affine.ColorM, filter driver.Filter, address driver.Address) error {
	d.drawCalled = true
	if err := d.useProgram(mode, colorM, filter, address); err != nil {
		return err
	}
	d.context.drawElements(indexLen, indexOffset*2) // 2 is uint16 size in bytes
	// glFlush() might be necessary at least on MacBook Pro (a smilar problem at #419),
	// but basically this pass the tests (esp. TestImageTooManyFill).
	// As glFlush() causes performance problems, this should be avoided as much as possible.
	// Let's wait and see, and file a new issue when this problem is newly found.
	return nil
}

func (d *Driver) SetVsyncEnabled(enabled bool) {
	// Do nothing
}

func (d *Driver) VDirection() driver.VDirection {
	return driver.VDownward
}

func (d *Driver) NeedsRestoring() bool {
	return d.context.needsRestoring()
}

func (d *Driver) IsGL() bool {
	return true
}

func (d *Driver) HasHighPrecisionFloat() bool {
	return d.context.hasHighPrecisionFloat()
}

func (d *Driver) MaxImageSize() int {
	return d.context.getMaxTextureSize()
}
