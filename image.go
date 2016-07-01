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
	"image/draw"
	"runtime"
	"sync"

	"github.com/hajimehoshi/ebiten/internal/graphics"
	"github.com/hajimehoshi/ebiten/internal/graphics/opengl"
	"github.com/hajimehoshi/ebiten/internal/loop"
	"github.com/hajimehoshi/ebiten/internal/ui"
)

var (
	imageM sync.Mutex
)

type images struct {
	images map[*imageImpl]struct{}
	m      sync.Mutex
}

var theImages = images{
	images: map[*imageImpl]struct{}{},
}

func (i *images) add(img *imageImpl) (*Image, error) {
	i.m.Lock()
	defer i.m.Unlock()
	i.images[img] = struct{}{}
	eimg := &Image{img}
	runtime.SetFinalizer(eimg, theImages.remove)
	return eimg, nil
}

func (i *images) remove(img *Image) {
	i.m.Lock()
	defer i.m.Unlock()
	delete(i.images, img.impl)
}

func (i *images) savePixels(context *opengl.Context, exceptions map[*imageImpl]struct{}) error {
	i.m.Lock()
	defer i.m.Unlock()
	for img := range i.images {
		if _, ok := exceptions[img]; ok {
			continue
		}
		if img.isDisposed() {
			continue
		}
		if err := img.savePixels(context); err != nil {
			return err
		}
	}
	return nil
}

func (i *images) restorePixels(context *opengl.Context) error {
	i.m.Lock()
	defer i.m.Unlock()
	for img := range i.images {
		if img.screen {
			continue
		}
		if img.isDisposed() {
			continue
		}
		if err := img.image.Dispose(); err != nil {
			return err
		}
	}
	for img := range i.images {
		if err := img.restorePixels(context); err != nil {
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
	return i.impl.Fill(color.Transparent)
}

// Fill fills the image with a solid color.
//
// This function is concurrent-safe.
func (i *Image) Fill(clr color.Color) error {
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
	return i.impl.At(x, y)
}

// Dispose disposes the image data. After disposing, the image becomes invalid.
// This is useful to save memory.
//
// The behavior of any functions for a disposed image is undefined.
//
// This function is concurrent-safe.
func (i *Image) Dispose() error {
	return i.impl.Dispose()
}

// ReplacePixels replaces the pixels of the image with p.
//
// The given p must represent RGBA pre-multiplied alpha values. len(p) must equal to 4 * (image width) * (image height).
//
// This function may be slow (as for implementation, this calls glTexSubImage2D).
//
// This function is concurrent-safe.
//
// BUG(hajimehoshi) This might not work on Android (#211).
func (i *Image) ReplacePixels(p []uint8) error {
	return i.impl.ReplacePixels(p)
}

type imageImpl struct {
	image    *graphics.Image
	screen   bool
	disposed bool
	width    int
	height   int
	filter   Filter
	pixels   []uint8
}

func (i *imageImpl) Fill(clr color.Color) error {
	imageM.Lock()
	defer imageM.Unlock()
	if i.isDisposed() {
		return errors.New("ebiten: image is already disposed")
	}
	i.pixels = nil
	return i.image.Fill(clr)
}

func (i *imageImpl) DrawImage(image *Image, options *DrawImageOptions) error {
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
			parts = &wholeImage{image.impl.width, image.impl.height}
		}
	}
	quads := &textureQuads{parts: parts, width: image.impl.width, height: image.impl.height}
	// TODO: Reuse one vertices instead of making here, but this would need locking.
	vertices := make([]int16, parts.Len()*16)
	n := quads.vertices(vertices)
	if n == 0 {
		return nil
	}
	if i == image.impl {
		return errors.New("ebiten: Image.DrawImage: image should be different from the receiver")
	}
	imageM.Lock()
	defer imageM.Unlock()
	if i.isDisposed() {
		return errors.New("ebiten: image is already disposed")
	}
	i.pixels = nil
	geom := &options.GeoM
	colorm := &options.ColorM
	mode := opengl.CompositeMode(options.CompositeMode)
	if err := i.image.DrawImage(image.impl.image, vertices[:16*n], geom, colorm, mode); err != nil {
		return err
	}
	return nil
}

func (i *imageImpl) At(x, y int) color.Color {
	if !loop.IsRunning() {
		panic("ebiten: At can't be called when the GL context is not initialized (this panic happens as of version 1.4.0-alpha)")
	}
	imageM.Lock()
	defer imageM.Unlock()
	if i.isDisposed() {
		return color.Transparent
	}
	if i.pixels == nil {
		var err error
		i.pixels, err = i.image.Pixels(ui.GLContext())
		if err != nil {
			panic(err)
		}
	}
	idx := 4*x + 4*y*i.width
	r, g, b, a := i.pixels[idx], i.pixels[idx+1], i.pixels[idx+2], i.pixels[idx+3]
	return color.RGBA{r, g, b, a}
}

