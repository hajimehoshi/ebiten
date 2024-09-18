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
	"io"
	"sort"
	"strconv"
	"strings"

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

	// attribute is used only for logs.
	attribute string

	// id is an identifier for the image. This is used only when dumping the information.
	//
	// This is duplicated with graphicsdriver.Image's ID, but this id is still necessary because this image might not
	// have its graphicsdriver.Image.
	id int

	bufferedWritePixelsArgs []writePixelsCommandArgs
}

var nextImageID = 1

func genNextImageID() int {
	id := nextImageID
	nextImageID++
	return id
}

// NewImage returns a new image.
//
// Note that the image is not initialized yet.
func NewImage(width, height int, screenFramebuffer bool, attribute string) *Image {
	i := &Image{
		width:     width,
		height:    height,
		screen:    screenFramebuffer,
		id:        genNextImageID(),
		attribute: attribute,
	}
	c := &newImageCommand{
		result:    i,
		width:     width,
		height:    height,
		screen:    screenFramebuffer,
		attribute: attribute,
	}
	theCommandQueueManager.enqueueCommand(c)
	return i
}

func (i *Image) flushBufferedWritePixels() {
	if len(i.bufferedWritePixelsArgs) == 0 {
		return
	}

	c := &writePixelsCommand{
		dst:  i,
		args: i.bufferedWritePixelsArgs,
	}
	theCommandQueueManager.enqueueCommand(c)

	i.bufferedWritePixelsArgs = nil
}

func (i *Image) Dispose() {
	i.bufferedWritePixelsArgs = nil
	c := &disposeImageCommand{
		target: i,
	}
	theCommandQueueManager.enqueueCommand(c)
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
func (i *Image) DrawTriangles(srcs [graphics.ShaderSrcImageCount]*Image, vertices []float32, indices []uint32, blend graphicsdriver.Blend, dstRegion image.Rectangle, srcRegions [graphics.ShaderSrcImageCount]image.Rectangle, shader *Shader, uniforms []uint32, fillRule graphicsdriver.FillRule) {
	for _, src := range srcs {
		if src == nil {
			continue
		}
		if src.screen {
			panic("graphicscommand: the screen image cannot be the rendering source")
		}
		src.flushBufferedWritePixels()
	}
	i.flushBufferedWritePixels()

	theCommandQueueManager.enqueueDrawTrianglesCommand(i, srcs, vertices, indices, blend, dstRegion, srcRegions, shader, uniforms, fillRule)
}

// ReadPixels reads the image's pixels.
// ReadPixels returns an error when an error happens in the graphics driver.
func (i *Image) ReadPixels(graphicsDriver graphicsdriver.Graphics, args []graphicsdriver.PixelsArgs) error {
	i.flushBufferedWritePixels()
	c := &readPixelsCommand{
		img:  i,
		args: args,
	}
	theCommandQueueManager.enqueueCommand(c)
	if err := theCommandQueueManager.flush(graphicsDriver, false); err != nil {
		return err
	}
	return nil
}

func (i *Image) WritePixels(pixels *graphics.ManagedBytes, region image.Rectangle) {
	// Release the previous pixels if the region is included by the new region.
	// Successive WritePixels calls might accumulate the pixels and never release,
	// especially when the image is unmanaged (#3036).
	var cur int
	for idx := 0; idx < len(i.bufferedWritePixelsArgs); idx++ {
		arg := i.bufferedWritePixelsArgs[idx]
		if arg.region.In(region) {
			arg.pixels.Release()
			continue
		}
		i.bufferedWritePixelsArgs[cur] = arg
		cur++
	}
	for idx := cur; idx < len(i.bufferedWritePixelsArgs); idx++ {
		i.bufferedWritePixelsArgs[idx] = writePixelsCommandArgs{}
	}
	i.bufferedWritePixelsArgs = append(i.bufferedWritePixelsArgs[:cur], writePixelsCommandArgs{
		pixels: pixels,
		region: region,
	})
}

func (i *Image) dumpName(path string) string {
	return strings.ReplaceAll(path, "*", strconv.Itoa(i.id))
}

// dumpTo dumps the image to the specified writer.
//
// If blackbg is true, any alpha values in the dumped image will be 255.
//
// This is for testing usage.
func (i *Image) dumpTo(w io.Writer, graphicsDriver graphicsdriver.Graphics, blackbg bool, rect image.Rectangle) error {
	if i.screen {
		return fmt.Errorf("graphicscommand: a screen image cannot be dumped")
	}

	pix := make([]byte, 4*i.width*i.height)
	if err := i.ReadPixels(graphicsDriver, []graphicsdriver.PixelsArgs{
		{
			Pixels: pix,
			Region: image.Rect(0, 0, i.width, i.height),
		},
	}); err != nil {
		return err
	}

	if blackbg {
		for i := 0; i < len(pix)/4; i++ {
			pix[4*i+3] = 0xff
		}
	}

	if err := png.Encode(w, (&image.RGBA{
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
		var screen string
		if i.screen {
			screen = " (screen)"
		}
		debug.FrameLogf("  %d: (%d, %d)%s\n", i.id, w, h, screen)
	}
}
