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

// TODO: Remove this in the future.
type imageParts []ImagePart

func (p imageParts) Len() int {
	return len(p)
}

func (p imageParts) Dst(i int) (x0, y0, x1, y1 int) {
	dst := &p[i].Dst
	return dst.Min.X, dst.Min.Y, dst.Max.X, dst.Max.Y
}

func (p imageParts) Src(i int) (x0, y0, x1, y1 int) {
	src := &p[i].Src
	return src.Min.X, src.Min.Y, src.Max.X, src.Max.Y
}

type wholeImage struct {
	width  int
	height int
}

func (w *wholeImage) Len() int {
	return 1
}

func (w *wholeImage) Dst(i int) (x0, y0, x1, y1 int) {
	return 0, 0, w.width, w.height
}

func (w *wholeImage) Src(i int) (x0, y0, x1, y1 int) {
	return 0, 0, w.width, w.height
}

func (i *innerImage) drawImage(c *opengl.Context, img *innerImage, options *DrawImageOptions) error {
	if options == nil {
		options = &DrawImageOptions{}
	}
	parts := options.ImageParts
	if parts == nil {
		dparts := options.Parts
		if dparts != nil {
			parts = imageParts(dparts)
		} else {
			w, h := img.size()
			parts = &wholeImage{w, h}
		}
	}
	w, h := img.size()
	quads := &textureQuads{parts: parts, width: w, height: h}
	return i.framebuffer.DrawTexture(c, img.texture, quads, &options.GeoM, &options.ColorM)
}

func u(x float32, width int) float32 {
	return x / float32(internal.NextPowerOf2Int(width))
}

func v(y float32, height int) float32 {
	return y / float32(internal.NextPowerOf2Int(height))
}

type textureQuads struct {
	parts  ImageParts
	width  int
	height int
}

func (t *textureQuads) Len() int {
	return t.parts.Len()
}

func (t *textureQuads) Vertex(i int) (x0, y0, x1, y1 float32) {
	ix0, iy0, ix1, iy1 := t.parts.Dst(i)
	return float32(ix0), float32(iy0), float32(ix1), float32(iy1)
}

func (t *textureQuads) Texture(i int) (u0, v0, u1, v1 float32) {
	x0, y0, x1, y1 := t.parts.Src(i)
	w, h := t.width, t.height
	return u(float32(x0), w), v(float32(y0), h), u(float32(x1), w), v(float32(y1), h)
}

// Image represents an image.
// The pixel format is alpha-premultiplied.
// Image implements image.Image.
type Image struct {
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
	ui.Use(func(c *opengl.Context) {
		err = i.inner.Clear(c)
	})
	return
}

// Fill fills the image with a solid color.
func (i *Image) Fill(clr color.Color) (err error) {
	i.pixels = nil
	ui.Use(func(c *opengl.Context) {
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
//     ImageParts: (0, 0) - (source width, source height) to (0, 0) - (source width, source height)
//                 (i.e. the whole source image)
//     GeoM:       Identity matrix
//     ColorM:     Identity matrix (that changes no colors)
//
// Be careful that this method is potentially slow.
// It would be better if you could call this method fewer times.
func (i *Image) DrawImage(image *Image, options *DrawImageOptions) (err error) {
	return i.drawImage(image.inner, options)
}

func (i *Image) drawImage(image *innerImage, option *DrawImageOptions) (err error) {
	i.pixels = nil
	ui.Use(func(c *opengl.Context) {
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
		ui.Use(func(c *opengl.Context) {
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

// Deprecated (as of 1.1.0-alpha): Use ImageParts instead.
type ImagePart struct {
	Dst image.Rectangle
	Src image.Rectangle
}

// An ImageParts represents the parts of the destination image and the parts of the source image.
type ImageParts interface {
	Len() int
	Dst(i int) (x0, y0, x1, y1 int)
	Src(i int) (x0, y0, x1, y1 int)
}

// A DrawImageOptions represents options to render an image on an image.
type DrawImageOptions struct {
	ImageParts ImageParts
	GeoM       GeoM
	ColorM     ColorM

	// Deprecated (as of 1.1.0-alpha): Use ImageParts instead.
	Parts []ImagePart
}
