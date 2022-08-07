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
}

func (i *Image) DrawTriangles(srcs [graphics.ShaderImageCount]*Image, vertices []float32, indices []uint16, colorm affine.ColorM, mode graphicsdriver.CompositeMode, filter graphicsdriver.Filter, address graphicsdriver.Address, dstRegion, srcRegion graphicsdriver.Region, subimageOffsets [graphics.ShaderImageCount - 1][2]float32, shader *Shader, uniforms [][]float32, evenOdd bool, canSkipMipmap bool) {
	var srcMipmaps [graphics.ShaderImageCount]*mipmap.Mipmap
	for i, src := range srcs {
		if src == nil {
			continue
		}
		srcMipmaps[i] = src.mipmap
	}

	var s *mipmap.Shader
	if shader != nil {
		s = shader.shader
	}

	i.mipmap.DrawTriangles(srcMipmaps, vertices, indices, colorm, mode, filter, address, dstRegion, srcRegion, subimageOffsets, s, uniforms, evenOdd, canSkipMipmap)
}

func (i *Image) WritePixels(pix []byte, x, y, width, height int) {
	i.mipmap.WritePixels(pix, x, y, width, height)
}

func (i *Image) ReadPixels(pixels []byte, x, y, width, height int) {
	// Check the error existence and avoid unnecessary calls.
	if theGlobalState.error() != nil {
		return
	}

	if err := theUI.readPixels(i.mipmap, pixels, x, y, width, height); err != nil {
		if panicOnErrorOnReadingPixels {
			panic(err)
		}
		theGlobalState.setError(err)
	}
}

func (i *Image) DumpScreenshot(name string, blackbg bool) error {
	return theUI.dumpScreenshot(i.mipmap, name, blackbg)
}

func DumpImages(dir string) error {
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

	i.DrawTriangles(srcs, vs, is, affine.ColorMIdentity{}, graphicsdriver.CompositeModeCopy, graphicsdriver.FilterNearest, graphicsdriver.AddressUnsafe, dstRegion, graphicsdriver.Region{}, [graphics.ShaderImageCount - 1][2]float32{}, nil, nil, false, true)
}
