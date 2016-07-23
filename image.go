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
	"sync"

	"github.com/hajimehoshi/ebiten/internal/graphics/opengl"
	"github.com/hajimehoshi/ebiten/internal/ui"
)

type images struct {
	images map[*imageImpl]struct{}
	m      sync.Mutex
}

var theImagesForRestoring = images{
	images: map[*imageImpl]struct{}{},
}

func (i *images) add(img *imageImpl) (*Image, error) {
	i.m.Lock()
	defer i.m.Unlock()
	i.images[img] = struct{}{}
	eimg := &Image{img}
	runtime.SetFinalizer(eimg, theImagesForRestoring.remove)
	return eimg, nil
}

func (i *images) remove(img *Image) {
	if err := img.Dispose(); err != nil {
		panic(err)
	}
	i.m.Lock()
	defer i.m.Unlock()
	delete(i.images, img.impl)
	runtime.SetFinalizer(img, nil)
}

func (i *images) flushPixelsIfNeeded(target *Image, context *opengl.Context) error {
	i.m.Lock()
	defer i.m.Unlock()
	for img := range i.images {
		if err := img.flushPixelsIfNeeded(target, context); err != nil {
			return err
		}
	}
	return nil
}

func (i *images) restore(context *opengl.Context) error {
	i.m.Lock()
	defer i.m.Unlock()
	// Dispose all images first because framebuffer/texture numbers can be reused.
	// If framebuffers/textures are not disposed here, a newly created framebuffer/texture
	// number can be a same number as existing one.
	for img := range i.images {
		if img.isDisposed() {
			continue
		}
		if err := img.image.Dispose(); err != nil {
			return err
		}
	}
	imagesWithoutHistory := []*imageImpl{}
	imagesWithHistory := []*imageImpl{}
	for img := range i.images {
		if img.hasHistory() {
			imagesWithHistory = append(imagesWithHistory, img)
		} else {
			imagesWithoutHistory = append(imagesWithoutHistory, img)
		}
	}
	// Images with history can depend on other images. Let's process images without history
	// first.
	for _, img := range imagesWithoutHistory {
		if err := img.restore(context); err != nil {
			return err
		}
	}
	for _, img := range imagesWithHistory {
		if err := img.restore(context); err != nil {
			return err
		}
	}
	return nil
}

func (i *images) clearVolatileImages() error {
	i.m.Lock()
	defer i.m.Unlock()
	for img := range i.images {
		if err := img.clearIfVolatile(); err != nil {
			return err
		}
	}
	return nil
}

// Image represents an image.
// The pixel format is alpha-premultiplied.
// Image implements image.Image.
type Image struct {
	impl *imageImpl
}

// Size returns the size of the image.
//
// This function is concurrent-safe.
func (i *Image) Size() (width, height int) {
	return i.impl.width, i.impl.height
}

// Clear resets the pixels of the image into 0.
//
// This function is concurrent-safe.
func (i *Image) Clear() error {
	if err := theImagesForRestoring.flushPixelsIfNeeded(i, ui.GLContext()); err != nil {
		return err
	}
	return i.impl.Fill(color.Transparent)
}

// Fill fills the image with a solid color.
//
// This function is concurrent-safe.
func (i *Image) Fill(clr color.Color) error {
	if err := theImagesForRestoring.flushPixelsIfNeeded(i, ui.GLContext()); err != nil {
		return err
	}
	return i.impl.Fill(clr)
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
// Note that this function returns immediately and actual drawing is done lazily.
//
// This function is concurrent-safe.
func (i *Image) DrawImage(image *Image, options *DrawImageOptions) error {
	if err := theImagesForRestoring.flushPixelsIfNeeded(i, ui.GLContext()); err != nil {
		return err
	}
	return i.impl.DrawImage(image, options)
}

// Bounds returns the bounds of the image.
//
// This function is concurrent-safe.
func (i *Image) Bounds() image.Rectangle {
	return image.Rect(0, 0, i.impl.width, i.impl.height)
}

// ColorModel returns the color model of the image.
//
// This function is concurrent-safe.
func (i *Image) ColorModel() color.Model {
	return color.RGBAModel
}

// At returns the color of the image at (x, y).
//
// This method loads pixels from VRAM to system memory if necessary.
//
// This method can't be called before the main loop (ebiten.Run) starts (as of version 1.4.0-alpha).
//
// This function is concurrent-safe.
func (i *Image) At(x, y int) color.Color {
	return i.impl.At(x, y, ui.GLContext())
}

// Dispose disposes the image data. After disposing, the image becomes invalid.
// This is useful to save memory.
//
// The behavior of any functions for a disposed image is undefined.
//
// This function is concurrent-safe.
func (i *Image) Dispose() error {
	if err := theImagesForRestoring.flushPixelsIfNeeded(i, ui.GLContext()); err != nil {
		return err
	}
	if i.impl.isDisposed() {
		return nil
	}
	return i.impl.Dispose()
}

// ReplacePixels replaces the pixels of the image with p.
//
// The given p must represent RGBA pre-multiplied alpha values. len(p) must equal to 4 * (image width) * (image height).
//
// This function may be slow (as for implementation, this calls glTexSubImage2D).
//
// This function is concurrent-safe.
func (i *Image) ReplacePixels(p []uint8) error {
	if err := theImagesForRestoring.flushPixelsIfNeeded(i, ui.GLContext()); err != nil {
		return err
	}
	return i.impl.ReplacePixels(p)
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
//
// This function is concurrent-safe.
func NewImage(width, height int, filter Filter) (*Image, error) {
	img, err := newImageImpl(width, height, filter, false)
	if err != nil {
		return nil, err
	}
	if err := img.Fill(color.Transparent); err != nil {
		return nil, err
	}
	eimg, err := theImagesForRestoring.add(img)
	if err != nil {
		return nil, err
	}
	return eimg, nil
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
// This function is concurrent-safe.
func newVolatileImage(width, height int, filter Filter) (*Image, error) {
	img, err := newImageImpl(width, height, filter, true)
	if err != nil {
		return nil, err
	}
	if err := img.Fill(color.Transparent); err != nil {
		return nil, err
	}
	eimg, err := theImagesForRestoring.add(img)
	if err != nil {
		return nil, err
	}
	return eimg, nil
}

// NewImageFromImage creates a new image with the given image (source).
//
// NewImageFromImage generates a new texture and a new framebuffer.
//
// This function is concurrent-safe.
func NewImageFromImage(source image.Image, filter Filter) (*Image, error) {
	img, err := newImageImplFromImage(source, filter)
	if err != nil {
		return nil, err
	}
	eimg, err := theImagesForRestoring.add(img)
	if err != nil {
		return nil, err
	}
	return eimg, nil
}

func newImageWithScreenFramebuffer(width, height int) (*Image, error) {
	img, err := newScreenImageImpl(width, height)
	if err != nil {
		return nil, err
	}
	eimg, err := theImagesForRestoring.add(img)
	if err != nil {
		return nil, err
	}
	return eimg, nil
}
