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
	"fmt"
	"image"
	"os"
	"strings"

	"github.com/hajimehoshi/ebiten/internal/affine"
	"github.com/hajimehoshi/ebiten/internal/driver"
	"github.com/hajimehoshi/ebiten/internal/graphics"
	"github.com/hajimehoshi/ebiten/internal/png"
)

type lastCommand int

const (
	lastCommandNone lastCommand = iota
	lastCommandClear
	lastCommandDrawTriangles
	lastCommandReplacePixels
)

// Image represents an image that is implemented with OpenGL.
type Image struct {
	image          driver.Image
	width          int
	height         int
	internalWidth  int
	internalHeight int
	screen         bool
	id             int

	lastCommand lastCommand
}

var nextID = 1

func genNextID() int {
	id := nextID
	nextID++
	return id
}

// NewImage returns a new image.
//
// Note that the image is not initialized yet.
func NewImage(width, height int) *Image {
	i := &Image{
		width:          width,
		height:         height,
		internalWidth:  graphics.InternalImageSize(width),
		internalHeight: graphics.InternalImageSize(height),
		id:             genNextID(),
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
		width:          width,
		height:         height,
		internalWidth:  graphics.InternalImageSize(width),
		internalHeight: graphics.InternalImageSize(height),
		screen:         true,
		id:             genNextID(),
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
	// i.image can be nil before initializing.
	return i.width, i.height
}

func (i *Image) InternalSize() (int, int) {
	return i.internalWidth, i.internalHeight
}

func (i *Image) DrawTriangles(src *Image, vertices []float32, indices []uint16, clr *affine.ColorM, mode driver.CompositeMode, filter driver.Filter, address driver.Address) {
	if src.screen {
		panic("graphicscommand: the screen image cannot be the rendering source")
	}

	if i.lastCommand == lastCommandNone {
		if !i.screen && mode != driver.CompositeModeClear {
			panic("graphicscommand: the image must be cleared first")
		}
	}

	theCommandQueue.EnqueueDrawTrianglesCommand(i, src, vertices, indices, clr, mode, filter, address)

	if i.lastCommand == lastCommandNone && !i.screen {
		i.lastCommand = lastCommandClear
	} else {
		i.lastCommand = lastCommandDrawTriangles
	}
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
	// ReplacePixels for a part might invalidate the current image that are drawn by DrawTriangles (#593, #738).
	if i.lastCommand == lastCommandDrawTriangles {
		if x != 0 || y != 0 || i.width != width || i.height != height {
			panic("graphicscommand: ReplacePixels for a part after DrawTriangles is forbidden")
		}
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
	i.lastCommand = lastCommandReplacePixels
}

func (i *Image) IsInvalidated() bool {
	if i.screen {
		// The screen image might not have a texture, and in this case it is impossible to detect whether
		// the image is invalidated or not.
		panic("graphicscommand: IsInvalidated cannot be called on the screen image")
	}

	// i.image can be nil before initializing.
	if i.image == nil {
		return false
	}
	return i.image.IsInvalidated()
}

// Dump dumps the image to the specified path.
// In the path, '*' is replaced with the image's ID.
//
// This is for testing usage.
func (i *Image) Dump(path string) error {
	path = strings.ReplaceAll(path, "*", fmt.Sprintf("%d", i.id))
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	if err := png.Encode(f, &image.RGBA{
		Pix:    i.Pixels(),
		Stride: 4 * i.width,
		Rect:   image.Rect(0, 0, i.width, i.height),
	}); err != nil {
		return err
	}
	return nil
}
