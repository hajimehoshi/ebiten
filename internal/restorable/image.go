// Copyright 2016 The Ebiten Authors
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

package restorable

import (
	"fmt"
	"image"

	"github.com/hajimehoshi/ebiten/v2/internal/graphics"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicscommand"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
)

// Image represents an image.
type Image struct {
	// Image is the underlying image.
	// This member is exported on purpose.
	// TODO: Move the implementation to internal/atlas package (#805).
	Image *graphicscommand.Image

	width  int
	height int
}

// NewImage creates an emtpy image with the given size.
//
// The returned image is cleared.
//
// Note that Dispose is not called automatically.
func NewImage(width, height int, screen bool) *Image {
	i := &Image{
		Image:  graphicscommand.NewImage(width, height, screen),
		width:  width,
		height: height,
	}

	// This needs to use 'InternalSize' to render the whole region, or edges are unexpectedly cleared on some
	// devices.
	iw, ih := i.Image.InternalSize()
	clearImage(i.Image, image.Rect(0, 0, iw, ih))
	return i
}

// Extend extends the image by the given size.
// Extend creates a new image with the given size and copies the pixels of the given source image.
// Extend disposes itself after its call.
func (i *Image) Extend(width, height int) *Image {
	if i.width >= width && i.height >= height {
		return i
	}

	// Assume that the screen image is never extended.
	newImg := NewImage(width, height, false)

	// Use DrawTriangles instead of WritePixels because the image i might be stale and not have its pixels
	// information.
	srcs := [graphics.ShaderSrcImageCount]*Image{i}
	sw, sh := i.Image.InternalSize()
	vs := make([]float32, 4*graphics.VertexFloatCount)
	graphics.QuadVerticesFromDstAndSrc(vs, 0, 0, float32(sw), float32(sh), 0, 0, float32(sw), float32(sh), 1, 1, 1, 1)
	is := graphics.QuadIndices()
	dr := image.Rect(0, 0, sw, sh)
	newImg.DrawTriangles(srcs, vs, is, graphicsdriver.BlendCopy, dr, [graphics.ShaderSrcImageCount]image.Rectangle{}, NearestFilterShader, nil, graphicsdriver.FillRuleFillAll)
	i.Image.Dispose()
	i.Image = nil

	return newImg
}

func clearImage(i *graphicscommand.Image, region image.Rectangle) {
	vs := make([]float32, 4*graphics.VertexFloatCount)
	graphics.QuadVerticesFromDstAndSrc(vs, float32(region.Min.X), float32(region.Min.Y), float32(region.Max.X), float32(region.Max.Y), 0, 0, 0, 0, 0, 0, 0, 0)
	is := graphics.QuadIndices()
	i.DrawTriangles([graphics.ShaderSrcImageCount]*graphicscommand.Image{}, vs, is, graphicsdriver.BlendClear, region, [graphics.ShaderSrcImageCount]image.Rectangle{}, clearShader.shader, nil, graphicsdriver.FillRuleFillAll)
}

// ClearPixels clears the specified region by WritePixels.
func (i *Image) ClearPixels(region image.Rectangle) {
	i.WritePixels(nil, region)
}

// WritePixels replaces the image pixels with the given pixels slice.
//
// The specified region must not be overlapped with other regions by WritePixels.
func (i *Image) WritePixels(pixels *graphics.ManagedBytes, region image.Rectangle) {
	if region.Dx() <= 0 || region.Dy() <= 0 {
		panic("restorable: width/height must be positive")
	}
	w, h := i.width, i.height
	if !region.In(image.Rect(0, 0, w, h)) {
		panic(fmt.Sprintf("restorable: out of range %v", region))
	}

	if pixels != nil {
		i.Image.WritePixels(pixels, region)
	} else {
		clearImage(i.Image, region)
	}
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
func (i *Image) DrawTriangles(srcs [graphics.ShaderSrcImageCount]*Image, vertices []float32, indices []uint32, blend graphicsdriver.Blend, dstRegion image.Rectangle, srcRegions [graphics.ShaderSrcImageCount]image.Rectangle, shader *Shader, uniforms []uint32, fillRule graphicsdriver.FillRule) {
	if len(vertices) == 0 {
		return
	}

	var imgs [graphics.ShaderSrcImageCount]*graphicscommand.Image
	for i, src := range srcs {
		if src == nil {
			continue
		}
		imgs[i] = src.Image
	}
	i.Image.DrawTriangles(imgs, vertices, indices, blend, dstRegion, srcRegions, shader.shader, uniforms, fillRule)
}

func (i *Image) ReadPixels(graphicsDriver graphicsdriver.Graphics, pixels []byte, region image.Rectangle) error {
	if err := i.Image.ReadPixels(graphicsDriver, []graphicsdriver.PixelsArgs{
		{
			Pixels: pixels,
			Region: region,
		},
	}); err != nil {
		return err
	}
	return nil
}
