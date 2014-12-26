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
	"github.com/hajimehoshi/ebiten/internal/opengl"
	"github.com/hajimehoshi/ebiten/internal/opengl/internal/shader"
	"image"
	"image/color"
)

type innerImage struct {
	framebuffer *opengl.Framebuffer
	texture     *opengl.Texture
}

func newInnerImage(texture *opengl.Texture) (*innerImage, error) {
	framebuffer, err := opengl.NewFramebufferFromTexture(texture)
	if err != nil {
		return nil, err
	}
	return &innerImage{framebuffer, texture}, nil
}

func (i *innerImage) size() (width, height int) {
	return i.framebuffer.Size()
}

func (i *innerImage) Clear() error {
	return i.Fill(color.Transparent)
}

func (i *innerImage) Fill(clr color.Color) error {
	if err := i.framebuffer.SetAsViewport(); err != nil {
		return err
	}
	r, g, b, a := internal.RGBA(clr)
	opengl.Clear(r, g, b, a)
	return nil
}

func (i *innerImage) drawImage(img *innerImage, options *DrawImageOptions) error {
	if options == nil {
		options = &DrawImageOptions{}
	}
	dsts := options.DstParts
	srcs := options.SrcParts
	if srcs == nil || dsts == nil {
		w, h := img.size()
		dsts = []image.Rectangle{
			image.Rect(0, 0, w, h),
		}
		srcs = []image.Rectangle{
			image.Rect(0, 0, w, h),
		}
	}
	geo := options.GeometryMatrix
	clr := options.ColorMatrix

	if err := i.framebuffer.SetAsViewport(); err != nil {
		return err
	}
	w, h := img.texture.Size()
	quads := textureQuads(dsts, srcs, w, h)
	projectionMatrix := i.framebuffer.ProjectionMatrix()
	shader.DrawTexture(img.texture.Native(), projectionMatrix, quads, geo, clr)
	return nil
}

func u(x float64, width int) float32 {
	return float32(x) / float32(internal.NextPowerOf2Int(width))
}

func v(y float64, height int) float32 {
	return float32(y) / float32(internal.NextPowerOf2Int(height))
}

func textureQuads(dsts, srcs []image.Rectangle, width, height int) []shader.TextureQuad {
	l := len(dsts)
	if len(srcs) < l {
		l = len(srcs)
	}
	quads := make([]shader.TextureQuad, 0, l)
	for i := 0; i < l; i++ {
		dst, src := dsts[i], srcs[i]
		x1 := float32(dst.Min.X)
		x2 := float32(dst.Max.X)
		y1 := float32(dst.Min.Y)
		y2 := float32(dst.Max.Y)
		u1 := u(float64(src.Min.X), width)
		u2 := u(float64(src.Max.X), width)
		v1 := v(float64(src.Min.Y), height)
		v2 := v(float64(src.Max.Y), height)
		quad := shader.TextureQuad{x1, x2, y1, y2, u1, u2, v1, v2}
		quads = append(quads, quad)
	}
	return quads
}

type syncer interface {
	Sync(func())
}

// Image represents an image.
// The pixel format is alpha-premultiplied.
// Image implements image.Image.
type Image struct {
	syncer syncer
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
	i.syncer.Sync(func() {
		err = i.inner.Clear()
	})
	return
}

// Fill fills the image with a solid color.
func (i *Image) Fill(clr color.Color) (err error) {
	i.pixels = nil
	i.syncer.Sync(func() {
		err = i.inner.Fill(clr)
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
//     DstParts:       (0, 0) - (source width, source height)
//     SrcParts:       (0, 0) - (source width, source height) (i.e. the whole source image)
//     GeometryMatrix: Identity matrix
//     ColorMatrix:    Identity matrix (that changes no colors)
func (i *Image) DrawImage(image *Image, options *DrawImageOptions) (err error) {
	return i.drawImage(image.inner, options)
}

// DrawImageAt draws the given image on the receiver image at the position (x, y).
//
// If a geometry matrix is specified, the geometry matrix is applied ahead of translating the image by (x, y).
func (i *Image) DrawImageAt(image *Image, x, y int, options *DrawImageOptions) (err error) {
	if options == nil {
		options = &DrawImageOptions{}
	}
	options.GeometryMatrix.Concat(TranslateGeometry(float64(x), float64(y)))
	return i.drawImage(image.inner, options)
}

func (i *Image) drawImage(image *innerImage, option *DrawImageOptions) (err error) {
	i.pixels = nil
	i.syncer.Sync(func() {
		err = i.inner.drawImage(image, option)
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
		i.syncer.Sync(func() {
			var err error
			i.pixels, err = i.inner.texture.Pixels()
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

// A DrawImageOptions presents options to render an image on an image.
type DrawImageOptions struct {
	DstParts       []image.Rectangle
	SrcParts       []image.Rectangle
	GeometryMatrix GeometryMatrix
	ColorMatrix    ColorMatrix
}
