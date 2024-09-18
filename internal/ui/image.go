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
	"github.com/hajimehoshi/ebiten/v2/internal/restorable"
)

// panicOnErrorOnReadingPixels indicates whether reading pixels panics on an error or not.
// This value is set only on testing.
var panicOnErrorOnReadingPixels bool

func SetPanicOnErrorOnReadingPixelsForTesting(value bool) {
	panicOnErrorOnReadingPixels = value
}

const bigOffscreenScale = 2

type Image struct {
	ui *UserInterface

	mipmap    *mipmap.Mipmap
	width     int
	height    int
	imageType atlas.ImageType

	// lastBlend is the lastly-used blend for mipmap.Image.
	lastBlend graphicsdriver.Blend

	// bigOffscreenBuffer is a double-sized offscreen for anti-alias rendering.
	bigOffscreenBuffer *bigOffscreenImage

	// modifyCallback is a callback called when DrawTriangles or WritePixels is called.
	// modifyCallback is useful to detect whether the image is manipulated or not after a certain time.
	modifyCallback func()

	tmpVerticesForFill []float32
}

func (u *UserInterface) NewImage(width, height int, imageType atlas.ImageType) *Image {
	return &Image{
		ui:        u,
		mipmap:    mipmap.New(width, height, imageType),
		width:     width,
		height:    height,
		imageType: imageType,
		lastBlend: graphicsdriver.BlendSourceOver,
	}
}

func (i *Image) Deallocate() {
	if i.mipmap == nil {
		return
	}
	if i.bigOffscreenBuffer != nil {
		i.bigOffscreenBuffer.deallocate()
	}
	i.mipmap.Deallocate()
}

func (i *Image) DrawTriangles(srcs [graphics.ShaderSrcImageCount]*Image, vertices []float32, indices []uint32, blend graphicsdriver.Blend, dstRegion image.Rectangle, srcRegions [graphics.ShaderSrcImageCount]image.Rectangle, shader *Shader, uniforms []uint32, fillRule graphicsdriver.FillRule, canSkipMipmap bool, antialias bool, hint restorable.Hint) {
	if i.modifyCallback != nil {
		i.modifyCallback()
	}

	i.lastBlend = blend

	if antialias {
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
			i.bigOffscreenBuffer = i.ui.newBigOffscreenImage(i, imageType)
		}

		i.bigOffscreenBuffer.drawTriangles(srcs, vertices, indices, blend, dstRegion, srcRegions, shader, uniforms, fillRule, canSkipMipmap)
		return
	}

	i.flushBufferIfNeeded()

	var srcMipmaps [graphics.ShaderSrcImageCount]*mipmap.Mipmap
	for i, src := range srcs {
		if src == nil {
			continue
		}
		src.flushBufferIfNeeded()
		srcMipmaps[i] = src.mipmap
	}

	i.mipmap.DrawTriangles(srcMipmaps, vertices, indices, blend, dstRegion, srcRegions, shader.shader, uniforms, fillRule, canSkipMipmap, hint)
}

func (i *Image) WritePixels(pix []byte, region image.Rectangle) {
	if i.modifyCallback != nil {
		i.modifyCallback()
	}
	i.flushBufferIfNeeded()
	i.mipmap.WritePixels(pix, region)
}

func (i *Image) ReadPixels(pixels []byte, region image.Rectangle) {
	// Check the error existence and avoid unnecessary calls.
	if i.ui.error() != nil {
		return
	}

	i.flushBigOffscreenBufferIfNeeded()

	if err := i.ui.readPixels(i.mipmap, pixels, region); err != nil {
		if panicOnErrorOnReadingPixels {
			panic(err)
		}
		i.ui.setError(err)
	}
}

func (i *Image) DumpScreenshot(name string, blackbg bool) (string, error) {
	i.flushBufferIfNeeded()
	return i.ui.dumpScreenshot(i.mipmap, name, blackbg)
}

func (i *Image) flushBufferIfNeeded() {
	i.flushBigOffscreenBufferIfNeeded()
}

func (i *Image) flushBigOffscreenBufferIfNeeded() {
	if i.bigOffscreenBuffer != nil {
		i.bigOffscreenBuffer.flush()
	}
}

func (u *UserInterface) DumpImages(dir string) (string, error) {
	return u.dumpImages(dir)
}

func (i *Image) clear() {
	i.Fill(0, 0, 0, 0, image.Rect(0, 0, i.width, i.height))
}

func (i *Image) Fill(r, g, b, a float32, region image.Rectangle) {
	if len(i.tmpVerticesForFill) < 4*graphics.VertexFloatCount {
		i.tmpVerticesForFill = make([]float32, 4*graphics.VertexFloatCount)
	}
	// i.tmpVerticesForFill can be reused as this is sent to DrawTriangles immediately.
	graphics.QuadVerticesFromSrcAndMatrix(
		i.tmpVerticesForFill,
		1, 1, float32(i.ui.whiteImage.width-1), float32(i.ui.whiteImage.height-1),
		float32(i.width), 0, 0, float32(i.height), 0, 0,
		r, g, b, a)
	is := graphics.QuadIndices()

	srcs := [graphics.ShaderSrcImageCount]*Image{i.ui.whiteImage}

	blend := graphicsdriver.BlendCopy
	// If possible, use BlendSourceOver to encourage batching (#2817).
	if a == 1 && i.lastBlend == graphicsdriver.BlendSourceOver {
		blend = graphicsdriver.BlendSourceOver
	}
	sr := image.Rect(0, 0, i.ui.whiteImage.width, i.ui.whiteImage.height)
	// i.lastBlend is updated in DrawTriangles.
	i.DrawTriangles(srcs, i.tmpVerticesForFill, is, blend, region, [graphics.ShaderSrcImageCount]image.Rectangle{sr}, NearestFilterShader, nil, graphicsdriver.FillRuleFillAll, true, false, restorable.HintOverwriteDstRegion)
}

