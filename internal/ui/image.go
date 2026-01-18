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
	"image"

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
	ui *UserInterface

	mipmap    *mipmap.Mipmap
	width     int
	height    int
	imageType atlas.ImageType

	// lastBlend is the lastly-used blend for mipmap.Image.
	lastBlend graphicsdriver.Blend

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
	i.mipmap.Deallocate()
}

func (i *Image) DrawTriangles(srcs [graphics.ShaderSrcImageCount]*Image, vertices []float32, indices []uint32, blend graphicsdriver.Blend, dstRegion image.Rectangle, srcRegions [graphics.ShaderSrcImageCount]image.Rectangle, shader *Shader, uniforms []uint32, canSkipMipmap bool) {
	if i.modifyCallback != nil {
		i.modifyCallback()
	}

	i.lastBlend = blend

	var srcMipmaps [graphics.ShaderSrcImageCount]*mipmap.Mipmap
	for i, src := range srcs {
		if src == nil {
			continue
		}
		srcMipmaps[i] = src.mipmap
	}

	i.mipmap.DrawTriangles(srcMipmaps, vertices, indices, blend, dstRegion, srcRegions, shader.shader, uniforms, canSkipMipmap)
}

func (i *Image) WritePixels(pix []byte, region image.Rectangle) {
	if i.modifyCallback != nil {
		i.modifyCallback()
	}
	i.mipmap.WritePixels(pix, region)
}

func (i *Image) ReadPixels(pixels []byte, region image.Rectangle) {
	// Check the error existence and avoid unnecessary calls.
	if i.ui.error() != nil {
		return
	}

	if err := i.ui.readPixels(i.mipmap, pixels, region); err != nil {
		if panicOnErrorOnReadingPixels {
			panic(err)
		}
		i.ui.setError(err)
	}
}

func (i *Image) DumpScreenshot(name string, blackbg bool) (string, error) {
	return i.ui.dumpScreenshot(i.mipmap, name, blackbg)
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
	i.DrawTriangles(srcs, i.tmpVerticesForFill, is, blend, region, [graphics.ShaderSrcImageCount]image.Rectangle{sr}, NearestFilterShader, nil, true)
}
