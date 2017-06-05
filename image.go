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
	"fmt"
	"image"
	"image/color"
	"runtime"

	"github.com/hajimehoshi/ebiten/internal/graphics"
	"github.com/hajimehoshi/ebiten/internal/opengl"
	"github.com/hajimehoshi/ebiten/internal/restorable"
)

// Image represents an image.
// The pixel format is alpha-premultiplied.
// Image implements image.Image.
//
// Functions of Image never returns error as of 1.5.0-alpha, and error values are always nil.
type Image struct {
	restorable *restorable.Image
}

// Size returns the size of the image.
func (i *Image) Size() (width, height int) {
	return i.restorable.Size()
}

// Clear resets the pixels of the image into 0.
//
// When the image is disposed, Clear does nothing.
//
// Clear always returns nil as of 1.5.0-alpha.
func (i *Image) Clear() error {
	i.restorable.Fill(color.RGBA{})
	return nil
}

// Fill fills the image with a solid color.
//
// When the image is disposed, Fill does nothing.
//
// Fill always returns nil as of 1.5.0-alpha.
func (i *Image) Fill(clr color.Color) error {
	rgba := color.RGBAModel.Convert(clr).(color.RGBA)
	i.restorable.Fill(rgba)
	return nil
}

// DrawImage draws the given image on the receiver image.
//
// This method accepts the options.
// The parts of the given image at the parts of the destination.
// After determining parts to draw, this applies the geometry matrix and the color matrix.
//
// Here are the default values:
//     SourceRect:    nil. When SourceRect is nil, the whole source image is used.
//     GeoM:          Identity matrix
//     ColorM:        Identity matrix (that changes no colors)
//     CompositeMode: CompositeModeSourceOver (regular alpha blending)
//
// For drawing, the pixels of the argument image at the time of this call is adopted.
// Even if the argument image is mutated after this call,
// the drawing result is never affected.
//
// When the image is disposed, DrawImage does nothing.
//
// When the given image is as same as i, DrawImage panics.
//
// DrawImage always returns nil as of 1.5.0-alpha.
func (i *Image) DrawImage(img *Image, options *DrawImageOptions) error {
	if i == img {
		panic("ebiten: Image.DrawImage: img must be different from the receiver")
	}
	if i.restorable == nil {
		return nil
	}
	// Calculate vertices before locking because the user can do anything in
	// options.ImageParts interface without deadlock (e.g. Call Image functions).
	if options == nil {
		options = &DrawImageOptions{}
	}
	parts := options.ImageParts
	// Parts is deprecated. This implementations is for backward compatibility.
	if parts == nil && options.Parts != nil {
		parts = imageParts(options.Parts)
	}
	// ImageParts is deprecated. This implementations is for backward compatibility.
	if parts != nil {
		l := parts.Len()
		for idx := 0; idx < l; idx++ {
			sx0, sy0, sx1, sy1 := parts.Src(idx)
			dx0, dy0, dx1, dy1 := parts.Dst(idx)
			op := &DrawImageOptions{
				ColorM:        options.ColorM,
				CompositeMode: options.CompositeMode,
			}
			r := image.Rect(sx0, sy0, sx1, sy1)
			op.SourceRect = &r
			op.GeoM.Scale(
				float64(dx1-dx0)/float64(sx1-sx0),
				float64(dy1-dy0)/float64(sy1-sy0))
			op.GeoM.Translate(float64(dx0), float64(dy0))
			op.GeoM.Concat(options.GeoM)
			i.DrawImage(img, op)
		}
		return nil
	}
	w, h := img.restorable.Size()
	sx0, sy0, sx1, sy1 := 0, 0, w, h
	if r := options.SourceRect; r != nil {
		sx0 = r.Min.X
		sy0 = r.Min.Y
		sx1 = r.Max.X
		sy1 = r.Max.Y
	}
	vs := vertices(sx0, sy0, sx1, sy1, w, h, &options.GeoM.impl)
	mode := opengl.CompositeMode(options.CompositeMode)
	i.restorable.DrawImage(img.restorable, vs, &options.ColorM.impl, mode)
	return nil
}

// Bounds returns the bounds of the image.
func (i *Image) Bounds() image.Rectangle {
	w, h := i.restorable.Size()
	return image.Rect(0, 0, w, h)
}

// ColorModel returns the color model of the image.
func (i *Image) ColorModel() color.Model {
	return color.RGBAModel
}

// At returns the color of the image at (x, y).
//
// This method loads pixels from GPU to system memory if necessary.
//
// This method can't be called before the main loop (ebiten.Run) starts (as of version 1.4.0-alpha).
func (i *Image) At(x, y int) color.Color {
	if i.restorable == nil {
		return color.Transparent
	}
	// TODO: Error should be delayed until flushing. Do not panic here.
	clr, err := i.restorable.At(x, y)
	if err != nil {
		panic(err)
	}
	return clr
}

