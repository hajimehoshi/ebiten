// Copyright 2022 The Ebiten Authors
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

package ui

import (
	"fmt"
	"image"
	"math"

	"github.com/hajimehoshi/ebiten/v2/internal/atlas"
	"github.com/hajimehoshi/ebiten/v2/internal/graphics"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/mipmap"
)

// panicOnErrorOnReadingPixels indicates whether reading pixels panics on an error or not.
// This value is set only on testing.
var panicOnErrorOnReadingPixels bool

func SetPanicOnErrorOnReadingPixelsForTesting(value bool) {
	panicOnErrorOnReadingPixels = value
}

const bigOffscreenScale = 2

type Image struct {
	mipmap    *mipmap.Mipmap
	width     int
	height    int
	imageType atlas.ImageType

	dotsBuffer map[image.Point][4]byte

	// bigOffscreenBuffer is a double-sized offscreen for anti-alias rendering.
	bigOffscreenBuffer *bigOffscreenImage

	// modifyCallback is a callback called when DrawTriangles or WritePixels is called.
	// modifyCallback is useful to detect whether the image is manipulated or not after a certain time.
	modifyCallback func()

	tmpVerticesForFill []float32
}

func NewImage(width, height int, imageType atlas.ImageType) *Image {
	return &Image{
		mipmap:    mipmap.New(width, height, imageType),
		width:     width,
		height:    height,
		imageType: imageType,
	}
}

func (i *Image) MarkDisposed() {
	if i.mipmap == nil {
		return
	}
	if i.bigOffscreenBuffer != nil {
		i.bigOffscreenBuffer.markDisposed()
		i.bigOffscreenBuffer = nil
	}
	i.mipmap.MarkDisposed()
	i.mipmap = nil
	i.dotsBuffer = nil
	i.modifyCallback = nil
}

func (i *Image) DrawTriangles(srcs [graphics.ShaderImageCount]*Image, vertices []float32, indices []uint16, blend graphicsdriver.Blend, dstRegion image.Rectangle, srcRegions [graphics.ShaderImageCount]image.Rectangle, shader *Shader, uniforms []uint32, evenOdd bool, canSkipMipmap bool, antialias bool) {
	if i.modifyCallback != nil {
		i.modifyCallback()
	}

	if antialias {
		// Flush the other buffer to make the buffers exclusive.
		i.flushDotsBufferIfNeeded()

		if i.bigOffscreenBuffer == nil {
			var imageType atlas.ImageType
			switch i.imageType {
			case atlas.ImageTypeRegular, atlas.ImageTypeUnmanaged:
				imageType = atlas.ImageTypeUnmanaged
			case atlas.ImageTypeScreen, atlas.ImageTypeVolatile:
				imageType = atlas.ImageTypeVolatile
			default:
				panic(fmt.Sprintf("ui: unexpected image type: %d", imageType))
			}
			i.bigOffscreenBuffer = newBigOffscreenImage(i, imageType)
		}

		i.bigOffscreenBuffer.drawTriangles(srcs, vertices, indices, blend, dstRegion, srcRegions, shader, uniforms, evenOdd, canSkipMipmap, false)
		return
	}

	i.flushBufferIfNeeded()

	var srcMipmaps [graphics.ShaderImageCount]*mipmap.Mipmap
	for i, src := range srcs {
		if src == nil {
			continue
		}
		src.flushBufferIfNeeded()
		srcMipmaps[i] = src.mipmap
	}

	i.mipmap.DrawTriangles(srcMipmaps, vertices, indices, blend, dstRegion, srcRegions, shader.shader, uniforms, evenOdd, canSkipMipmap)
}

func (i *Image) WritePixels(pix []byte, region image.Rectangle) {
	if i.modifyCallback != nil {
		i.modifyCallback()
	}

	if region.Dx() == 1 && region.Dy() == 1 {
		// Flush the other buffer to make the buffers exclusive.
		i.flushBigOffscreenBufferIfNeeded()

		if i.dotsBuffer == nil {
			i.dotsBuffer = map[image.Point][4]byte{}
		}

		var clr [4]byte
		copy(clr[:], pix)
		i.dotsBuffer[region.Min] = clr

		// One square requires 6 indices (= 2 triangles).
		if len(i.dotsBuffer) >= graphics.MaxVerticesCount/6 {
			i.flushDotsBufferIfNeeded()
		}
		return
	}

	i.flushBufferIfNeeded()
	i.mipmap.WritePixels(pix, region)
}

func (i *Image) ReadPixels(pixels []byte, region image.Rectangle) {
	// Check the error existence and avoid unnecessary calls.
	if theGlobalState.error() != nil {
		return
	}

	i.flushBigOffscreenBufferIfNeeded()

	if region.Dx() == 1 && region.Dy() == 1 {
		if c, ok := i.dotsBuffer[region.Min]; ok {
			copy(pixels, c[:])
			return
		}
		// Do not call flushDotsBufferIfNeeded here. This would slow (image/draw).Draw.
		// See ebiten.TestImageDrawOver.
	} else {
		i.flushDotsBufferIfNeeded()
	}

	if err := theUI.readPixels(i.mipmap, pixels, region); err != nil {
		if panicOnErrorOnReadingPixels {
			panic(err)
		}
		theGlobalState.setError(err)
	}
}

