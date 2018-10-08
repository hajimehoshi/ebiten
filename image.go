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
	"math"
	"runtime"

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

	// shareableImages is a set of shareable.Image sorted by the order of mipmap level.
	// The level 0 image is a regular image and higher-level images are used for mipmap.
	shareableImages []*shareable.Image

	filter Filter
}

func (i *Image) copyCheck() {
	if i.addr != i {
		panic("ebiten: illegal use of non-zero Image copied by value")
	}
}

// Size returns the size of the image.
func (i *Image) Size() (width, height int) {
	return i.shareableImages[0].Size()
}

func (i *Image) isDisposed() bool {
	return len(i.shareableImages) == 0
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
	if r == 0 && g == 0 && b == 0 && a == 0 {
		i.shareableImages[0].ReplacePixels(nil)
		i.disposeMipmaps()
		return
	}

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
	i.drawImage(emptyImage, op)
}

func (i *Image) disposeMipmaps() {
	if i.isDisposed() {
		panic("not reached")
	}
	for idx := 1; idx < len(i.shareableImages); idx++ {
		i.shareableImages[idx].Dispose()
	}
	i.shareableImages = i.shareableImages[:1]
}

// DrawImage draws the given image on the image i.
//
// DrawImage accepts the options. For details, see the document of DrawImageOptions.
//
// DrawImage determines the part to draw, then DrawImage applies the geometry matrix and the color matrix.
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
//   * All ColorM values are same, or all the ColorM have only 'scale' operations
//   * All CompositeMode values are same
//   * All Filter values are same
//
// For more performance tips, see https://github.com/hajimehoshi/ebiten/wiki/Performance-Tips.
//
// DrawImage always returns nil as of 1.5.0-alpha.
func (i *Image) DrawImage(img *Image, options *DrawImageOptions) error {
	i.drawImage(img, options)
	return nil
}

func (i *Image) drawImage(img *Image, options *DrawImageOptions) {
	i.copyCheck()
	if img.isDisposed() {
		panic("ebiten: the given image to DrawImage must not be disposed")
	}
	if i.isDisposed() {
		return
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
		return
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
	geom := &options.GeoM
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
		geom = &GeoM{}
		geom.Translate(dx, dy)
		geom.Concat(options.GeoM)
	}

	mode := opengl.CompositeMode(options.CompositeMode)

	filter := graphics.FilterNearest
	if options.Filter != FilterDefault {
		filter = graphics.Filter(options.Filter)
	} else if img.filter != FilterDefault {
		filter = graphics.Filter(img.filter)
	}

	a, b, c, d, tx, ty := geom.elements()

	level := 0
	if filter == graphics.FilterLinear {
		det := geom.det()
		if det == 0 {
			return
		}
		if math.IsNaN(float64(det)) {
			return
		}
		level = graphicsutil.MipmapLevel(det)
		if level < 0 {
			panic("not reached")
		}
	}
	if level > 6 {
		level = 6
	}

	if level > 0 {
		s := 1 << uint(level)
		a *= float32(s)
		b *= float32(s)
		c *= float32(s)
		d *= float32(s)
		sx0 = sx0 / s
		sy0 = sy0 / s
		sx1 = sx1 / s
		sy1 = sy1 / s
	}

	w, h = img.shareableImages[len(img.shareableImages)-1].Size()
	for len(img.shareableImages) < level+1 {
		lastl := len(img.shareableImages) - 1
		src := img.shareableImages[lastl]
		w2 := w / 2
		h2 := h / 2
		if w2 == 0 || h2 == 0 {
			break
		}
		var s *shareable.Image
		if img.shareableImages[0].IsVolatile() {
			s = shareable.NewVolatileImage(w2, h2)
		} else {
			s = shareable.NewImage(w2, h2)
		}
		vs := src.QuadVertices(0, 0, w, h, 0.5, 0, 0, 0.5, 0, 0, 1, 1, 1, 1)
		is := graphicsutil.QuadIndices()
		s.DrawImage(src, vs, is, options.ColorM.impl, opengl.CompositeModeCopy, graphics.FilterLinear)
		img.shareableImages = append(img.shareableImages, s)
		w = w2
		h = h2
	}

	if level < len(img.shareableImages) {
		src := img.shareableImages[level]
		colorm := options.ColorM.impl
		cr, cg, cb, ca := float32(1), float32(1), float32(1), float32(1)
		if colorm.ScaleOnly() {
			body, _ := colorm.UnsafeElements()
			cr = body[0]
			cg = body[5]
			cb = body[10]
			ca = body[15]
		}
		vs := src.QuadVertices(sx0, sy0, sx1, sy1, a, b, c, d, tx, ty, cr, cg, cb, ca)
		is := graphicsutil.QuadIndices()

		if colorm.ScaleOnly() {
			colorm = nil
		}
		i.shareableImages[0].DrawImage(src, vs, is, colorm, mode, filter)
	}
	i.disposeMipmaps()
}

