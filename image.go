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
	"sync"

	"github.com/hajimehoshi/ebiten/internal/graphics"
	"github.com/hajimehoshi/ebiten/internal/graphics/opengl"
)

var imageM sync.Mutex

// Image represents an image.
// The pixel format is alpha-premultiplied.
// Image implements image.Image.
//
// Note that manipulating an Image is NOT concurrent-safe (so far).
type Image struct {
	framebuffer *graphics.Framebuffer
	texture     *graphics.Texture
	pixels      []uint8
	width       int
	height      int
}

// Size returns the size of the image.
func (i *Image) Size() (width, height int) {
	return i.width, i.height
}

// Clear resets the pixels of the image into 0.
func (i *Image) Clear() (err error) {
	imageM.Lock()
	defer imageM.Unlock()
	return i.clear()
}

func (i *Image) clear() (err error) {
	return i.fill(color.Transparent)
}

// Fill fills the image with a solid color.
func (i *Image) Fill(clr color.Color) (err error) {
	imageM.Lock()
	defer imageM.Unlock()
	return i.fill(clr)
}

func (i *Image) fill(clr color.Color) (err error) {
	if i.isDisposed() {
		return errors.New("ebiten: image is already disposed")
	}
	i.pixels = nil
	return i.framebuffer.Fill(glContext, clr)
}

// DrawImage draws the given image on the receiver image.
//
// This method accepts the options.
// The parts of the given image at the parts of the destination.
// After determining parts to draw, this applies the geometry matrix and the color matrix.
//
// Here are the default values:
//     ImageParts:    (0, 0) - (source width, source height) to (0, 0) - (source width, source height)
//                    (i.e. the whole source image)
//     GeoM:          Identity matrix
//     ColorM:        Identity matrix (that changes no colors)
//     CompositeMode: CompositeModeSourceOver (regular alpha blending)
//
// Be careful that this method is potentially slow.
// It would be better if you could call this method fewer times.
func (i *Image) DrawImage(image *Image, options *DrawImageOptions) (err error) {
	// Calculate vertices before locking because the user can do anything in
	// options.ImageParts interface without deadlock (e.g. Call Image functions).
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
			parts = &wholeImage{image.width, image.height}
		}
	}
	quads := &textureQuads{parts: parts, width: image.width, height: image.height}
	// TODO: Reuse one vertices instead of making here, but this would need locking.
	vertices := make([]int16, parts.Len()*16)
	n := quads.vertices(vertices)
	if n == 0 {
		return nil
	}

	imageM.Lock()
	defer imageM.Unlock()
	if i.isDisposed() {
		return errors.New("ebiten: image is already disposed")
	}
	if i == image {
		return errors.New("ebiten: Image.DrawImage: image should be different from the receiver")
	}
	i.pixels = nil
	m := opengl.CompositeMode(options.CompositeMode)
	return i.framebuffer.DrawTexture(glContext, image.texture, vertices[:16*n], &options.GeoM, &options.ColorM, m)
}

// Bounds returns the bounds of the image.
func (i *Image) Bounds() image.Rectangle {
	return image.Rect(0, 0, i.width, i.height)
}

// ColorModel returns the color model of the image.
func (i *Image) ColorModel() color.Model {
	return color.RGBAModel
}

// At returns the color of the image at (x, y).
//
// This method loads pixels from VRAM to system memory if necessary.
func (i *Image) At(x, y int) color.Color {
	// TODO: What if At is called internaly (like from image parts?)
	imageM.Lock()
	defer imageM.Unlock()
	if i.isDisposed() {
		return color.Transparent
	}
	if i.pixels == nil {
		var err error
		i.pixels, err = i.framebuffer.Pixels(glContext)
		if err != nil {
			panic(err)
		}
	}
	w := int(graphics.NextPowerOf2Int32(int32(i.width)))
	idx := 4*x + 4*y*w
	r, g, b, a := i.pixels[idx], i.pixels[idx+1], i.pixels[idx+2], i.pixels[idx+3]
	return color.RGBA{r, g, b, a}
}

// Dispose disposes the image data. After disposing, the image becomes invalid.
// This is useful to save memory.
//
// The behavior of any functions for a disposed image is undefined.
func (i *Image) Dispose() error {
	imageM.Lock()
	defer imageM.Unlock()
	if i.isDisposed() {
		return errors.New("ebiten: image is already disposed")
	}
	if i.framebuffer != nil {
		i.framebuffer.Dispose(glContext)
		i.framebuffer = nil
	}
	if i.texture != nil {
		i.texture.Dispose(glContext)
		i.texture = nil
	}
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
	imageM.Lock()
	defer imageM.Unlock()
	if i.isDisposed() {
		return errors.New("ebiten: image is already disposed")
	}
	// Don't set i.pixels here because i.pixels is used not every time.
	i.pixels = nil
	l := 4 * i.width * i.height
	if len(p) != l {
		return fmt.Errorf("ebiten: p's length must be %d", l)
	}
	return i.framebuffer.ReplacePixels(glContext, i.texture, p)
}

// A DrawImageOptions represents options to render an image on an image.
type DrawImageOptions struct {
	ImageParts    ImageParts
	GeoM          GeoM
	ColorM        ColorM
	CompositeMode CompositeMode

	// Deprecated (as of 1.1.0-alpha): Use ImageParts instead.
	Parts []ImagePart
}

// NewImage returns an empty image.
//
// NewImage generates a new texture and a new framebuffer.
// Be careful that image objects will never be released
// even though nothing refers the image object and GC works.
// It is because there is no way to define finalizers for Go objects if you use GopherJS.
//
// This function is concurrent-safe.
func NewImage(width, height int, filter Filter) (*Image, error) {
	imageM.Lock()
	defer imageM.Unlock()

	texture, err := graphics.NewTexture(glContext, width, height, glFilter(glContext, filter))
	if err != nil {
		return nil, err
	}
	framebuffer, err := graphics.NewFramebufferFromTexture(glContext, texture)
	if err != nil {
		// TODO: texture should be removed here?
		return nil, err
	}
	img := &Image{
		framebuffer: framebuffer,
		texture:     texture,
		width:       width,
		height:      height,
	}
	runtime.SetFinalizer(img, (*Image).Dispose)
	if err := img.clear(); err != nil {
		return nil, err
	}
	return img, nil
}

// NewImageFromImage creates a new image with the given image (img).
//
// NewImageFromImage generates a new texture and a new framebuffer.
// Be careful that image objects will never be released
// even though nothing refers the image object and GC works.
// It is because there is no way to define finalizers for Go objects if you use GopherJS.
//
// This function is concurrent-safe.
func NewImageFromImage(img image.Image, filter Filter) (*Image, error) {
	imageM.Lock()
	defer imageM.Unlock()

	texture, err := graphics.NewTextureFromImage(glContext, img, glFilter(glContext, filter))
	if err != nil {
		return nil, err
	}
	framebuffer, err := graphics.NewFramebufferFromTexture(glContext, texture)
	if err != nil {
		// TODO: texture should be removed here?
		return nil, err
	}
	w, h := framebuffer.Size()
	eimg := &Image{
		framebuffer: framebuffer,
		texture:     texture,
		width:       w,
		height:      h,
	}
	runtime.SetFinalizer(eimg, (*Image).Dispose)
	return eimg, nil
}