func (i *Image) DumpScreenshot(name string, blackbg bool) (string, error) {
	i.flushBufferIfNeeded()
	return theUI.dumpScreenshot(i.mipmap, name, blackbg)
}

func (i *Image) flushBufferIfNeeded() {
	// The buffers are exclusive and the order should not matter.
	i.flushDotsBufferIfNeeded()
	i.flushBigOffscreenBufferIfNeeded()
}

func (i *Image) flushDotsBufferIfNeeded() {
	if len(i.dotsBuffer) == 0 {
		return
	}

	l := len(i.dotsBuffer)
	vs := make([]float32, l*4*graphics.VertexFloatCount)
	is := make([]uint16, l*6)
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

		is[6*idx] = uint16(4 * idx)
		is[6*idx+1] = uint16(4*idx + 1)
		is[6*idx+2] = uint16(4*idx + 2)
		is[6*idx+3] = uint16(4*idx + 1)
		is[6*idx+4] = uint16(4*idx + 2)
		is[6*idx+5] = uint16(4*idx + 3)

		idx++
	}
	i.dotsBuffer = nil

	srcs := [graphics.ShaderImageCount]*mipmap.Mipmap{whiteImage.mipmap}
	dr := image.Rect(0, 0, i.width, i.height)
	i.mipmap.DrawTriangles(srcs, vs, is, graphicsdriver.BlendCopy, dr, [graphics.ShaderImageCount]image.Rectangle{}, NearestFilterShader.shader, nil, false, true)
}

func (i *Image) flushBigOffscreenBufferIfNeeded() {
	if i.bigOffscreenBuffer != nil {
		i.bigOffscreenBuffer.flush()
	}
}

func DumpImages(dir string) (string, error) {
	return theUI.dumpImages(dir)
}

var (
	whiteImage = NewImage(3, 3, atlas.ImageTypeRegular)
)

func init() {
	pix := make([]byte, 4*whiteImage.width*whiteImage.height)
	for i := range pix {
		pix[i] = 0xff
	}
	// As whiteImage is used at Fill, use WritePixels instead.
	whiteImage.WritePixels(pix, image.Rect(0, 0, whiteImage.width, whiteImage.height))
}

func (i *Image) clear() {
	i.Fill(0, 0, 0, 0, image.Rect(0, 0, i.width, i.height))
}

func (i *Image) Fill(r, g, b, a float32, region image.Rectangle) {
	if len(i.tmpVerticesForFill) < 4*graphics.VertexFloatCount {
		i.tmpVerticesForFill = make([]float32, 4*graphics.VertexFloatCount)
	}
	// i.tmpVerticesForFill can be reused as this is sent to DrawTriangles immediately.
	graphics.QuadVertices(
		i.tmpVerticesForFill,
		1, 1, float32(whiteImage.width-1), float32(whiteImage.height-1),
		float32(i.width), 0, 0, float32(i.height), 0, 0,
		r, g, b, a)
	is := graphics.QuadIndices()

	srcs := [graphics.ShaderImageCount]*Image{whiteImage}

	i.DrawTriangles(srcs, i.tmpVerticesForFill, is, graphicsdriver.BlendCopy, region, [graphics.ShaderImageCount]image.Rectangle{}, NearestFilterShader, nil, false, true, false)
}

type bigOffscreenImage struct {
	orig      *Image
	imageType atlas.ImageType

	image  *Image
	region image.Rectangle

	blend graphicsdriver.Blend
	dirty bool

	tmpVerticesForFlushing []float32
	tmpVerticesForCopying  []float32
}

func newBigOffscreenImage(orig *Image, imageType atlas.ImageType) *bigOffscreenImage {
	return &bigOffscreenImage{
		orig:      orig,
		imageType: imageType,
	}
}

func (i *bigOffscreenImage) markDisposed() {
	if i.image != nil {
		i.image.MarkDisposed()
		i.image = nil
	}
	i.dirty = false
}