// Vertex represents a vertex passed to DrawTriangles.
//
// Note that this API is experimental.
type Vertex struct {
	// DstX and DstY represents a point on a destination image.
	DstX float32
	DstY float32

	// SrcX and SrcY represents a point on a source image.
	SrcX float32
	SrcY float32

	// ColorR/ColorG/ColorB/ColorA represents color scaling values.
	// 1 means the original source image color is used.
	// 0 means a transparent color is used.
	ColorR float32
	ColorG float32
	ColorB float32
	ColorA float32
}

// DrawTrianglesOptions represents options to render triangles on an image.
//
// Note that this API is experimental.
type DrawTrianglesOptions struct {
	// ColorM is a color matrix to draw.
	// The default (zero) value is identity, which doesn't change any color.
	// ColorM is applied before vertex color scale is applied.
	ColorM ColorM

	// CompositeMode is a composite mode to draw.
	// The default (zero) value is regular alpha blending.
	CompositeMode CompositeMode

	// Filter is a type of texture filter.
	// The default (zero) value is FilterDefault.
	Filter Filter
}

// DrawTriangles draws a triangle with the specified vertices and their indices.
//
// If len(indices) is not multiple of 3, DrawTriangles panics.
//
// The rule in which DrawTriangles works effectively is same as DrawImage's.
//
// In contrast to DrawImage, DrawTriangles doesn't care source image edges.
// This means that you might need to add 1px gap on a source region when you render an image by DrawTriangles.
// Note that Ebiten creates texture atlases internally, so you still have to care this even when
// you render a single image.
//
// Note that this API is experimental.
func (i *Image) DrawTriangles(vertices []Vertex, indices []uint16, img *Image, options *DrawTrianglesOptions) {
	if len(indices)%3 != 0 {
		panic("ebiten: len(indices) % 3 must be 0")
	}
	// TODO: Check the maximum value of indices and len(vertices)?

	if options == nil {
		options = &DrawTrianglesOptions{}
	}

	mode := opengl.CompositeMode(options.CompositeMode)

	filter := graphics.FilterNearest
	if options.Filter != FilterDefault {
		filter = graphics.Filter(options.Filter)
	} else if img.filter != FilterDefault {
		filter = graphics.Filter(img.filter)
	}

	vs := []float32{}
	src := img.shareableImages[0]
	for _, v := range vertices {
		vs = append(vs, src.Vertex(float32(v.DstX), float32(v.DstY), v.SrcX, v.SrcY, v.ColorR, v.ColorG, v.ColorB, v.ColorA)...)
	}
	i.shareableImages[0].DrawImage(img.shareableImages[0], vs, indices, options.ColorM.impl, mode, filter)
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
	return i.shareableImages[0].At(x, y)
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
	for _, img := range i.shareableImages {
		img.Dispose()
	}
	i.shareableImages = nil
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
	i.shareableImages[0].ReplacePixels(p)
	i.disposeMipmaps()
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
		shareableImages: []*shareable.Image{s},
		filter:          filter,
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
		shareableImages: []*shareable.Image{shareable.NewVolatileImage(width, height)},
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
		shareableImages: []*shareable.Image{s},
		filter:          filter,
	}
	i.addr = i
	runtime.SetFinalizer(i, (*Image).Dispose)

	_ = i.ReplacePixels(graphicsutil.CopyImage(source))
	return i, nil
}

func newImageWithScreenFramebuffer(width, height int) *Image {
	i := &Image{
		shareableImages: []*shareable.Image{shareable.NewScreenFramebufferImage(width, height)},
		filter:          FilterDefault,
	}
	i.addr = i
	runtime.SetFinalizer(i, (*Image).Dispose)
	return i
}

// MaxImageSize is deprecated as of 1.7.0-alpha. No replacement so far.
//
// TODO: Make this replacement (#541)
var MaxImageSize = 4096
