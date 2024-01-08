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

type Image struct {
	img    *atlas.Image
	width  int
	height int

	// pixels is valid only when restorable.AlwaysReadPixelsFromGPU() returns true.
	pixels []byte
}

func NewImage(width, height int, imageType atlas.ImageType) *Image {
	return &Image{
		width:  width,
		height: height,
		img:    atlas.NewImage(width, height, imageType),
	}
}

func (i *Image) invalidatePixels() {
	i.pixels = nil
}

func (i *Image) Deallocate() {
	i.img.Deallocate()
}

func (i *Image) ReadPixels(graphicsDriver graphicsdriver.Graphics, pixels []byte, region image.Rectangle) error {
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
	return nil
}

func (i *Image) DumpScreenshot(graphicsDriver graphicsdriver.Graphics, name string, blackbg bool) (string, error) {
	return i.img.DumpScreenshot(graphicsDriver, name, blackbg)
}

// WritePixels replaces the pixels at the specified region.
func (i *Image) WritePixels(pix []byte, region image.Rectangle) {
	if l := 4 * region.Dx() * region.Dy(); len(pix) != l {
		panic(fmt.Sprintf("buffered: len(pix) was %d but must be %d", len(pix), l))
	}
	i.invalidatePixels()

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
	}
	i.invalidatePixels()

	var imgs [graphics.ShaderImageCount]*atlas.Image
	for i, img := range srcs {
		if img == nil {
			continue
		}
		imgs[i] = img.img
	}

	i.img.DrawTriangles(imgs, vertices, indices, blend, dstRegion, srcRegions, shader, uniforms, fillRule)
}
