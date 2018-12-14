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
	"github.com/hajimehoshi/ebiten/internal/graphicsdriver"
)

var (
	// maxImageSize is the maximum texture size
	//
	// maxImageSize also represents the default size (width or height) of viewport.
	maxImageSize = 0
)

// imageState is a state of an image.
type imageState int

const (
	// imageStateInit represents that the image is just allocated and not ready for DrawImages.
	imageStateInit imageState = iota

	// imageStateReplacePixelsOnly represents that only ReplacePixels is acceptable.
	imageStateReplacePixelsOnly

	// imageStateDrawable represents that the image is ready to draw with any commands.
	imageStateDrawable

	// imageStateScreen is the special state for screen framebuffer.
	// Only copying image on the screen image is allowed.
	imageStateScreen
)

// Image represents an image that is implemented with OpenGL.
type Image struct {
	image  graphicsdriver.Image
	width  int
	height int
	state  imageState
}

func NewImage(width, height int) *Image {
	i := &Image{
		width:  width,
		height: height,
		state:  imageStateInit, // The screen image must be inited with ReplacePixels first.
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
		state:  imageStateScreen,
	}
	c := &newScreenFramebufferImageCommand{
		result: i,
		width:  width,
		height: height,
	}
	theCommandQueue.Enqueue(c)
	return i
}

// clearByReplacingPixels clears the image by replacing pixels.
//
// The implementation must use replacing-pixels way instead of drawing polygons, since
// some environments (e.g. Metal) require replacing-pixels way as initialization.
func (i *Image) clearByReplacingPixels() {
	if i.state != imageStateInit {
		panic("not reached")
	}
	c := &replacePixelsCommand{
		dst:    i,
		pixels: make([]byte, 4*i.width*i.height),
		x:      0,
		y:      0,
		width:  i.width,
		height: i.height,
	}
	theCommandQueue.Enqueue(c)
	i.state = imageStateDrawable
}

func (i *Image) Dispose() {
	c := &disposeCommand{
		target: i,
	}
	theCommandQueue.Enqueue(c)
}

func (i *Image) Size() (int, int) {
	// i.image can be nil before initializing.
	return i.width, i.height
}

func (i *Image) DrawImage(src *Image, vertices []float32, indices []uint16, clr *affine.ColorM, mode graphics.CompositeMode, filter graphics.Filter) {
	switch i.state {
	case imageStateInit:
		// Before DrawImage, the image must be initialized with ReplacePixels.
		// Especially on Metal, the image might be broken when drawing without initializing.
		i.clearByReplacingPixels()
	case imageStateReplacePixelsOnly:
		panic("not reached")
	case imageStateDrawable:
		// Do nothing
	case imageStateScreen:
		if mode != graphics.CompositeModeCopy {
			panic("not reached")
		}
	default:
		panic("not reached")
	}

	switch src.state {
	case imageStateInit:
		src.clearByReplacingPixels()
	case imageStateReplacePixelsOnly:
		// Do nothing
		// TODO: Check the region.
	case imageStateDrawable:
		// Do nothing
	case imageStateScreen:
		panic("not reached")
	default:
		panic("not reached")
	}

	theCommandQueue.EnqueueDrawImageCommand(i, src, vertices, indices, clr, mode, filter)
}

// Pixels returns the image's pixels.
// Pixels might return nil when OpenGL error happens.
func (i *Image) Pixels() []byte {
	switch i.state {
	case imageStateInit:
		i.clearByReplacingPixels()
	case imageStateReplacePixelsOnly:
		// Do nothing
		// TODO: Check the region?
	case imageStateDrawable:
		// Do nothing
	case imageStateScreen:
		panic("not reached")
	default:
		panic("not reached")
	}
	c := &pixelsCommand{
		result: nil,
		img:    i,
	}
	theCommandQueue.Enqueue(c)
	theCommandQueue.Flush()
	return c.result
}

func (i *Image) ReplacePixels(p []byte, x, y, width, height int) {
	switch i.state {
	case imageStateInit:
		if x == 0 && y == 0 && width == i.width && height == i.height {
			i.state = imageStateDrawable
		} else {
			i.state = imageStateReplacePixelsOnly
		}
	case imageStateReplacePixelsOnly:
		// Do nothing
	case imageStateDrawable:
		// Do nothing
	case imageStateScreen:
		panic("not reached")
	default:
		panic("not reached")
	}
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
	// i.image can be nil before initializing.
	if i.image == nil {
		return false
	}
	return i.image.IsInvalidated()
}
