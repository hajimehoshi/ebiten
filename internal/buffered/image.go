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

	// cache is the CPU-side cache of the pixel data.
	cache pixelCache
}

func NewImage(width, height int, imageType atlas.ImageType) *Image {
	i := &Image{
		img:    atlas.NewImage(width, height, imageType),
		width:  width,
		height: height,
	}
	i.cache.init(width, height)
	return i
}

func (i *Image) Deallocate() {
	i.img.Deallocate()
	i.cache.deallocate()
}

func (i *Image) ReadPixels(graphicsDriver graphicsdriver.Graphics, pixels []byte, region image.Rectangle) (bool, error) {
	return i.cache.readPixels(i.img, graphicsDriver, pixels, region)
}

func (i *Image) DumpScreenshot(graphicsDriver graphicsdriver.Graphics, name string, blackbg bool) (string, error) {
	i.writeBackPixelsIfNeeded()
	return i.img.DumpScreenshot(graphicsDriver, name, blackbg)
}

// WritePixels replaces the pixels at the specified region.
func (i *Image) WritePixels(pix []byte, region image.Rectangle) {
	if l := 4 * region.Dx() * region.Dy(); len(pix) != l {
		panic(fmt.Sprintf("buffered: len(pix) was %d but must be %d", len(pix), l))
	}
	i.cache.writePixels(i.img, pix, region)
}

// DrawTriangles draws the src image with the given vertices.
//
// Copying vertices and indices is the caller's responsibility.
func (i *Image) DrawTriangles(srcs [graphics.ShaderSrcImageCount]*Image, vertices []float32, indices []uint32, blend graphicsdriver.Blend, dstRegion image.Rectangle, srcRegions [graphics.ShaderSrcImageCount]image.Rectangle, shader *atlas.Shader, uniforms []uint32) {
	for _, src := range srcs {
		if i == src {
			panic("buffered: Image.DrawTriangles: source images must be different from the receiver")
		}
		if src != nil {
			// src's pending pixel writes have to be applied to the GPU,
			// but the cache doesn't have to be cleared since src is not modified in this function.
			src.writeBackPixelsIfNeeded()
		}
	}

	i.writeBackPixelsIfNeeded()

	var imgs [graphics.ShaderSrcImageCount]*atlas.Image
	for i, img := range srcs {
		if img == nil {
			continue
		}
		imgs[i] = img.img
	}

	i.img.DrawTriangles(imgs, vertices, indices, blend, dstRegion, srcRegions, shader, uniforms)

	// After rendering, the pixel cache is no longer valid.
	i.cache.reset()
}

// writeBackPixelsIfNeeded applies the pending pixel writes to the GPU.
func (i *Image) writeBackPixelsIfNeeded() {
	i.cache.writeBackPixelsIfNeeded(i.img)
}
