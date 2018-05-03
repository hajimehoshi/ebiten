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
	"image"
	"image/color"
	"runtime"

	"github.com/hajimehoshi/ebiten/internal/affine"
	"github.com/hajimehoshi/ebiten/internal/graphics"
	"github.com/hajimehoshi/ebiten/internal/graphicsutil"
	"github.com/hajimehoshi/ebiten/internal/opengl"
	"github.com/hajimehoshi/ebiten/internal/shareable"
)

// emptyImage is an empty image used for filling other images with a uniform color.
//
// Do not call Fill or Clear on emptyImage or the program causes infinite recursion.
var emptyImage *Image

func init() {
	emptyImage, _ = NewImage(16, 16, FilterDefault)
}

// Image represents a rectangle set of pixels.
// The pixel format is alpha-premultiplied RGBA.
// Image implements image.Image.
//
// Functions of Image never returns error as of 1.5.0-alpha, and error values are always nil.
type Image struct {
	// addr holds self to check copying.
	// See strings.Builder for similar examples.
	addr *Image

	shareableImage *shareable.Image

	filter Filter
}

func (i *Image) copyCheck() {
	if i.addr != i {
		panic("ebiten: illegal use of non-zero Image copied by value")
	}
}

// Size returns the size of the image.
func (i *Image) Size() (width, height int) {
	return i.shareableImage.Size()
}

func (i *Image) isDisposed() bool {
	return i.shareableImage == nil
}

// Clear resets the pixels of the image into 0.
//
// When the image is disposed, Clear does nothing.
//
// Clear always returns nil as of 1.5.0-alpha.
func (i *Image) Clear() error {
	i.copyCheck()
	if i.isDisposed() {
		return nil
	}
	i.fill(0, 0, 0, 0)
	return nil
}

// Fill fills the image with a solid color.
//
// When the image is disposed, Fill does nothing.
//
// Fill always returns nil as of 1.5.0-alpha.
func (i *Image) Fill(clr color.Color) error {
	i.copyCheck()
	if i.isDisposed() {
		return nil
	}
	r, g, b, a := clr.RGBA()
	i.fill(uint8(r>>8), uint8(g>>8), uint8(b>>8), uint8(a>>8))
	return nil
}

func (i *Image) fill(r, g, b, a uint8) {
	wd, hd := i.Size()

	if wd*hd <= 256 {
		// Prefer ReplacePixels since ReplacePixels can keep the images shared.
		pix := make([]uint8, 4*wd*hd)
		for i := 0; i < wd*hd; i++ {
			pix[4*i] = r
			pix[4*i+1] = g
			pix[4*i+2] = b
			pix[4*i+3] = a
		}
		i.ReplacePixels(pix)
		return
	}

	ws, hs := emptyImage.Size()
	sw := float64(wd) / float64(ws)
	sh := float64(hd) / float64(hs)
	op := &DrawImageOptions{}
	op.GeoM.Scale(sw, sh)
	if a > 0 {
		rf := float64(r) / float64(a)
		gf := float64(g) / float64(a)
		bf := float64(b) / float64(a)
		af := float64(a) / 0xff
		op.ColorM.Translate(rf, gf, bf, af)
	}
	op.CompositeMode = CompositeModeCopy
	op.Filter = FilterNearest
	_ = i.DrawImage(emptyImage, op)
}

