// Copyright 2014 Hajime Hoshi
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

package ebiten

import (
	"github.com/hajimehoshi/ebiten/internal"
	"github.com/hajimehoshi/ebiten/internal/graphics"
	"github.com/hajimehoshi/ebiten/internal/opengl"
	"github.com/hajimehoshi/ebiten/internal/ui"
	"image"
	"image/color"
)

type innerImage struct {
	framebuffer *graphics.Framebuffer
	texture     *graphics.Texture
}

func newInnerImage(c *opengl.Context, texture *graphics.Texture) (*innerImage, error) {
	framebuffer, err := graphics.NewFramebufferFromTexture(c, texture)
	if err != nil {
		return nil, err
	}
	return &innerImage{framebuffer, texture}, nil
}

func (i *innerImage) size() (width, height int) {
	return i.framebuffer.Size()
}

func (i *innerImage) Clear(c *opengl.Context) error {
	return i.Fill(c, color.Transparent)
}

func (i *innerImage) Fill(c *opengl.Context, clr color.Color) error {
	r, g, b, a := internal.RGBA(clr)
	return i.framebuffer.Fill(c, r, g, b, a)
}

func (i *innerImage) drawImage(c *opengl.Context, img *innerImage, options *DrawImageOptions) error {
	if options == nil {
		options = &DrawImageOptions{}
	}
	parts := options.Parts
	if parts == nil {
		w, h := img.size()
		parts = []ImagePart{
			{
				Dst: image.Rect(0, 0, w, h),
				Src: image.Rect(0, 0, w, h),
			},
		}
	}
	geo := options.GeoM
	clr := options.ColorM
	w, h := img.size()
	quads := &textureQuads{parts, w, h}
	return i.framebuffer.DrawTexture(c, img.texture, quads, geo, clr)
}

func u(x float64, width int) float32 {
	return float32(x) / float32(internal.NextPowerOf2Int(width))
}

func v(y float64, height int) float32 {
	return float32(y) / float32(internal.NextPowerOf2Int(height))
}

type textureQuads struct {
	parts  []ImagePart
	width  int
	height int
}

func (t *textureQuads) Len() int {
	return len(t.parts)
}

func (t *textureQuads) Vertex(i int) (x0, y0, x1, y1 float32) {
	dst := t.parts[i].Dst
	return float32(dst.Min.X), float32(dst.Min.Y), float32(dst.Max.X), float32(dst.Max.Y)
}

func (t *textureQuads) Texture(i int) (u0, v0, u1, v1 float32) {
	src := t.parts[i].Src
	w, h := t.width, t.height
	return u(float64(src.Min.X), w), v(float64(src.Min.Y), h), u(float64(src.Max.X), w), v(float64(src.Max.Y), h)
}

// Image represents an image.
// The pixel format is alpha-premultiplied.
// Image implements image.Image.
type Image struct {
	ui     *ui.UI
	inner  *innerImage
	pixels []uint8
}

// Size returns the size of the image.
func (i *Image) Size() (width, height int) {
	return i.inner.size()
}

// Clear resets the pixels of the image into 0.
func (i *Image) Clear() (err error) {
	i.pixels = nil
	i.ui.Use(func(c *opengl.Context) {
		err = i.inner.Clear(c)
	})
	return
}

// Fill fills the image with a solid color.
func (i *Image) Fill(clr color.Color) (err error) {
	i.pixels = nil
	i.ui.Use(func(c *opengl.Context) {
		err = i.inner.Fill(c, clr)
	})
	return
}

// DrawImage draws the given image on the receiver image.
//
// This method accepts the options.
// The parts of the given image at the parts of the destination.
// After determining parts to draw, this applies the geometry matrix and the color matrix.
//
// Here are the default values:
//     Parts:  (0, 0) - (source width, source height) to (0, 0) - (source width, source height)
//             (i.e. the whole source image)
//     GeoM:   Identity matrix
//     ColorM: Identity matrix (that changes no colors)
func (i *Image) DrawImage(image *Image, options *DrawImageOptions) (err error) {
	return i.drawImage(image.inner, options)
}

func (i *Image) drawImage(image *innerImage, option *DrawImageOptions) (err error) {
	i.pixels = nil
	i.ui.Use(func(c *opengl.Context) {
		err = i.inner.drawImage(c, image, option)
	})
	return
}

// Bounds returns the bounds of the image.
func (i *Image) Bounds() image.Rectangle {
	w, h := i.inner.size()
	return image.Rect(0, 0, w, h)
}

// ColorModel returns the color model of the image.
func (i *Image) ColorModel() color.Model {
	return color.RGBAModel
}

// At returns the color of the image at (x, y).
//
// This method loads pixels from GPU to VRAM if necessary.
func (i *Image) At(x, y int) color.Color {
	if i.pixels == nil {
		i.ui.Use(func(c *opengl.Context) {
			var err error
			i.pixels, err = i.inner.texture.Pixels(c)
			if err != nil {
				panic(err)
			}
		})
	}
	w, _ := i.inner.size()
	w = internal.NextPowerOf2Int(w)
	idx := 4*x + 4*y*w
	r, g, b, a := i.pixels[idx], i.pixels[idx+1], i.pixels[idx+2], i.pixels[idx+3]
	return color.RGBA{r, g, b, a}
}

type ImagePart struct {
	Dst image.Rectangle
	Src image.Rectangle
}

// A DrawImageOptions represents options to render an image on an image.
type DrawImageOptions struct {
	Parts  []ImagePart
	GeoM   GeoM
	ColorM ColorM
}
