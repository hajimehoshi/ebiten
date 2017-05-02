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
	"sync"

	"github.com/hajimehoshi/ebiten/internal/graphics"
	"github.com/hajimehoshi/ebiten/internal/opengl"
	"github.com/hajimehoshi/ebiten/internal/restorable"
)

func glContext() *opengl.Context {
	// This is called from finalizers even when the context or the program is not set.
	g, ok := theGraphicsContext.Load().(*graphicsContext)
	if !ok {
		return nil
	}
	if g == nil {
		return nil
	}
	return g.GLContext()
}

type images struct {
	images      map[*restorable.Image]struct{}
	m           sync.Mutex
	lastChecked *imageImpl
}

var theImagesForRestoring = images{
	images: map[*restorable.Image]struct{}{},
}

func (i *images) add(img *imageImpl) *Image {
	i.m.Lock()
	defer i.m.Unlock()
	i.images[img.restorable] = struct{}{}
	eimg := &Image{img}
	runtime.SetFinalizer(eimg, theImagesForRestoring.remove)
	return eimg
}

func (i *images) remove(img *Image) {
	if err := img.Dispose(); err != nil {
		panic(err)
	}
	i.m.Lock()
	defer i.m.Unlock()
	delete(i.images, img.impl.restorable)
	runtime.SetFinalizer(img, nil)
}

func (i *images) resolveStalePixels(context *opengl.Context) error {
	i.m.Lock()
	defer i.m.Unlock()
	i.lastChecked = nil
	for img := range i.images {
		if err := img.ReadPixelsFromVRAMIfStale(context); err != nil {
			return err
		}
	}
	return nil
}

func (i *images) resetPixelsIfDependingOn(target *Image) {
	i.m.Lock()
	defer i.m.Unlock()
	if i.lastChecked == target.impl {
		return
	}
	i.lastChecked = target.impl
	if target.impl.isDisposed() {
		return
	}
	for img := range i.images {
		img.MakeStaleIfDependingOn(target.impl.restorable)
	}
}

func (i *images) restore(context *opengl.Context) error {
	i.m.Lock()
	defer i.m.Unlock()
	// Framebuffers/textures cannot be disposed since framebuffers/textures that
	// don't belong to the current context.
	imagesWithoutDependency := []*restorable.Image{}
	imagesWithDependency := []*restorable.Image{}
	for img := range i.images {
		if img.HasDependency() {
			imagesWithDependency = append(imagesWithDependency, img)
		} else {
			imagesWithoutDependency = append(imagesWithoutDependency, img)
		}
	}
	// Images depending on other images should be processed first.
	for _, img := range imagesWithoutDependency {
		if err := img.Restore(context); err != nil {
			return err
		}
	}
	for _, img := range imagesWithDependency {
		if err := img.Restore(context); err != nil {
			return err
		}
	}
	return nil
}

func (i *images) clearVolatileImages() {
	i.m.Lock()
	defer i.m.Unlock()
	for img := range i.images {
		img.ClearIfVolatile()
	}
}

// Image represents an image.
// The pixel format is alpha-premultiplied.
// Image implements image.Image.
//
// Functions of Image never returns error as of 1.5.0-alpha, and error values are always nil.
type Image struct {
	impl *imageImpl
}

// Size returns the size of the image.
func (i *Image) Size() (width, height int) {
	return i.impl.restorable.Size()
}

// Clear resets the pixels of the image into 0.
//
// When the image is disposed, Clear does nothing.
//
// Clear always returns nil as of 1.5.0-alpha.
func (i *Image) Clear() error {
	theImagesForRestoring.resetPixelsIfDependingOn(i)
	i.impl.Fill(color.Transparent)
	return nil
}

// Fill fills the image with a solid color.
//
// When the image is disposed, Fill does nothing.
//
// Fill always returns nil as of 1.5.0-alpha.
func (i *Image) Fill(clr color.Color) error {
	theImagesForRestoring.resetPixelsIfDependingOn(i)
	i.impl.Fill(clr)
	return nil
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
// For drawing, the pixels of the argument image at the time of this call is adopted.
// Even if the argument image is mutated after this call,
// the drawing result is never affected.
//
// When the image is disposed, DrawImage does nothing.
//
// When image is as same as i, DrawImage panics.
//
// DrawImage always returns nil as of 1.5.0-alpha.
func (i *Image) DrawImage(image *Image, options *DrawImageOptions) error {
	theImagesForRestoring.resetPixelsIfDependingOn(i)
	i.impl.DrawImage(image, options)
	return nil
}

// Bounds returns the bounds of the image.
func (i *Image) Bounds() image.Rectangle {
	w, h := i.impl.restorable.Size()
	return image.Rect(0, 0, w, h)
}

// ColorModel returns the color model of the image.
func (i *Image) ColorModel() color.Model {
	return color.RGBAModel
}

// At returns the color of the image at (x, y).
//
// This method loads pixels from VRAM to system memory if necessary.
//
// This method can't be called before the main loop (ebiten.Run) starts (as of version 1.4.0-alpha).
func (i *Image) At(x, y int) color.Color {
	return i.impl.At(x, y, glContext())
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
	if i.impl.isDisposed() {
		return nil
	}
	theImagesForRestoring.resetPixelsIfDependingOn(i)
	i.impl.Dispose()
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
	theImagesForRestoring.resetPixelsIfDependingOn(i)
	i.impl.ReplacePixels(p)
	return nil
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
// If width or height is less than 1 or more than MaxImageSize, NewImage panics.
//
// Error returned by NewImage is always nil as of 1.5.0-alpha.
func NewImage(width, height int, filter Filter) (*Image, error) {
	checkSize(width, height)
	img := newImageImpl(width, height, filter, false)
	img.Fill(color.Transparent)
	return theImagesForRestoring.add(img), nil
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
// If width or height is less than 1 or more than MaxImageSize, newVolatileImage panics.
//
// Error returned by newVolatileImage is always nil as of 1.5.0-alpha.
func newVolatileImage(width, height int, filter Filter) (*Image, error) {
	checkSize(width, height)
	img := newImageImpl(width, height, filter, true)
	img.Fill(color.Transparent)
	return theImagesForRestoring.add(img), nil
}

// NewImageFromImage creates a new image with the given image (source).
//
// If source's width or height is less than 1 or more than MaxImageSize, NewImageFromImage panics.
//
// Error returned by NewImageFromImage is always nil as of 1.5.0-alpha.
func NewImageFromImage(source image.Image, filter Filter) (*Image, error) {
	size := source.Bounds().Size()
	checkSize(size.X, size.Y)
	img := newImageImplFromImage(source, filter)
	return theImagesForRestoring.add(img), nil
}

func newImageWithScreenFramebuffer(width, height int) (*Image, error) {
	checkSize(width, height)
	img := newScreenImageImpl(width, height)
	return theImagesForRestoring.add(img), nil
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
