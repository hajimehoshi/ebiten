// Copyright 2019 The Ebiten Authors
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

package buffered

import (
	"fmt"
	"image"

	"github.com/hajimehoshi/ebiten/v2/internal/atlas"
	"github.com/hajimehoshi/ebiten/v2/internal/graphics"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
)

var whiteImage *Image

func init() {
	whiteImage = NewImage(3, 3, atlas.ImageTypeRegular)
	pix := make([]byte, 4*3*3)
	for i := range pix {
		pix[i] = 0xff
	}
	whiteImage.WritePixels(pix, image.Rect(0, 0, 3, 3))
}

type Image struct {
	img    *atlas.Image
	width  int
	height int

	// pixels is cached pixels for ReadPixels.
	pixels []byte

	dotsBuffer map[image.Point][4]byte
}

func NewImage(width, height int, imageType atlas.ImageType) *Image {
	return &Image{
		img:    atlas.NewImage(width, height, imageType),
		width:  width,
		height: height,
	}
}

func (i *Image) Deallocate() {
	i.img.Deallocate()
	i.dotsBuffer = nil
}

func (i *Image) ReadPixels(graphicsDriver graphicsdriver.Graphics, pixels []byte, region image.Rectangle) error {
	// Do not call flushDotsBufferIfNeeded here. This would slow (image/draw).Draw.
	// See ebiten.TestImageDrawOver.

	if region.Dx() == 1 && region.Dy() == 1 {
		if c, ok := i.dotsBuffer[region.Min]; ok {
			copy(pixels, c[:])
			return nil
		}
	}

	if i.pixels == nil {
		pix := make([]byte, 4*i.width*i.height)
		if err := i.img.ReadPixels(graphicsDriver, pix, image.Rect(0, 0, i.width, i.height)); err != nil {
			return err
		}
		i.pixels = pix
	}

	lineWidth := 4 * region.Dx()
	for j := 0; j < region.Dy(); j++ {
		dstX := 4 * j * region.Dx()
		srcX := 4 * ((region.Min.Y+j)*i.width + region.Min.X)
		copy(pixels[dstX:dstX+lineWidth], i.pixels[srcX:srcX+lineWidth])
	}

	// ReadPixels doesn't modify the actual pixels.
	// Just 'apply' the dots buffer to the pixels for reading.
	for pos, clr := range i.dotsBuffer {
		if !pos.In(region) {
			continue
		}
		idx := 4 * ((pos.Y-region.Min.Y)*region.Dx() + (pos.X - region.Min.X))
		pixels[idx] = clr[0]
		pixels[idx+1] = clr[1]
		pixels[idx+2] = clr[2]
		pixels[idx+3] = clr[3]
	}

	return nil
}

func (i *Image) DumpScreenshot(graphicsDriver graphicsdriver.Graphics, name string, blackbg bool) (string, error) {
	i.flushDotsBufferIfNeeded()
	return i.img.DumpScreenshot(graphicsDriver, name, blackbg)
}

// WritePixels replaces the pixels at the specified region.
func (i *Image) WritePixels(pix []byte, region image.Rectangle) {
	if l := 4 * region.Dx() * region.Dy(); len(pix) != l {
		panic(fmt.Sprintf("buffered: len(pix) was %d but must be %d", len(pix), l))
	}

	// It is in theroy possible to copy the argument pixels, but this tends to consume a lot of memory.
	// Avoid this unless ReadPixels is called.
	i.pixels = nil

	if region.Dx() == 1 && region.Dy() == 1 {
		if i.dotsBuffer == nil {
			i.dotsBuffer = map[image.Point][4]byte{}
		}

		var clr [4]byte
		copy(clr[:], pix)
		i.dotsBuffer[region.Min] = clr

		if len(i.dotsBuffer) >= 10000 {
			i.flushDotsBufferIfNeeded()
		}
		return
	}

	// Remove entries in the dots buffer that are overwritten by this WritePixels call.
	for pos := range i.dotsBuffer {
		if !pos.In(region) {
			continue
		}
		delete(i.dotsBuffer, pos)
	}

	i.img.WritePixels(pix, region)
}

// DrawTriangles draws the src image with the given vertices.
//
// Copying vertices and indices is the caller's responsibility.
func (i *Image) DrawTriangles(srcs [graphics.ShaderImageCount]*Image, vertices []float32, indices []uint32, blend graphicsdriver.Blend, dstRegion image.Rectangle, srcRegions [graphics.ShaderImageCount]image.Rectangle, shader *atlas.Shader, uniforms []uint32, fillRule graphicsdriver.FillRule) {
	for _, src := range srcs {
		if i == src {
			panic("buffered: Image.DrawTriangles: source images must be different from the receiver")
		}
		if src != nil {
			// src's dots buffer should be applied to sync data in CPU and GPU,
			// but src's pixels doesn't have to be cleared since src is not modified.
			src.flushDotsBufferIfNeeded()
		}
	}

	i.pixels = nil
	i.flushDotsBufferIfNeeded()

	var imgs [graphics.ShaderImageCount]*atlas.Image
	for i, img := range srcs {
		if img == nil {
			continue
		}
		imgs[i] = img.img
	}

	i.img.DrawTriangles(imgs, vertices, indices, blend, dstRegion, srcRegions, shader, uniforms, fillRule)
}