type bigOffscreenImage struct {
	ui *UserInterface

	orig      *Image
	imageType atlas.ImageType

	image  *Image
	region image.Rectangle

	blend graphicsdriver.Blend
	dirty bool

	tmpVerticesForFlushing []float32
	tmpVerticesForCopying  []float32
}

func (u *UserInterface) newBigOffscreenImage(orig *Image, imageType atlas.ImageType) *bigOffscreenImage {
	return &bigOffscreenImage{
		ui:        u,
		orig:      orig,
		imageType: imageType,
	}
}

func (i *bigOffscreenImage) deallocate() {
	if i.image != nil {
		i.image.Deallocate()
	}
	i.dirty = false
}

func (i *bigOffscreenImage) drawTriangles(srcs [graphics.ShaderSrcImageCount]*Image, vertices []float32, indices []uint32, blend graphicsdriver.Blend, dstRegion image.Rectangle, srcRegions [graphics.ShaderSrcImageCount]image.Rectangle, shader *Shader, uniforms []uint32, fillRule graphicsdriver.FillRule, canSkipMipmap bool) {
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
		i.image = i.ui.NewImage(i.region.Dx()*bigOffscreenScale, i.region.Dy()*bigOffscreenScale, i.imageType)
	}

	// Copy the current rendering result to get the correct blending result.
	if blend != graphicsdriver.BlendSourceOver && !i.dirty {
		srcs := [graphics.ShaderSrcImageCount]*Image{i.orig}
		if len(i.tmpVerticesForCopying) < 4*graphics.VertexFloatCount {
			i.tmpVerticesForCopying = make([]float32, 4*graphics.VertexFloatCount)
		}
		// i.tmpVerticesForCopying can be reused as this is sent to DrawTriangles immediately.
		graphics.QuadVerticesFromSrcAndMatrix(
			i.tmpVerticesForCopying,
			float32(i.region.Min.X), float32(i.region.Min.Y), float32(i.region.Max.X), float32(i.region.Max.Y),
			bigOffscreenScale, 0, 0, bigOffscreenScale, 0, 0,
			1, 1, 1, 1)
		is := graphics.QuadIndices()
		dstRegion := image.Rect(0, 0, i.region.Dx()*bigOffscreenScale, i.region.Dy()*bigOffscreenScale)
		srcRegion := i.region
		i.image.DrawTriangles(srcs, i.tmpVerticesForCopying, is, graphicsdriver.BlendCopy, dstRegion, [graphics.ShaderSrcImageCount]image.Rectangle{srcRegion}, NearestFilterShader, nil, graphicsdriver.FillRuleFillAll, true, false, restorable.HintOverwriteDstRegion)
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

	i.image.DrawTriangles(srcs, vertices, indices, blend, dstRegion, srcRegions, shader, uniforms, fillRule, canSkipMipmap, false, restorable.HintNone)
	i.dirty = true
}

func (i *bigOffscreenImage) flush() {
	if i.image == nil {
		return
	}

	if !i.dirty {
		return
	}

	// Mark the offscreen clean earlier to avoid recursive calls.
	i.dirty = false

	srcs := [graphics.ShaderSrcImageCount]*Image{i.image}
	if len(i.tmpVerticesForFlushing) < 4*graphics.VertexFloatCount {
		i.tmpVerticesForFlushing = make([]float32, 4*graphics.VertexFloatCount)
	}
	// i.tmpVerticesForFlushing can be reused as this is sent to DrawTriangles in this function.
	graphics.QuadVerticesFromSrcAndMatrix(
		i.tmpVerticesForFlushing,
		0, 0, float32(i.region.Dx()*bigOffscreenScale), float32(i.region.Dy()*bigOffscreenScale),
		1.0/bigOffscreenScale, 0, 0, 1.0/bigOffscreenScale, float32(i.region.Min.X), float32(i.region.Min.Y),
		1, 1, 1, 1)
	is := graphics.QuadIndices()
	dstRegion := i.region
	srcRegion := image.Rect(0, 0, i.region.Dx()*bigOffscreenScale, i.region.Dy()*bigOffscreenScale)
	blend := graphicsdriver.BlendSourceOver
	hint := restorable.HintNone
	if i.blend != graphicsdriver.BlendSourceOver {
		blend = graphicsdriver.BlendCopy
		hint = restorable.HintOverwriteDstRegion
	}
	i.orig.DrawTriangles(srcs, i.tmpVerticesForFlushing, is, blend, dstRegion, [graphics.ShaderSrcImageCount]image.Rectangle{srcRegion}, LinearFilterShader, nil, graphicsdriver.FillRuleFillAll, true, false, hint)

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
