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
	"github.com/hajimehoshi/ebiten/internal/affine"
	"github.com/hajimehoshi/ebiten/internal/graphics"
	"github.com/hajimehoshi/ebiten/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/internal/math"
)

var theDriver Driver

func GetDriver() *Driver {
	return &theDriver
}

type Driver struct {
	state openGLState
}

func (d *Driver) NewImage(width, height int) (graphicsdriver.Image, error) {
	i := &Image{
		driver: d,
		width:  width,
		height: height,
	}
	w := math.NextPowerOf2Int(width)
	h := math.NextPowerOf2Int(height)
	checkSize(w, h)
	t, err := theContext.newTexture(w, h)
	if err != nil {
		return nil, err
	}
	i.textureNative = t
	return i, nil
}

func (d *Driver) NewScreenFramebufferImage(width, height int) graphicsdriver.Image {
	checkSize(width, height)
	i := &Image{
		driver: d,
		width:  width,
		height: height,
	}
	// The (default) framebuffer size can't be converted to a power of 2.
	// On browsers, c.width and c.height are used as viewport size and
	// Edge can't treat a bigger viewport than the drawing area (#71).
	i.framebuffer = newScreenFramebuffer(width, height)
	return i
}

// Reset resets or initializes the current OpenGL state.
func (d *Driver) Reset() error {
	return d.state.reset()
}

func (d *Driver) BufferSubData(vertices []float32, indices []uint16) {
	bufferSubData(vertices, indices)
}

func (d *Driver) UseProgram(mode graphics.CompositeMode, colorM *affine.ColorM, filter graphics.Filter) error {
	return d.state.useProgram(mode, colorM, filter)
}

func (d *Driver) DrawElements(len int, offsetInBytes int) {
	theContext.drawElements(len, offsetInBytes)
	// glFlush() might be necessary at least on MacBook Pro (a smilar problem at #419),
	// but basically this pass the tests (esp. TestImageTooManyFill).
	// As glFlush() causes performance problems, this should be avoided as much as possible.
	// Let's wait and see, and file a new issue when this problem is newly found.
}

func (d *Driver) Flush() {
	theContext.flush()
}

func (d *Driver) MaxImageSize() int {
	return theContext.getMaxTextureSize()
}