func (i *Image) flushDotsBufferIfNeeded() {
	if len(i.dotsBuffer) == 0 {
		return
	}

	l := len(i.dotsBuffer)
	vs := make([]float32, l*4*graphics.VertexFloatCount)
	is := make([]uint32, l*6)
	sx, sy := float32(1), float32(1)
	var idx int
	for p, c := range i.dotsBuffer {
		dx := float32(p.X)
		dy := float32(p.Y)
		crf := float32(c[0]) / 0xff
		cgf := float32(c[1]) / 0xff
		cbf := float32(c[2]) / 0xff
		caf := float32(c[3]) / 0xff

		vs[graphics.VertexFloatCount*4*idx] = dx
		vs[graphics.VertexFloatCount*4*idx+1] = dy
		vs[graphics.VertexFloatCount*4*idx+2] = sx
		vs[graphics.VertexFloatCount*4*idx+3] = sy
		vs[graphics.VertexFloatCount*4*idx+4] = crf
		vs[graphics.VertexFloatCount*4*idx+5] = cgf
		vs[graphics.VertexFloatCount*4*idx+6] = cbf
		vs[graphics.VertexFloatCount*4*idx+7] = caf
		vs[graphics.VertexFloatCount*4*idx+8] = dx + 1
		vs[graphics.VertexFloatCount*4*idx+9] = dy
		vs[graphics.VertexFloatCount*4*idx+10] = sx + 1
		vs[graphics.VertexFloatCount*4*idx+11] = sy
		vs[graphics.VertexFloatCount*4*idx+12] = crf
		vs[graphics.VertexFloatCount*4*idx+13] = cgf
		vs[graphics.VertexFloatCount*4*idx+14] = cbf
		vs[graphics.VertexFloatCount*4*idx+15] = caf
		vs[graphics.VertexFloatCount*4*idx+16] = dx
		vs[graphics.VertexFloatCount*4*idx+17] = dy + 1
		vs[graphics.VertexFloatCount*4*idx+18] = sx
		vs[graphics.VertexFloatCount*4*idx+19] = sy + 1
		vs[graphics.VertexFloatCount*4*idx+20] = crf
		vs[graphics.VertexFloatCount*4*idx+21] = cgf
		vs[graphics.VertexFloatCount*4*idx+22] = cbf
		vs[graphics.VertexFloatCount*4*idx+23] = caf
		vs[graphics.VertexFloatCount*4*idx+24] = dx + 1
		vs[graphics.VertexFloatCount*4*idx+25] = dy + 1
		vs[graphics.VertexFloatCount*4*idx+26] = sx + 1
		vs[graphics.VertexFloatCount*4*idx+27] = sy + 1
		vs[graphics.VertexFloatCount*4*idx+28] = crf
		vs[graphics.VertexFloatCount*4*idx+29] = cgf
		vs[graphics.VertexFloatCount*4*idx+30] = cbf
		vs[graphics.VertexFloatCount*4*idx+31] = caf

		is[6*idx] = uint32(4 * idx)
		is[6*idx+1] = uint32(4*idx + 1)
		is[6*idx+2] = uint32(4*idx + 2)
		is[6*idx+3] = uint32(4*idx + 1)
		is[6*idx+4] = uint32(4*idx + 2)
		is[6*idx+5] = uint32(4*idx + 3)

		idx++
	}

	if i.pixels != nil {
		for pos, clr := range i.dotsBuffer {
			idx := 4 * (pos.Y*i.width + pos.X)
			i.pixels[idx] = clr[0]
			i.pixels[idx+1] = clr[1]
			i.pixels[idx+2] = clr[2]
			i.pixels[idx+3] = clr[3]
			delete(i.dotsBuffer, pos)
		}
	} else {
		// TODO: Use clear if Go 1.21 is available.
		for pos := range i.dotsBuffer {
			delete(i.dotsBuffer, pos)
		}
	}

	srcs := [graphics.ShaderImageCount]*atlas.Image{whiteImage.img}
	dr := image.Rect(0, 0, i.width, i.height)
	blend := graphicsdriver.BlendCopy
	i.img.DrawTriangles(srcs, vs, is, blend, dr, [graphics.ShaderImageCount]image.Rectangle{}, atlas.NearestFilterShader, nil, graphicsdriver.FillAll)
}
