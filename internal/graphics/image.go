// Copyright 2016 Hajime Hoshi
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
	"github.com/hajimehoshi/ebiten/internal/affine"
	"github.com/hajimehoshi/ebiten/internal/math"
	"github.com/hajimehoshi/ebiten/internal/opengl"
)

var (
	// maxTextureSize is the maximum texture size
	//
	// maxTextureSize also represents the default size (width or height) of viewport.
	maxTextureSize = 0
)

// MaxImageSize returns the maximum of width/height of an image.
func MaxImageSize() int {
	if maxTextureSize == 0 {
		maxTextureSize = opengl.GetContext().MaxTextureSize()
		if maxTextureSize == 0 {
			panic("graphics: failed to get the max texture size")
		}
	}
	s := maxTextureSize
	return s
}

// Image represents an image that is implemented with OpenGL.
type Image struct {
	texture     *texture
	framebuffer *framebuffer
	width       int
	height      int
}

func NewImage(width, height int) *Image {
	i := &Image{
		width:  width,
		height: height,
	}
	c := &newImageCommand{
		result: i,
		width:  width,
		height: height,
	}
	theCommandQueue.Enqueue(c)
	return i
}

func NewScreenFramebufferImage(width, height int) *Image {
	i := &Image{
		width:  width,
		height: height,
	}
	c := &newScreenFramebufferImageCommand{
		result: i,
		width:  width,
		height: height,
	}
	theCommandQueue.Enqueue(c)
	return i
}

func (i *Image) Dispose() {
	c := &disposeCommand{
		target: i,
	}
	theCommandQueue.Enqueue(c)
}

func (i *Image) Size() (int, int) {
	return i.width, i.height
}

func (i *Image) DrawImage(src *Image, vertices []float32, clr *affine.ColorM, mode opengl.CompositeMode, filter Filter) {
	theCommandQueue.EnqueueDrawImageCommand(i, src, vertices, clr, mode, filter)
}

func (i *Image) Pixels() ([]byte, error) {
	// Flush the enqueued commands so that pixels are certainly read.
	if err := theCommandQueue.Flush(); err != nil {
		return nil, err
	}
	f, err := i.createFramebufferIfNeeded()
	if err != nil {
		return nil, err
	}
	return opengl.GetContext().FramebufferPixels(f.native, i.width, i.height)
}

func (i *Image) ReplacePixels(p []byte, x, y, width, height int) {
	pixels := make([]byte, len(p))
	copy(pixels, p)
	c := &replacePixelsCommand{
		dst:    i,
		pixels: pixels,
		x:      x,
		y:      y,
		width:  width,
		height: height,
	}
	theCommandQueue.Enqueue(c)
}

func (i *Image) IsInvalidated() bool {
	return !opengl.GetContext().IsTexture(i.texture.native)
}

func (i *Image) createFramebufferIfNeeded() (*framebuffer, error) {
	if i.framebuffer != nil {
		return i.framebuffer, nil
	}
	f, err := newFramebufferFromTexture(i.texture, math.NextPowerOf2Int(i.width), math.NextPowerOf2Int(i.height))
	if err != nil {
		return nil, err
	}
	i.framebuffer = f
	return i.framebuffer, nil
}
