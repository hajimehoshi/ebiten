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
	"github.com/hajimehoshi/ebiten/v2/internal/affine"
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

type Image struct {
	mipmap   *mipmap.Mipmap
	width    int
	height   int
	volatile bool

	dotsCache map[[2]int][4]byte
}

func NewImage(width, height int, imageType atlas.ImageType) *Image {
	return &Image{
		mipmap:   mipmap.New(width, height, imageType),
		width:    width,
		height:   height,
		volatile: imageType == atlas.ImageTypeVolatile,
	}
}

func (i *Image) MarkDisposed() {
	if i.mipmap == nil {
		return
	}
	i.mipmap.MarkDisposed()
	i.mipmap = nil
	i.dotsCache = nil
}

func (i *Image) DrawTriangles(srcs [graphics.ShaderImageCount]*Image, vertices []float32, indices []uint16, colorm affine.ColorM, mode graphicsdriver.CompositeMode, filter graphicsdriver.Filter, address graphicsdriver.Address, dstRegion, srcRegion graphicsdriver.Region, subimageOffsets [graphics.ShaderImageCount - 1][2]float32, shader *Shader, uniforms [][]float32, evenOdd bool, canSkipMipmap bool) {
	i.resolveDotsCacheIfNeeded()

	var srcMipmaps [graphics.ShaderImageCount]*mipmap.Mipmap
	for i, src := range srcs {
		if src == nil {
			continue
		}
		src.resolveDotsCacheIfNeeded()
		srcMipmaps[i] = src.mipmap
	}

	var s *mipmap.Shader
	if shader != nil {
		s = shader.shader
	}

	i.mipmap.DrawTriangles(srcMipmaps, vertices, indices, colorm, mode, filter, address, dstRegion, srcRegion, subimageOffsets, s, uniforms, evenOdd, canSkipMipmap)
}

func (i *Image) WritePixels(pix []byte, x, y, width, height int) {
	if width == 1 && height == 1 {
		if i.dotsCache == nil {
			i.dotsCache = map[[2]int][4]byte{}
		}

		var clr [4]byte
		copy(clr[:], pix)
		i.dotsCache[[2]int{x, y}] = clr

		// One square requires 6 indices (= 2 triangles).
		if len(i.dotsCache) >= graphics.IndicesCount/6 {
			i.resolveDotsCacheIfNeeded()
		}
		return
	}

	i.resolveDotsCacheIfNeeded()
	i.mipmap.WritePixels(pix, x, y, width, height)
}

func (i *Image) ReadPixels(pixels []byte, x, y, width, height int) {
	// Check the error existence and avoid unnecessary calls.
	if theGlobalState.error() != nil {
		return
	}

	if width == 1 && height == 1 {
		if c, ok := i.dotsCache[[2]int{x, y}]; ok {
			copy(pixels, c[:])
			return
		}
		// Do not call resolveDotsCacheIfNeeded here. This would slow (image/draw).Draw.
		// See ebiten.TestImageDrawOver.
	} else {
		i.resolveDotsCacheIfNeeded()
	}

	if err := theUI.readPixels(i.mipmap, pixels, x, y, width, height); err != nil {
		if panicOnErrorOnReadingPixels {
			panic(err)
		}
		theGlobalState.setError(err)
	}
}

func (i *Image) DumpScreenshot(name string, blackbg bool) (string, error) {
	return theUI.dumpScreenshot(i.mipmap, name, blackbg)
}

func (i *Image) resolveDotsCacheIfNeeded() {
	if len(i.dotsCache) == 0 {
		return
	}

	l := len(i.dotsCache)
	vs := graphics.Vertices(l * 4)
	is := make([]uint16, l*6)
	sx, sy := float32(1), float32(1)
	var idx int
	for p, c := range i.dotsCache {
		dx := float32(p[0])
		dy := float32(p[1])
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
	i.dotsCache = nil

	srcs := [graphics.ShaderImageCount]*mipmap.Mipmap{emptyImage.mipmap}
	dr := graphicsdriver.Region{
		X:      0,
		Y:      0,
		Width:  float32(i.width),
		Height: float32(i.height),
	}
	i.mipmap.DrawTriangles(srcs, vs, is, affine.ColorMIdentity{}, graphicsdriver.CompositeModeCopy, graphicsdriver.FilterNearest, graphicsdriver.AddressUnsafe, dr, graphicsdriver.Region{}, [graphics.ShaderImageCount - 1][2]float32{}, NearestFilterShader.shader, nil, false, true)
}

func DumpImages(dir string) (string, error) {
	return theUI.dumpImages(dir)
}

var (
	emptyImage = NewImage(3, 3, atlas.ImageTypeRegular)
)

func init() {
	pix := make([]byte, 4*emptyImage.width*emptyImage.height)
	for i := range pix {
		pix[i] = 0xff
	}
	// As emptyImage is used at Fill, use WritePixels instead.
	emptyImage.WritePixels(pix, 0, 0, emptyImage.width, emptyImage.height)
}

func (i *Image) clear() {
	i.Fill(0, 0, 0, 0, 0, 0, i.width, i.height)
}

func (i *Image) Fill(r, g, b, a float32, x, y, width, height int) {
	dstRegion := graphicsdriver.Region{
		X:      float32(x),
		Y:      float32(y),
		Width:  float32(width),
		Height: float32(height),
	}

	vs := graphics.QuadVertices(
		1, 1, float32(emptyImage.width-1), float32(emptyImage.height-1),
		float32(i.width), 0, 0, float32(i.height), 0, 0,
		r, g, b, a)
	is := graphics.QuadIndices()

	srcs := [graphics.ShaderImageCount]*Image{emptyImage}

	i.DrawTriangles(srcs, vs, is, affine.ColorMIdentity{}, graphicsdriver.CompositeModeCopy, graphicsdriver.FilterNearest, graphicsdriver.AddressUnsafe, dstRegion, graphicsdriver.Region{}, [graphics.ShaderImageCount - 1][2]float32{}, NearestFilterShader, nil, false, true)
}