func (i *bigOffscreenImage) drawTriangles(srcs [graphics.ShaderImageCount]*Image, vertices []float32, indices []uint16, blend graphicsdriver.Blend, dstRegion image.Rectangle, srcRegions [graphics.ShaderImageCount]image.Rectangle, shader *Shader, uniforms []uint32, evenOdd bool, canSkipMipmap bool, antialias bool) {
	if i.blend != blend {
		i.flush()
	}
	i.blend = blend

	// If the new region doesn't match with the current region, remove the buffer image and recreate it later.
	if r := i.requiredRegion(vertices); i.region != r {
		i.flush()
		i.image = nil
		i.region = r
	}

	if i.region.Empty() {
		return
	}

	if i.image == nil {
		i.image = NewImage(i.region.Dx()*bigOffscreenScale, i.region.Dy()*bigOffscreenScale, i.imageType)
	}

	// Copy the current rendering result to get the correct blending result.
	if blend != graphicsdriver.BlendSourceOver && !i.dirty {
		srcs := [graphics.ShaderImageCount]*Image{i.orig}
		if len(i.tmpVerticesForCopying) < 4*graphics.VertexFloatCount {
			i.tmpVerticesForCopying = make([]float32, 4*graphics.VertexFloatCount)
		}
		// i.tmpVerticesForCopying can be resused as this is sent to DrawTriangles immediately.
		graphics.QuadVertices(
			i.tmpVerticesForCopying,
			float32(i.region.Min.X), float32(i.region.Min.Y), float32(i.region.Max.X), float32(i.region.Max.Y),
			bigOffscreenScale, 0, 0, bigOffscreenScale, 0, 0,
			1, 1, 1, 1)
		is := graphics.QuadIndices()
		dstRegion := image.Rect(0, 0, i.region.Dx()*bigOffscreenScale, i.region.Dy()*bigOffscreenScale)
		i.image.DrawTriangles(srcs, i.tmpVerticesForCopying, is, graphicsdriver.BlendCopy, dstRegion, [graphics.ShaderImageCount]image.Rectangle{}, NearestFilterShader, nil, false, true, false)
	}

	for idx := 0; idx < len(vertices); idx += graphics.VertexFloatCount {
		vertices[idx] = (vertices[idx] - float32(i.region.Min.X)) * bigOffscreenScale
		vertices[idx+1] = (vertices[idx+1] - float32(i.region.Min.Y)) * bigOffscreenScale
	}

	// Translate to i.region coordinate space, and clamp against region size.
	dstRegion = dstRegion.Sub(i.region.Min)
	dstRegion = dstRegion.Intersect(image.Rect(0, 0, i.region.Dx(), i.region.Dy()))
	dstRegion.Min.X *= bigOffscreenScale
	dstRegion.Min.Y *= bigOffscreenScale
	dstRegion.Max.X *= bigOffscreenScale
	dstRegion.Max.Y *= bigOffscreenScale

	i.image.DrawTriangles(srcs, vertices, indices, blend, dstRegion, srcRegions, shader, uniforms, evenOdd, canSkipMipmap, false)
	i.dirty = true
}

func (i *bigOffscreenImage) flush() {
	if i.image == nil {
		return
	}

	if !i.dirty {
		return
	}

	// Mark the offscreen clearn earlier to avoid recursive calls.
	i.dirty = false

	srcs := [graphics.ShaderImageCount]*Image{i.image}
	if len(i.tmpVerticesForFlushing) < 4*graphics.VertexFloatCount {
		i.tmpVerticesForFlushing = make([]float32, 4*graphics.VertexFloatCount)
	}
	// i.tmpVerticesForFlushing can be reused as this is sent to DrawTriangles in this function.
	graphics.QuadVertices(
		i.tmpVerticesForFlushing,
		0, 0, float32(i.region.Dx()*bigOffscreenScale), float32(i.region.Dy()*bigOffscreenScale),
		1.0/bigOffscreenScale, 0, 0, 1.0/bigOffscreenScale, float32(i.region.Min.X), float32(i.region.Min.Y),
		1, 1, 1, 1)
	is := graphics.QuadIndices()
	dstRegion := i.region
	blend := graphicsdriver.BlendSourceOver
	if i.blend != graphicsdriver.BlendSourceOver {
		blend = graphicsdriver.BlendCopy
	}
	i.orig.DrawTriangles(srcs, i.tmpVerticesForFlushing, is, blend, dstRegion, [graphics.ShaderImageCount]image.Rectangle{}, LinearFilterShader, nil, false, true, false)

	i.image.clear()
	i.dirty = false
}

func (i *bigOffscreenImage) requiredRegion(vertices []float32) image.Rectangle {
	minX := float32(i.orig.width)
	minY := float32(i.orig.height)
	maxX := float32(0)
	maxY := float32(0)
	for i := 0; i < len(vertices); i += graphics.VertexFloatCount {
		dstX := vertices[i]
		dstY := vertices[i+1]
		if minX > floor(dstX)-1 {
			minX = floor(dstX) - 1
		}
		if minY > floor(dstY)-1 {
			minY = floor(dstY) - 1
		}
		if maxX < ceil(dstX)+1 {
			maxX = ceil(dstX) + 1
		}
		if maxY < ceil(dstY)+1 {
			maxY = ceil(dstY) + 1
		}
	}

	// Adjust the granularity of the rectangle.
	r := image.Rect(
		roundDown16(int(minX)),
		roundDown16(int(minY)),
		roundUp16(int(maxX)),
		roundUp16(int(maxY)))
	r = r.Intersect(image.Rect(0, 0, i.orig.width, i.orig.height))

	// TODO: Is this check required?
	if r.Dx() < 0 || r.Dy() < 0 {
		return i.region
	}

	return r.Union(i.region)
}

func floor(x float32) float32 {
	return float32(math.Floor(float64(x)))
}

func ceil(x float32) float32 {
	return float32(math.Ceil(float64(x)))
}

func roundDown16(x int) int {
	return x & ^(0xf)
}

func roundUp16(x int) int {
	return ((x - 1) & ^(0xf)) + 0x10
}
