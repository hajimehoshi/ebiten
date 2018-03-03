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

	"github.com/hajimehoshi/ebiten/internal/affine"
	"github.com/hajimehoshi/ebiten/internal/graphics"
	"github.com/hajimehoshi/ebiten/internal/graphicsutil"
	"github.com/hajimehoshi/ebiten/internal/opengl"
	"github.com/hajimehoshi/ebiten/internal/restorable"
)

// emptyImage is an empty image used for filling other images with a uniform color.
//
// Do not call Fill or Clear on emptyImage or the program causes infinite recursion.
var emptyImage *Image

func init() {
	const (
		w = 16
		h = 16
	)
	emptyImage = newImageWithoutInit(w, h)
	pix := make([]uint8, w*h*4)
	_ = emptyImage.ReplacePixels(pix)
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

	// restorable and sharedImagePart are exclusive.
	restorable      *restorable.Image
	sharedImagePart *sharedImagePart

	filter Filter
}

func (i *Image) copyCheck() {
	if i.addr == nil {
		// As it is OK that an image is allocated at heap,
		// 'noespace' function like strings.noescape is not needed.
		i.addr = i
		return
	}
	if i.addr != i {
		panic("ebiten: illegal use of non-zero Image copied by value")
	}
}

// Size returns the size of the image.
func (i *Image) Size() (width, height int) {
	w := 0
	h := 0
	if i.restorable != nil {
		w, h = i.restorable.Size()
	} else if i.sharedImagePart != nil {
		_, _, w, h = i.sharedImagePart.region()
	}
	return w, h
}

// Clear resets the pixels of the image into 0.
//
// When the image is disposed, Clear does nothing.
//
// Clear always returns nil as of 1.5.0-alpha.
func (i *Image) Clear() error {
	i.copyCheck()
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
	r, g, b, a := clr.RGBA()
	i.fill(uint8(r>>8), uint8(g>>8), uint8(b>>8), uint8(a>>8))
	return nil
}

func (img *Image) ensureNotShared() {
	if img.sharedImagePart == nil {
		return
	}
	if img.restorable == nil {
		// The image is already disposed.
		return
	}

	s := img.sharedImagePart
	x, y, w, h := s.region()

	img.restorable = restorable.NewImage(w, h, false)
	img.sharedImagePart = nil
	img.restorable.DrawImage(s.image(), x, y, w, h, nil, nil, opengl.CompositeModeCopy, graphics.FilterNearest)
	s.Dispose()
}

