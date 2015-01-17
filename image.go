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
	"errors"
	"github.com/hajimehoshi/ebiten/internal"
	"github.com/hajimehoshi/ebiten/internal/graphics"
	"github.com/hajimehoshi/ebiten/internal/opengl"
	"github.com/hajimehoshi/ebiten/internal/ui"
	"image"
	"image/color"
)

// Image represents an image.
// The pixel format is alpha-premultiplied.
// Image implements image.Image.
type Image struct {
	framebuffer *graphics.Framebuffer
	texture     *graphics.Texture
	pixels      []uint8
}

// Size returns the size of the image.
func (i *Image) Size() (width, height int) {
	return i.framebuffer.Size()
}

// Clear resets the pixels of the image into 0.
func (i *Image) Clear() (err error) {
	return i.Fill(color.Transparent)
}

// Fill fills the image with a solid color.
func (i *Image) Fill(clr color.Color) (err error) {
	i.pixels = nil
	r, g, b, a := internal.RGBA(clr)
	ui.Use(func(c *opengl.Context) {
		// TODO: Change to pass color.Color
		err = i.framebuffer.Fill(c, r, g, b, a)
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
	if i == image {
		return errors.New("Image.DrawImage: image should be different from the receiver")
	}
	i.pixels = nil
	if options == nil {
		options = &DrawImageOptions{}
	}
	parts := options.ImageParts
	if parts == nil {
		// Check options.Parts for backward-compatibility.
		dparts := options.Parts
		if dparts != nil {
			parts = imageParts(dparts)
		} else {
			w, h := image.Size()
			parts = &wholeImage{w, h}
		}
	}
	w, h := image.Size()
	quads := &textureQuads{parts: parts, width: w, height: h}
	ui.Use(func(c *opengl.Context) {
		err = i.framebuffer.DrawTexture(c, image.texture, quads, &options.GeoM, &options.ColorM)
	})
	return
}

// DrawLine draws a line.
func (i *Image) DrawLine(x0, y0, x1, y1 int, clr color.Color) error {
	return i.DrawLines(&line{x0, y0, x1, y1, clr})
}

// DrawLines draws lines.
func (i *Image) DrawLines(lines Lines) (err error) {
	ui.Use(func(c *opengl.Context) {
		err = i.framebuffer.DrawLines(c, lines)
	})
	return
}

// FillRect draws a filled rectangle.
func (i *Image) FillRect(x, y, width, height int, clr color.Color) error {
	return i.FillRects(&rect{x, y, width, height, clr})
}

// FillRects draws filled rectangles on the image.
func (i *Image) FillRects(rects Rects) (err error) {
	ui.Use(func(c *opengl.Context) {
		err = i.framebuffer.FillRects(c, rects)
	})
	return
}

// Bounds returns the bounds of the image.
func (i *Image) Bounds() image.Rectangle {
	w, h := i.Size()
	return image.Rect(0, 0, w, h)
}

// ColorModel returns the color model of the image.
func (i *Image) ColorModel() color.Model {
	return color.RGBAModel
}

// At returns the color of the image at (x, y).
//
// This method loads pixels from VRAM to system memory if necessary.
func (i *Image) At(x, y int) color.Color {
	if i.pixels == nil {
		ui.Use(func(c *opengl.Context) {
			var err error
			i.pixels, err = i.texture.Pixels(c)
			if err != nil {
				panic(err)
			}
		})
	}
	w, _ := i.Size()
	w = internal.NextPowerOf2Int(w)
	idx := 4*x + 4*y*w
	r, g, b, a := i.pixels[idx], i.pixels[idx+1], i.pixels[idx+2], i.pixels[idx+3]
	return color.RGBA{r, g, b, a}
}

// A DrawImageOptions represents options to render an image on an image.
type DrawImageOptions struct {
	ImageParts ImageParts
	GeoM       GeoM
	ColorM     ColorM

	// Deprecated (as of 1.1.0-alpha): Use ImageParts instead.
	Parts []ImagePart
}
