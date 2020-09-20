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

	// id is an indentifier for the image. This is used only when dummping the information.
	//
	// This is duplicated with driver.Image's ID, but this id is still necessary because this image might not
	// have its driver.Image.
	id int

	bufferedRP []*driver.ReplacePixelsArgs

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
		width:  width,
		height: height,
		id:     genNextID(),
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
		screen: true,
		id:     genNextID(),
	}
	c := &newScreenFramebufferImageCommand{
		result: i,
		width:  width,
		height: height,
	}
	theCommandQueue.Enqueue(c)
	return i
}

func (i *Image) resolveBufferedReplacePixels() {
	if len(i.bufferedRP) == 0 {
		return
	}
	c := &replacePixelsCommand{
		dst:  i,
		args: i.bufferedRP,
	}
	theCommandQueue.Enqueue(c)
	i.bufferedRP = nil
}

func (i *Image) Dispose() {
	c := &disposeImageCommand{
		target: i,
	}
	theCommandQueue.Enqueue(c)
}

func (i *Image) InternalSize() (int, int) {
	if i.screen {
		return i.width, i.height
	}
	if i.internalWidth == 0 {
		i.internalWidth = graphics.InternalImageSize(i.width)
	}
	if i.internalHeight == 0 {
		i.internalHeight = graphics.InternalImageSize(i.height)
	}
	return i.internalWidth, i.internalHeight
}

// DrawTriangles draws triangles with the given image.
//
// The vertex floats are:
//
//   0: Destination X in pixels
//   1: Destination Y in pixels
//   2: Source X in pixels (not texels!)
//   3: Source Y in pixels
//   4: Color R [0.0-1.0]
//   5: Color G
//   6: Color B
//   7: Color Y
//
// src and shader are exclusive and only either is non-nil.
//
// The elements that index is in between 2 and 7 are used for the source images.
// The source image is 1) src argument if non-nil, or 2) an image value in the uniform variables if it exists.
// If there are multiple images in the uniform variables, the smallest ID's value is adopted.
//
// If the source image is not specified, i.e., src is nil and there is no image in the uniform variables, the
// elements for the source image are not used.
func (i *Image) DrawTriangles(srcs [graphics.ShaderImageNum]*Image, offsets [graphics.ShaderImageNum - 1][2]float32, vertices []float32, indices []uint16, clr *affine.ColorM, mode driver.CompositeMode, filter driver.Filter, address driver.Address, sourceRegion driver.Region, shader *Shader, uniforms []interface{}) {
	if i.lastCommand == lastCommandNone {
		if !i.screen && mode != driver.CompositeModeClear {
			panic("graphicscommand: the image must be cleared first")
		}
	}

	if shader == nil {
		// Fast path for rendering without a shader (#1355).
		img := srcs[0]
		if img.screen {
			panic("graphicscommand: the screen image cannot be the rendering source")
		}
		img.resolveBufferedReplacePixels()
	} else {
		for _, src := range srcs {
			if src == nil {
				continue
			}
			if src.screen {
				panic("graphicscommand: the screen image cannot be the rendering source")
			}
			src.resolveBufferedReplacePixels()
		}
	}
	i.resolveBufferedReplacePixels()

	theCommandQueue.EnqueueDrawTrianglesCommand(i, srcs, offsets, vertices, indices, clr, mode, filter, address, sourceRegion, shader, uniforms)

	if i.lastCommand == lastCommandNone && !i.screen {
		i.lastCommand = lastCommandClear
	} else {
		i.lastCommand = lastCommandDrawTriangles
	}
}

// Pixels returns the image's pixels.
// Pixels might return nil when OpenGL error happens.
func (i *Image) Pixels() ([]byte, error) {
	i.resolveBufferedReplacePixels()
	c := &pixelsCommand{
		result: nil,
		img:    i,
	}
	theCommandQueue.Enqueue(c)
	if err := theCommandQueue.Flush(); err != nil {
		return nil, err
	}
	return c.result, nil
}

func (i *Image) ReplacePixels(pixels []byte, x, y, width, height int) {
	// ReplacePixels for a part might invalidate the current image that are drawn by DrawTriangles (#593, #738).
	if i.lastCommand == lastCommandDrawTriangles {
		if x != 0 || y != 0 || i.width != width || i.height != height {
			panic("graphicscommand: ReplacePixels for a part after DrawTriangles is forbidden")
		}
	}
	i.bufferedRP = append(i.bufferedRP, &driver.ReplacePixelsArgs{
		Pixels: pixels,
		X:      x,
		Y:      y,
		Width:  width,
		Height: height,
	})
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
// If blackbg is true, any alpha values in the dumped image will be 255.
//
// This is for testing usage.
func (i *Image) Dump(path string, blackbg bool) error {
	// Screen image cannot be dumped.
	if i.screen {
		return nil
	}

	path = strings.ReplaceAll(path, "*", fmt.Sprintf("%d", i.id))
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	pix, err := i.Pixels()
	if err != nil {
		return err
	}

	if blackbg {
		for i := 0; i < len(pix)/4; i++ {
			pix[4*i+3] = 0xff
		}
	}

	if err := png.Encode(f, &image.RGBA{
		Pix:    pix,
		Stride: 4 * i.width,
		Rect:   image.Rect(0, 0, i.width, i.height),
	}); err != nil {
		return err
	}
	return nil
}