func (i *Image) fill(r, g, b, a uint8) {
	wd, hd := i.Size()
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

func (i *Image) isDisposed() bool {
	return i.restorable == nil && i.sharedImagePart == nil
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
	i.ensureNotShared()
	// Compare i and img after ensuring i is not shared, or
	// i and img might share the same texture even though i != img.
	if i == img {
		panic("ebiten: Image.DrawImage: img must be different from the receiver")
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

	if img.sharedImagePart != nil {
		dx, dy, _, _ := img.sharedImagePart.region()
		sx0 += dx
		sy0 += dy
		sx1 += dx
		sy1 += dy
	}

	mode := opengl.CompositeMode(options.CompositeMode)

	filter := graphics.FilterNearest
	if options.Filter != FilterDefault {
		filter = graphics.Filter(options.Filter)
	} else if img.filter != FilterDefault {
		filter = graphics.Filter(img.filter)
	}

	i.restorable.DrawImage(img.restorableImage(), sx0, sy0, sx1, sy1, geom, options.ColorM.impl, mode, filter)
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
// At can't be called before the main loop (ebiten.Run) starts (as of version 1.4.0-alpha).
func (i *Image) At(x, y int) color.Color {
	if i.isDisposed() {
		return color.RGBA{}
	}
	switch {
	case i.restorable != nil:
		// TODO: Error should be delayed until flushing. Do not panic here.
		clr, err := i.restorable.At(x, y)
		if err != nil {
			panic(err)
		}
		return clr
	case i.sharedImagePart != nil:
		ox, oy, w, h := i.sharedImagePart.region()
		if x < 0 || y < 0 || x >= w || y >= h {
			return color.RGBA{}
		}
		clr, err := i.sharedImagePart.image().At(x+ox, y+oy)
		if err != nil {
			panic(err)
		}
		return clr
	default:
		panic("not reached")
	}
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
	switch {
	case i.restorable != nil:
		i.restorable.Dispose()
		i.restorable = nil
	case i.sharedImagePart != nil:
		i.sharedImagePart.Dispose()
		i.sharedImagePart = nil
	default:
		panic("not reached")
	}
	runtime.SetFinalizer(i, nil)
	return nil
}

func (i *Image) region() (x, y, width, height int) {
	if i.restorable != nil {
		w, h := i.restorable.Size()
		return 0, 0, w, h
	}
	if i.sharedImagePart != nil {
		return i.sharedImagePart.region()
	}
	panic("not reached")
}

func (i *Image) restorableImage() *restorable.Image {
	if i.restorable != nil {
		return i.restorable
	}
	if i.sharedImagePart != nil {
		return i.sharedImagePart.image()
	}
	panic("not reached")
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
	x, y, w, h := i.region()
	if l := 4 * w * h; len(p) != l {
		panic(fmt.Sprintf("ebiten: len(p) was %d but must be %d", len(p), l))
	}
	i.restorableImage().ReplacePixels(p, x, y, w, h)
	return nil
}

// A DrawImageOptions represents options to render an image on an image.
type DrawImageOptions struct {
	// SourceRect is the region of the source image to draw.
	// If SourceRect is nil, whole image is used.
	//
	// It is assured that texels out of the SourceRect are never used.
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
// If width or height is less than 1 or more than MaxImageSize, NewImage panics.
//
// filter argument is just for backward compatibility.
// If you are not sure, specify FilterDefault.
//
// Error returned by NewImage is always nil as of 1.5.0-alpha.
func NewImage(width, height int, filter Filter) (*Image, error) {
	checkSize(width, height)
	// TODO: Is it possible to use the shared texture here? (#514)
	r := restorable.NewImage(width, height, false)
	i := &Image{
		restorable: r,
		filter:     filter,
	}
	i.fill(0, 0, 0, 0)
	runtime.SetFinalizer(i, (*Image).Dispose)
	return i, nil
}

// newImageWithoutInit creates an empty image without initialization.
func newImageWithoutInit(width, height int) *Image {
	checkSize(width, height)
	// TODO: Is it possible to use the shared texture here? (#514)
	r := restorable.NewImage(width, height, false)
	i := &Image{
		restorable: r,
		filter:     FilterDefault,
	}
	runtime.SetFinalizer(i, (*Image).Dispose)
	return i
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
	r := restorable.NewImage(width, height, true)
	i := &Image{
		restorable: r,
		filter:     filter,
	}
	i.fill(0, 0, 0, 0)
	runtime.SetFinalizer(i, (*Image).Dispose)
	return i
}

// NewImageFromImage creates a new image with the given image (source).
//
// If source's width or height is less than 1 or more than MaxImageSize, NewImageFromImage panics.
//
// filter argument is just for backward compatibility.
// If you are not sure, specify FilterDefault.
//
// Error returned by NewImageFromImage is always nil as of 1.5.0-alpha.
func NewImageFromImage(source image.Image, filter Filter) (*Image, error) {
	size := source.Bounds().Size()
	checkSize(size.X, size.Y)

	width, height := size.X, size.Y

	var i *Image
	s := newSharedImagePart(width, height)
	if s == nil {
		r := restorable.NewImage(width, height, false)
		i = &Image{
			restorable: r,
			filter:     filter,
		}
	} else {
		i = &Image{
			sharedImagePart: s,
			filter:          filter,
		}
	}
	runtime.SetFinalizer(i, (*Image).Dispose)

	rgbaImg := graphicsutil.CopyImage(source)
	p := make([]byte, 4*width*height)
	for j := 0; j < height; j++ {
		copy(p[j*width*4:(j+1)*width*4], rgbaImg.Pix[j*rgbaImg.Stride:])
	}
	_ = i.ReplacePixels(p)
	return i, nil
}

func newImageWithScreenFramebuffer(width, height int) *Image {
	checkSize(width, height)
	r := restorable.NewScreenFramebufferImage(width, height)
	i := &Image{
		restorable: r,
		filter:     FilterDefault,
	}
	runtime.SetFinalizer(i, (*Image).Dispose)
	return i
}

// MaxImageSize represents the maximum width/height of an image.
const MaxImageSize = restorable.MaxImageSize

func checkSize(width, height int) {
	if width <= 0 {
		panic("ebiten: width must be more than 0")
	}
	if height <= 0 {
		panic("ebiten: height must be more than 0")
	}
	if width > MaxImageSize {
		panic(fmt.Sprintf("ebiten: width (%d) must be less than or equal to %d", width, MaxImageSize))
	}
	if height > MaxImageSize {
		panic(fmt.Sprintf("ebiten: height (%d) must be less than or equal to %d", height, MaxImageSize))
	}
}
