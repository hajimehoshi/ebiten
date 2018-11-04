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

package graphicscommand

import (
	"github.com/hajimehoshi/ebiten/internal/affine"
	"github.com/hajimehoshi/ebiten/internal/graphics"
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
	image *opengl.Image
}

func NewImage(width, height int) *Image {
	i := &Image{
		image: opengl.NewImage(width, height),
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
		image: opengl.NewImage(width, height),
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
	return i.image.Size()
}

func (i *Image) DrawImage(src *Image, vertices []float32, indices []uint16, clr *affine.ColorM, mode graphics.CompositeMode, filter graphics.Filter) {
	theCommandQueue.EnqueueDrawImageCommand(i, src, vertices, indices, clr, mode, filter)
}

// Pixels returns the image's pixels.
// Pixels might return nil when OpenGL error happens.
func (i *Image) Pixels() []byte {
	c := &pixelsCommand{
		result: nil,
		img:    i,
	}
	theCommandQueue.Enqueue(c)
	theCommandQueue.Flush()
	return c.result
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
	return i.image.IsInvalidated()
}

func (i *Image) ensureFramebuffer() (*opengl.Framebuffer, error) {
	if i.image.Framebuffer != nil {
		return i.image.Framebuffer, nil
	}
	w, h := i.image.Size()
	f, err := opengl.NewFramebufferFromTexture(i.image.Texture, math.NextPowerOf2Int(w), math.NextPowerOf2Int(h))
	if err != nil {
		return nil, err
	}
	i.image.Framebuffer = f
	return i.image.Framebuffer, nil
}
