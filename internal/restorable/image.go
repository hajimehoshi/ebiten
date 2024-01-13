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
	srcs := [graphics.ShaderImageCount]*graphicscommand.Image{i.Image}
	sw, sh := i.Image.InternalSize()
	vs := quadVertices(0, 0, float32(sw), float32(sh), 0, 0, float32(sw), float32(sh), 1, 1, 1, 1)
	is := graphics.QuadIndices()
	dr := image.Rect(0, 0, sw, sh)
	newImg.Image.DrawTriangles(srcs, vs, is, graphicsdriver.BlendCopy, dr, [graphics.ShaderImageCount]image.Rectangle{}, NearestFilterShader.Shader, nil, graphicsdriver.FillAll)
	i.Image.Dispose()
	i.Image = nil

	return newImg
}

// quadVertices returns vertices to render a quad. These values are passed to graphicscommand.Image.
func quadVertices(dx0, dy0, dx1, dy1, sx0, sy0, sx1, sy1, cr, cg, cb, ca float32) []float32 {
	return []float32{
		dx0, dy0, sx0, sy0, cr, cg, cb, ca,
		dx1, dy0, sx1, sy0, cr, cg, cb, ca,
		dx0, dy1, sx0, sy1, cr, cg, cb, ca,
		dx1, dy1, sx1, sy1, cr, cg, cb, ca,
	}
}

func clearImage(i *graphicscommand.Image, region image.Rectangle) {
	vs := quadVertices(float32(region.Min.X), float32(region.Min.Y), float32(region.Max.X), float32(region.Max.Y), 0, 0, 0, 0, 0, 0, 0, 0)
	is := graphics.QuadIndices()
	i.DrawTriangles([graphics.ShaderImageCount]*graphicscommand.Image{}, vs, is, graphicsdriver.BlendClear, region, [graphics.ShaderImageCount]image.Rectangle{}, clearShader.Shader, nil, graphicsdriver.FillAll)
}

// ClearPixels clears the specified region by WritePixels.
func (i *Image) ClearPixels(region image.Rectangle) {
	if region.Dx() <= 0 || region.Dy() <= 0 {
		panic("restorable: width/height must be positive")
	}
	clearImage(i.Image, region.Intersect(image.Rect(0, 0, i.width, i.height)))
}
