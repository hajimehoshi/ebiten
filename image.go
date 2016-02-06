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
	"fmt"
	"image"
	"image/color"
	"runtime"

	"github.com/hajimehoshi/ebiten/internal/graphics"
	"github.com/hajimehoshi/ebiten/internal/graphics/opengl"
)

// Image represents an image.
// The pixel format is alpha-premultiplied.
// Image implements image.Image.
type Image struct {
	framebuffer *graphics.Framebuffer
	texture     *graphics.Texture
	pixels      []uint8
	width       int
	height      int
}

// Size returns the size of the image.
func (i *Image) Size() (width, height int) {
	if i.width == 0 {
		i.width, i.height = i.framebuffer.Size()
	}
	return i.width, i.height
}

// Clear resets the pixels of the image into 0.
func (i *Image) Clear() (err error) {
	if i.isDisposed() {
		return errors.New("image is already disposed")
	}
	return i.Fill(color.Transparent)
}

// Fill fills the image with a solid color.
func (i *Image) Fill(clr color.Color) (err error) {
	if i.isDisposed() {
		return errors.New("image is already disposed")
	}
	i.pixels = nil
	useGLContext(func(c *opengl.Context) {
		err = i.framebuffer.Fill(c, clr)
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
	if i.isDisposed() {
		return errors.New("image is already disposed")
	}
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
	useGLContext(func(c *opengl.Context) {
		err = i.framebuffer.DrawTexture(c, image.texture, quads, &options.GeoM, &options.ColorM)
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
	if i.isDisposed() {
		return color.Transparent
	}
	if i.pixels == nil {
		useGLContext(func(c *opengl.Context) {
			var err error
			i.pixels, err = i.framebuffer.Pixels(c)
			if err != nil {
				panic(err)
			}
		})
	}
	w, _ := i.Size()
	w = graphics.NextPowerOf2Int(w)
	idx := 4*x + 4*y*w
	r, g, b, a := i.pixels[idx], i.pixels[idx+1], i.pixels[idx+2], i.pixels[idx+3]
	return color.RGBA{r, g, b, a}
}

// Dispose disposes the image data. After disposing, the image becomes invalid.
// This is useful to save memory.
//
// The behavior of any functions for a disposed image is undefined.
func (i *Image) Dispose() error {
	if i.isDisposed() {
		return errors.New("image is already disposed")
	}
	useGLContext(func(c *opengl.Context) {
		i.framebuffer.Dispose(c)
		i.framebuffer = nil
		i.texture.Dispose(c)
		i.texture = nil
	})
	i.pixels = nil
	runtime.SetFinalizer(i, nil)
	return nil
}

func (i *Image) isDisposed() bool {
	// i.texture can be nil even when the image is not disposed,
	// so we need to check if both are nil.
	// See graphicsContext.setSize function.
	return i.texture == nil && i.framebuffer == nil
}

// ReplacePixels replaces the pixels of the image with p.
//
// The given p must represent RGBA pre-multiplied alpha values. len(p) must equal to 4 * (image width) * (image height).
//
// This function may be slow (as for implementation, this calls glTexSubImage2D).
func (i *Image) ReplacePixels(p []uint8) error {
	if i.isDisposed() {
		return errors.New("image is already disposed")
	}
	// Don't set i.pixels here because i.pixels is used not every time.
	i.pixels = nil
	w, h := i.Size()
	l := 4 * w * h
	if len(p) != l {
		return errors.New(fmt.Sprintf("p's length must be %d", l))
	}
	var err error
	useGLContext(func(c *opengl.Context) {
		err = i.texture.ReplacePixels(c, p)
	})
	return err
}

// A DrawImageOptions represents options to render an image on an image.
type DrawImageOptions struct {
	ImageParts ImageParts
	GeoM       GeoM
	ColorM     ColorM

	// Deprecated (as of 1.1.0-alpha): Use ImageParts instead.
	Parts []ImagePart
}

// NewImage returns an empty image.
//
// NewImage generates a new texture and a new framebuffer.
// Be careful that image objects will never be released
// even though nothing refers the image object and GC works.
// It is because there is no way to define finalizers for Go objects if you use GopherJS.
func NewImage(width, height int, filter Filter) (*Image, error) {
	var img *Image
	var err error
	useGLContext(func(c *opengl.Context) {
		var texture *graphics.Texture
		var framebuffer *graphics.Framebuffer
		texture, err = graphics.NewTexture(c, width, height, glFilter(c, filter))
		if err != nil {
			return
		}
		framebuffer, err = graphics.NewFramebufferFromTexture(c, texture)
		if err != nil {
			return
		}
		img = &Image{framebuffer: framebuffer, texture: texture}
	})
	if err != nil {
		return nil, err
	}
	if err := img.Clear(); err != nil {
		return nil, err
	}
	runtime.SetFinalizer(img, (*Image).Dispose)
	return img, nil
}

// NewImageFromImage creates a new image with the given image (img).
//
// NewImageFromImage generates a new texture and a new framebuffer.
// Be careful that image objects will never be released
// even though nothing refers the image object and GC works.
// It is because there is no way to define finalizers for Go objects if you use GopherJS.
func NewImageFromImage(img image.Image, filter Filter) (*Image, error) {
	var eimg *Image
	var err error
	useGLContext(func(c *opengl.Context) {
		var texture *graphics.Texture
		var framebuffer *graphics.Framebuffer
		texture, err = graphics.NewTextureFromImage(c, img, glFilter(c, filter))
		if err != nil {
			return
		}
		framebuffer, err = graphics.NewFramebufferFromTexture(c, texture)
		if err != nil {
			return
		}
		eimg = &Image{framebuffer: framebuffer, texture: texture}
	})
	if err != nil {
		return nil, err
	}
	runtime.SetFinalizer(eimg, (*Image).Dispose)
	return eimg, nil
}