// DrawImage draws the given image on the image i.
//
// DrawImage accepts the options. For details, see the document of DrawImageOptions.
//
// DrawImage determinines the part to draw, then DrawImage applies the geometry matrix and the color matrix.
//
// For drawing, the pixels of the argument image at the time of this call is adopted.
// Even if the argument image is mutated after this call,
// the drawing result is never affected.
//
// When the image i is disposed, DrawImage does nothing.
// When the given image img is disposed, DrawImage panics.
//
// When the given image is as same as i, DrawImage panics.
//
// DrawImage works more efficiently as batches
// when the successive calls of DrawImages satisfies the below conditions:
//
//   * All render targets are same (A in A.DrawImage(B, op))
//   * All render sources are same (B in A.DrawImage(B, op))
//     * This is not a strong request since different images might share a same inner
//       OpenGL texture in high possibility. This is not 100%, so using the same render
//       source is safer.
//   * All ColorM values are same
//   * All CompositeMode values are same
//   * All Filter values are same
//
// For more performance tips, see https://github.com/hajimehoshi/ebiten/wiki/Performance-Tips.
//
// DrawImage always returns nil as of 1.5.0-alpha.
func (i *Image) DrawImage(img *Image, options *DrawImageOptions) error {
	i.copyCheck()
	if img.isDisposed() {
		panic("ebiten: the given image to DrawImage must not be disposed")
	}
	if i.isDisposed() {
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

	w, h := img.Size()
	sx0, sy0, sx1, sy1 := 0, 0, w, h
	if r := options.SourceRect; r != nil {
		sx0 = r.Min.X
		sy0 = r.Min.Y
		if sx1 > r.Max.X {
			sx1 = r.Max.X
		}
		if sy1 > r.Max.Y {
			sy1 = r.Max.Y
		}
	}
	geom := options.GeoM.impl
	if sx0 < 0 || sy0 < 0 {
		dx := 0.0
		dy := 0.0
		if sx0 < 0 {
			dx = -float64(sx0)
			sx0 = 0
		}
		if sy0 < 0 {
			dy = -float64(sy0)
			sy0 = 0
		}
		var g *affine.GeoM
		g = g.Translate(dx, dy)
		g = g.Concat(geom)
		geom = g
	}

	mode := opengl.CompositeMode(options.CompositeMode)

	filter := graphics.FilterNearest
	if options.Filter != FilterDefault {
		filter = graphics.Filter(options.Filter)
	} else if img.filter != FilterDefault {
		filter = graphics.Filter(img.filter)
	}

	i.shareableImage.DrawImage(img.shareableImage, sx0, sy0, sx1, sy1, geom, options.ColorM.impl, mode, filter)
	return nil
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
// At loads pixels from GPU to system memory if necessary, which means that At can be slow.
//
// At always returns a transparent color if the image is disposed.
//
// Note that important logic should not rely on At result since
// At might include a very slight error on some machines.
//
// At can't be called before the main loop (ebiten.Run) starts (as of version 1.4.0-alpha).
func (i *Image) At(x, y int) color.Color {
	if i.isDisposed() {
		return color.RGBA{}
	}
	clr, err := i.shareableImage.At(x, y)
	if err != nil {
		panic(err)
	}
	return clr
}

// Dispose disposes the image data. After disposing, most of image functions do nothing and returns meaningless values.
//
// Dispose is useful to save memory.
//
// When the image is disposed, Dipose does nothing.
//
// Dipose always return nil as of 1.5.0-alpha.
func (i *Image) Dispose() error {
	i.copyCheck()
	if i.isDisposed() {
		return nil
	}
	i.shareableImage.Dispose()
	i.shareableImage = nil
	runtime.SetFinalizer(i, nil)
	return nil
}

// ReplacePixels replaces the pixels of the image with p.
//
// The given p must represent RGBA pre-multiplied alpha values. len(p) must equal to 4 * (image width) * (image height).
//
// ReplacePixels may be slow (as for implementation, this calls glTexSubImage2D).
//
// When len(p) is not appropriate, ReplacePixels panics.
//
// When the image is disposed, ReplacePixels does nothing.
//
// ReplacePixels always returns nil as of 1.5.0-alpha.
func (i *Image) ReplacePixels(p []byte) error {
	i.copyCheck()
	if i.isDisposed() {
		return nil
	}
	i.shareableImage.ReplacePixels(p)
	return nil
}

// A DrawImageOptions represents options to render an image on an image.
type DrawImageOptions struct {
	// SourceRect is the region of the source image to draw.
	// If SourceRect is nil, whole image is used.
	//
	// It is assured that texels out of the SourceRect are never used.
	//
	// Calling DrawImage copies the content of SourceRect pointer. This means that
	// even if the SourceRect value is modified after passed to DrawImage,
	// the result of DrawImage doen't change.
	//
	//     op := &ebiten.DrawImageOptions{}
	//     r := image.Rect(0, 0, 100, 100)
	//     op.SourceRect = &r
	//     dst.DrawImage(src, op)
	//     r.Min.X = 10 // This doesn't affect the previous DrawImage.
	SourceRect *image.Rectangle

	// GeoM is a geometry matrix to draw.
	// The default (zero) value is identify, which draws the image at (0, 0).
	GeoM GeoM

	// ColorM is a color matrix to draw.
	// The default (zero) value is identity, which doesn't change any color.
	ColorM ColorM

	// CompositeMode is a composite mode to draw.
	// The default (zero) value is regular alpha blending.
	CompositeMode CompositeMode

	// Filter is a type of texture filter.
	// The default (zero) value is FilterDefault.
	//
	// Filter can also be specified at NewImage* functions, but
	// specifying filter at DrawImageOptions is recommended (as of 1.7.0-alpha).
	//
	// If both Filter specified at NewImage* and DrawImageOptions are FilterDefault,
	// FilterNearest is used.
	// If either is FilterDefault and the other is not, the latter is used.
	// Otherwise, Filter specified at DrawImageOptions is used.
	Filter Filter

	// Deprecated (as of 1.5.0-alpha): Use SourceRect instead.
	ImageParts ImageParts

	// Deprecated (as of 1.1.0-alpha): Use SourceRect instead.
	Parts []ImagePart
}

// NewImage returns an empty image.
//
// If width or height is less than 1 or more than device-dependent maximum size, NewImage panics.
//
// filter argument is just for backward compatibility.
// If you are not sure, specify FilterDefault.
//
// Error returned by NewImage is always nil as of 1.5.0-alpha.
func NewImage(width, height int, filter Filter) (*Image, error) {
	s := shareable.NewImage(width, height)
	i := &Image{
		shareableImage: s,
		filter:         filter,
	}
	i.addr = i
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
// If width or height is less than 1 or more than device-dependent maximum size, newVolatileImage panics.
func newVolatileImage(width, height int) *Image {
	i := &Image{
		shareableImage: shareable.NewVolatileImage(width, height),
	}
	i.addr = i
	runtime.SetFinalizer(i, (*Image).Dispose)
	return i
}

// NewImageFromImage creates a new image with the given image (source).
//
// If source's width or height is less than 1 or more than device-dependent maximum size, NewImageFromImage panics.
//
// filter argument is just for backward compatibility.
// If you are not sure, specify FilterDefault.
//
// Error returned by NewImageFromImage is always nil as of 1.5.0-alpha.
func NewImageFromImage(source image.Image, filter Filter) (*Image, error) {
	size := source.Bounds().Size()

	width, height := size.X, size.Y

	s := shareable.NewImage(width, height)
	i := &Image{
		shareableImage: s,
		filter:         filter,
	}
	i.addr = i
	runtime.SetFinalizer(i, (*Image).Dispose)

	_ = i.ReplacePixels(graphicsutil.CopyImage(source))
	return i, nil
}

func newImageWithScreenFramebuffer(width, height int) *Image {
	i := &Image{
		shareableImage: shareable.NewScreenFramebufferImage(width, height),
		filter:         FilterDefault,
	}
	i.addr = i
	runtime.SetFinalizer(i, (*Image).Dispose)
	return i
}

// MaxImageSize is deprecated as of 1.7.0-alpha. No replacement so far.
//
// TODO: Make this replacement (#541)
var MaxImageSize = 4096