// Dispose disposes the image data. After disposing, the image becomes invalid.
// This is useful to save memory.
//
// The behavior of any functions for a disposed image is undefined.
//
// When the image is disposed, Dipose does nothing.
//
// Dipose always return nil as of 1.5.0-alpha.
func (i *Image) Dispose() error {
	if i.restorable == nil {
		return nil
	}
	i.restorable.Dispose()
	i.restorable = nil
	runtime.SetFinalizer(i, nil)
	return nil
}

// ReplacePixels replaces the pixels of the image with p.
//
// The given p must represent RGBA pre-multiplied alpha values. len(p) must equal to 4 * (image width) * (image height).
//
// ReplacePixels may be slow (as for implementation, this calls glTexSubImage2D).
//
// When len(p) is not 4 * (width) * (height), ReplacePixels panics.
//
// When the image is disposed, ReplacePixels does nothing.
//
// ReplacePixels always returns nil as of 1.5.0-alpha.
func (i *Image) ReplacePixels(p []uint8) error {
	if i.restorable == nil {
		return nil
	}
	w, h := i.restorable.Size()
	if l := 4 * w * h; len(p) != l {
		panic(fmt.Sprintf("ebiten: len(p) was %d but must be %d", len(p), l))
	}
	w2, h2 := graphics.NextPowerOf2Int(w), graphics.NextPowerOf2Int(h)
	pix := make([]uint8, 4*w2*h2)
	for j := 0; j < h; j++ {
		copy(pix[j*w2*4:], p[j*w*4:(j+1)*w*4])
	}
	i.restorable.ReplacePixels(pix)
	return nil
}

// A DrawImageOptions represents options to render an image on an image.
type DrawImageOptions struct {
	SourceRect    *image.Rectangle
	GeoM          GeoM
	ColorM        ColorM
	CompositeMode CompositeMode

	// Deprecated (as of 1.5.0-alpha): Use Part instead.
	ImageParts ImageParts

	// Deprecated (as of 1.1.0-alpha): Use Part instead.
	Parts []ImagePart
}

// NewImage returns an empty image.
//
// If width or height is less than 1 or more than MaxImageSize, NewImage panics.
//
// Error returned by NewImage is always nil as of 1.5.0-alpha.
func NewImage(width, height int, filter Filter) (*Image, error) {
	checkSize(width, height)
	r := restorable.NewImage(width, height, glFilter(filter), false)
	r.Fill(color.RGBA{})
	i := &Image{r}
	runtime.SetFinalizer(i, (*Image).Dispose)
	return i, nil
}

// newVolatileImage returns an empty 'volatile' image.
// A volatile image is always cleared at the start of a frame.
//
// This is suitable for offscreen images that pixels are changed often.
//
// Pixels in regular non-volatile images are saved at each end of a frame if the image
// is changed, and restored automatically from the saved pixels on GL context lost.
// On the other hand, pixels in volatile images are not saved.
// Saving pixels is an expensive operation, and it is desirable to avoid it if possible.
//
// Note that volatile images are internal only and will never be source of drawing.
//
// If width or height is less than 1 or more than MaxImageSize, newVolatileImage panics.
//
// Error returned by newVolatileImage is always nil as of 1.5.0-alpha.
func newVolatileImage(width, height int, filter Filter) *Image {
	checkSize(width, height)
	r := restorable.NewImage(width, height, glFilter(filter), true)
	r.Fill(color.RGBA{})
	i := &Image{r}
	runtime.SetFinalizer(i, (*Image).Dispose)
	return i
}

// NewImageFromImage creates a new image with the given image (source).
//
// If source's width or height is less than 1 or more than MaxImageSize, NewImageFromImage panics.
//
// Error returned by NewImageFromImage is always nil as of 1.5.0-alpha.
func NewImageFromImage(source image.Image, filter Filter) (*Image, error) {
	size := source.Bounds().Size()
	w, h := size.X, size.Y
	checkSize(w, h)
	rgbaImg := graphics.CopyImage(source)
	r := restorable.NewImageFromImage(rgbaImg, w, h, glFilter(filter))
	i := &Image{r}
	runtime.SetFinalizer(i, (*Image).Dispose)
	return i, nil
}

func newImageWithScreenFramebuffer(width, height int) *Image {
	checkSize(width, height)
	r := restorable.NewScreenFramebufferImage(width, height)
	i := &Image{r}
	runtime.SetFinalizer(i, (*Image).Dispose)
	return i
}

const MaxImageSize = graphics.MaxImageSize

func checkSize(width, height int) {
	if width <= 0 {
		panic("ebiten: width must be more than 0")
	}
	if height <= 0 {
		panic("ebiten: height must be more than 0")
	}
	if width > MaxImageSize {
		panic(fmt.Sprintf("ebiten: width must be less than or equal to %d", MaxImageSize))
	}
	if height > MaxImageSize {
		panic(fmt.Sprintf("ebiten: height must be less than or equal to %d", MaxImageSize))
	}
}
