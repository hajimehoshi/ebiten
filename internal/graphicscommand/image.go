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
	"sort"
	"strings"

	"github.com/hajimehoshi/ebiten/v2/internal/affine"
	"github.com/hajimehoshi/ebiten/v2/internal/debug"
	"github.com/hajimehoshi/ebiten/v2/internal/graphics"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/png"
)

// Image represents an image that is implemented with OpenGL.
type Image struct {
	image          graphicsdriver.Image
	width          int
	height         int
	internalWidth  int
	internalHeight int
	screen         bool

	// id is an indentifier for the image. This is used only when dummping the information.
	//
	// This is duplicated with graphicsdriver.Image's ID, but this id is still necessary because this image might not
	// have its graphicsdriver.Image.
	id int

	bufferedWP []*graphicsdriver.WritePixelsArgs
}

var nextID = 1

func genNextID() int {
	id := nextID
	nextID++
	return id
}

// unresolvedImages is the set of unresolved images.
// An unresolved image is an image that might have an state unsent to the command queue yet.
var unresolvedImages []*Image

// addUnresolvedImage adds an image to the list of unresolved images.
func addUnresolvedImage(img *Image) {
	unresolvedImages = append(unresolvedImages, img)
}

// resolveImages resolves all the image states unsent to the command queue.
// resolveImages should be called before flushing commands.
func resolveImages() {
	for i, img := range unresolvedImages {
		img.resolveBufferedWritePixels()
		unresolvedImages[i] = nil
	}
	unresolvedImages = unresolvedImages[:0]
}

// NewImage returns a new image.
//
// Note that the image is not initialized yet.
func NewImage(width, height int, screenFramebuffer bool) *Image {
	i := &Image{
		width:  width,
		height: height,
		screen: screenFramebuffer,
		id:     genNextID(),
	}
	c := &newImageCommand{
		result: i,
		width:  width,
		height: height,
		screen: screenFramebuffer,
	}
	theCommandQueue.Enqueue(c)
	return i
}

func (i *Image) resolveBufferedWritePixels() {
	if len(i.bufferedWP) == 0 {
		return
	}
	c := &writePixelsCommand{
		dst:  i,
		args: i.bufferedWP,
	}
	theCommandQueue.Enqueue(c)
	i.bufferedWP = nil
}

func (i *Image) Dispose() {
	i.bufferedWP = nil
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
//	0: Destination X in pixels
//	1: Destination Y in pixels
//	2: Source X in texels
//	3: Source Y in texels
//	4: Color R [0.0-1.0]
//	5: Color G
//	6: Color B
//	7: Color Y
//
// src and shader are exclusive and only either is non-nil.
//
// The elements that index is in between 2 and 7 are used for the source images.
// The source image is 1) src argument if non-nil, or 2) an image value in the uniform variables if it exists.
// If there are multiple images in the uniform variables, the smallest ID's value is adopted.
//
// If the source image is not specified, i.e., src is nil and there is no image in the uniform variables, the
// elements for the source image are not used.
func (i *Image) DrawTriangles(srcs [graphics.ShaderImageCount]*Image, offsets [graphics.ShaderImageCount - 1][2]float32, vertices []float32, indices []uint16, clr affine.ColorM, mode graphicsdriver.CompositeMode, filter graphicsdriver.Filter, address graphicsdriver.Address, dstRegion, srcRegion graphicsdriver.Region, shader *Shader, uniforms [][]float32, evenOdd bool) {
	if shader == nil {
		// Fast path for rendering without a shader (#1355).
		img := srcs[0]
		if img.screen {
			panic("graphicscommand: the screen image cannot be the rendering source")
		}
		img.resolveBufferedWritePixels()
	} else {
		for _, src := range srcs {
			if src == nil {
				continue
			}
			if src.screen {
				panic("graphicscommand: the screen image cannot be the rendering source")
			}
			src.resolveBufferedWritePixels()
		}
	}
	i.resolveBufferedWritePixels()

	theCommandQueue.EnqueueDrawTrianglesCommand(i, srcs, offsets, vertices, indices, clr, mode, filter, address, dstRegion, srcRegion, shader, uniforms, evenOdd)
}

// ReadPixels reads the image's pixels.
// ReadPixels returns an error when an error happens in the graphics driver.
func (i *Image) ReadPixels(graphicsDriver graphicsdriver.Graphics, buf []byte) error {
	i.resolveBufferedWritePixels()
	c := &readPixelsCommand{
		img:    i,
		result: buf,
	}
	theCommandQueue.Enqueue(c)
	if err := theCommandQueue.Flush(graphicsDriver); err != nil {
		return err
	}
	return nil
}

func (i *Image) WritePixels(pixels []byte, x, y, width, height int) {
	i.bufferedWP = append(i.bufferedWP, &graphicsdriver.WritePixelsArgs{
		Pixels: pixels,
		X:      x,
		Y:      y,
		Width:  width,
		Height: height,
	})
	addUnresolvedImage(i)
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
func (i *Image) Dump(graphicsDriver graphicsdriver.Graphics, path string, blackbg bool, rect image.Rectangle) error {
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

	pix := make([]byte, 4*i.width*i.height)
	if err := i.ReadPixels(graphicsDriver, pix); err != nil {
		return err
	}

	if blackbg {
		for i := 0; i < len(pix)/4; i++ {
			pix[4*i+3] = 0xff
		}
	}

	if err := png.Encode(f, (&image.RGBA{
		Pix:    pix,
		Stride: 4 * i.width,
		Rect:   image.Rect(0, 0, i.width, i.height),
	}).SubImage(rect)); err != nil {
		return err
	}
	return nil
}

func LogImagesInfo(images []*Image) {
	sort.Slice(images, func(a, b int) bool {
		return images[a].id < images[b].id
	})
	for _, i := range images {
		w, h := i.InternalSize()
		debug.Logf("  %d: (%d, %d)\n", i.id, w, h)
	}
}