func (i *imageImpl) savePixels(context *opengl.Context) error {
	if i.pixels != nil {
		return nil
	}
	var err error
	i.pixels, err = i.image.Pixels(context)
	if err != nil {
		return err
	}
	return nil
}

func (i *imageImpl) restorePixels(context *opengl.Context) error {
	if i.screen {
		return nil
	}
	if i.disposed {
		return nil
	}
	if i.pixels != nil {
		img := image.NewRGBA(image.Rect(0, 0, i.width, i.height))
		for j := 0; j < i.height; j++ {
			copy(img.Pix[j*img.Stride:], i.pixels[j*i.width*4:(j+1)*i.width*4])
		}
		var err error
		i.image, err = graphics.NewImageFromImage(img, glFilter(context, i.filter))
		if err != nil {
			return err
		}
		return nil
	}
	var err error
	i.image, err = graphics.NewImage(i.width, i.height, glFilter(context, i.filter))
	if err != nil {
		return err
	}
	return nil
}

func (i *imageImpl) Dispose() error {
	imageM.Lock()
	defer imageM.Unlock()
	if i.isDisposed() {
		return errors.New("ebiten: image is already disposed")
	}
	if err := i.image.Dispose(); err != nil {
		return err
	}
	i.image = nil
	i.disposed = true
	i.pixels = nil
	runtime.SetFinalizer(i, nil)
	return nil
}

func (i *imageImpl) isDisposed() bool {
	return i.disposed
}

func (i *imageImpl) ReplacePixels(p []uint8) error {
	if l := 4 * i.width * i.height; len(p) != l {
		return fmt.Errorf("ebiten: p's length must be %d", l)
	}
	imageM.Lock()
	defer imageM.Unlock()
	if i.pixels == nil {
		i.pixels = make([]uint8, len(p))
	}
	copy(i.pixels, p)
	if i.isDisposed() {
		return errors.New("ebiten: image is already disposed")
	}
	return i.image.ReplacePixels(p)
}

func (i *imageImpl) isInvalidated(context *opengl.Context) bool {
	return i.image.IsInvalidated(context)
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
	image := &imageImpl{
		width:  width,
		height: height,
		filter: filter,
	}
	imageM.Lock()
	defer imageM.Unlock()
	eimg, err := theImages.add(image)
	if err != nil {
		return nil, err
	}
	image.image, err = graphics.NewImage(width, height, glFilter(ui.GLContext(), filter))
	if err != nil {
		return nil, err
	}
	runtime.SetFinalizer(image, (*imageImpl).Dispose)
	if err := image.image.Fill(color.Transparent); err != nil {
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
	size := source.Bounds().Size()
	w, h := size.X, size.Y
	// TODO: Return error when the image is too big!
	// Don't lock while manipulating an image.Image interface.
	rgbaImg, ok := source.(*image.RGBA)
	if !ok || source.Bounds().Min != image.ZP {
		origImg := source
		newImg := image.NewRGBA(origImg.Bounds())
		draw.Draw(newImg, newImg.Bounds(), origImg, origImg.Bounds().Min, draw.Src)
		rgbaImg = newImg
	}
	pixels := make([]uint8, 4*w*h)
	for j := 0; j < h; j++ {
		copy(pixels[j*w*4:(j+1)*w*4], rgbaImg.Pix[j*rgbaImg.Stride:])
	}
	imageM.Lock()
	defer imageM.Unlock()
	img := &imageImpl{
		width:  w,
		height: h,
		filter: filter,
		pixels: pixels,
	}
	var err error
	img.image, err = graphics.NewImageFromImage(rgbaImg, glFilter(ui.GLContext(), filter))
	if err != nil {
		// TODO: texture should be removed here?
		return nil, err
	}
	runtime.SetFinalizer(img, (*imageImpl).Dispose)
	eimg, err := theImages.add(img)
	if err != nil {
		return nil, err
	}
	return eimg, nil
}

func newImageWithScreenFramebuffer(width, height int) (*Image, error) {
	imageM.Lock()
	defer imageM.Unlock()
	i, err := graphics.NewScreenFramebufferImage(width, height)
	if err != nil {
		return nil, err
	}
	img := &imageImpl{
		image:  i,
		width:  width,
		height: height,
		screen: true,
	}
	eimg, err := theImages.add(img)
	if err != nil {
		return nil, err
	}
	runtime.SetFinalizer(img, (*imageImpl).Dispose)
	return eimg, nil
}
