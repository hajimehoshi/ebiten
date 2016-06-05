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

type delayedImageTasks struct {
	tasks      []func() error
	m          sync.Mutex
	execCalled bool
}

var theDelayedImageTasks = &delayedImageTasks{
	tasks: []func() error{},
}

func (t *delayedImageTasks) add(f func() error) bool {
	t.m.Lock()
	defer t.m.Unlock()
	if t.execCalled {
		return false
	}
	t.tasks = append(t.tasks, f)
	return true
}

func (t *delayedImageTasks) exec() error {
	t.m.Lock()
	defer t.m.Unlock()
	t.execCalled = true
	for _, f := range t.tasks {
		if err := f(); err != nil {
			return err
		}
	}
	return nil
}

type images struct {
	images    map[*imageImpl]struct{}
	evacuated bool
	m         sync.Mutex
}

var theImages = images{
	images: map[*imageImpl]struct{}{},
}

func (i *images) add(img *imageImpl) (*Image, error) {
	i.m.Lock()
	defer i.m.Unlock()
	if i.evacuated {
		return nil, errors.New("ebiten: images must not be evacuated")
	}
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

func (i *images) isEvacuated() bool {
	i.m.Lock()
	defer i.m.Unlock()
	return i.evacuated
}

func (i *images) evacuatePixels() error {
	i.m.Lock()
	defer i.m.Unlock()
	if i.evacuated {
		return errors.New("ebiten: images must not be evacuated")
	}
	i.evacuated = true
	for img := range i.images {
		if err := img.evacuatePixels(); err != nil {
			return err
		}
	}
	return nil
}

func (i *images) restorePixels() error {
	i.m.Lock()
	defer i.m.Unlock()
	if !i.evacuated {
		return errors.New("ebiten: images must be evacuated")
	}
	for img := range i.images {
		if err := img.restorePixels(); err != nil {
			return err
		}
	}
	i.evacuated = false
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
// Be careful that this method is potentially slow.
// It would be better if you could call this method fewer times.
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
func (i *Image) ReplacePixels(p []uint8) error {
	return i.impl.ReplacePixels(p)
}

type imageImpl struct {
	framebuffer        *graphics.Framebuffer
	texture            *graphics.Texture
	defaultFramebuffer bool
	disposed           bool
	evacuated          bool
	pixels             []uint8
	width              int
	height             int
	filter             Filter
}

func (i *imageImpl) Fill(clr color.Color) error {
	f := func() error {
		imageM.Lock()
		defer imageM.Unlock()
		if i.isDisposed() {
			return errors.New("ebiten: image is already disposed")
		}
		i.pixels = nil
		return i.framebuffer.Fill(ui.GLContext(), clr)
	}
	if theDelayedImageTasks.add(f) {
		return nil
	}
	return f()
}

func isWholeNumber(x float64) bool {
	return x == float64(int64(x))
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
	geom := &options.GeoM
	colorm := &options.ColorM
	scaleX := geom.Element(0, 0)
	scaleY := geom.Element(1, 1)
	dx := geom.Element(0, 2)
	dy := geom.Element(1, 2)
	// If possible, avoid using a geometry matrix so that we can reduce calls of
	// glUniformMatrix4fv.
	if isWholeNumber(scaleX) && geom.Element(1, 0) == 0 &&
		geom.Element(0, 1) == 0 && isWholeNumber(scaleY) &&
		isWholeNumber(dx) && isWholeNumber(dy) {
		if scaleX != 1 || scaleY != 1 || dx != 0 || dy != 0 {
			parts = &transitionImageParts{
				parts:  parts,
				scaleX: int(scaleX),
				scaleY: int(scaleY),
				dx:     int(dx),
				dy:     int(dy),
			}
			geom = &GeoM{}
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
	f := func() error {
		imageM.Lock()
		defer imageM.Unlock()
		if i.isDisposed() {
			return errors.New("ebiten: image is already disposed")
		}
		i.pixels = nil
		mode := opengl.CompositeMode(options.CompositeMode)
		if err := i.framebuffer.DrawTexture(ui.GLContext(), image.impl.texture, vertices[:16*n], geom, colorm, mode); err != nil {
			return err
		}
		return nil
	}
	if theDelayedImageTasks.add(f) {
		return nil
	}
	return f()
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
		i.pixels, err = i.framebuffer.Pixels(ui.GLContext())
		if err != nil {
			panic(err)
		}
	}
	idx := 4*x + 4*y*i.width
	r, g, b, a := i.pixels[idx], i.pixels[idx+1], i.pixels[idx+2], i.pixels[idx+3]
	return color.RGBA{r, g, b, a}
}

func (i *imageImpl) evacuatePixels() error {
	imageM.Lock()
	defer imageM.Unlock()
	defer func() {
		i.evacuated = true
	}()
	if i.defaultFramebuffer {
		return nil
	}
	if i.disposed {
		return nil
	}
	if i.evacuated {
		return errors.New("ebiten: image must not be evacuated")
	}
	if i.pixels == nil {
		var err error
		i.pixels, err = i.framebuffer.Pixels(ui.GLContext())
		if err != nil {
			return err
		}
	}
	if i.framebuffer != nil {
		if err := i.framebuffer.Dispose(ui.GLContext()); err != nil {
			return err
		}
		i.framebuffer = nil
	}
	if i.texture != nil {
		if err := i.texture.Dispose(ui.GLContext()); err != nil {
			return err
		}
		i.texture = nil
	}
	return nil
}

func (i *imageImpl) restorePixels() error {
	imageM.Lock()
	defer imageM.Unlock()
	defer func() {
		i.evacuated = false
	}()
	if i.defaultFramebuffer {
		return nil
	}
	if i.disposed {
		return nil
	}
	if !i.evacuated {
		return errors.New("ebiten: image must be evacuated")
	}
	if i.pixels == nil {
		return errors.New("ebiten: pixels must not be nil")
	}
	if i.texture != nil {
		return errors.New("ebiten: texture must be nil")
	}
	if i.framebuffer != nil {
		return errors.New("ebiten: framebuffer must be nil")
	}
	img := image.NewRGBA(image.Rect(0, 0, i.width, i.height))
	for j := 0; j < i.height; j++ {
		copy(img.Pix[j*img.Stride:], i.pixels[j*i.width*4:(j+1)*i.width*4])
	}
	var err error
	i.texture, err = graphics.NewTextureFromImage(ui.GLContext(), img, glFilter(ui.GLContext(), i.filter))
	if err != nil {
		return err
	}
	i.framebuffer, err = graphics.NewFramebufferFromTexture(ui.GLContext(), i.texture)
	if err != nil {
		return err
	}
	return nil
}

func (i *imageImpl) Dispose() error {
	f := func() error {
		imageM.Lock()
		defer imageM.Unlock()
		if i.isDisposed() {
			return errors.New("ebiten: image is already disposed")
		}
		if i.framebuffer != nil {
			if err := i.framebuffer.Dispose(ui.GLContext()); err != nil {
				return err
			}
			i.framebuffer = nil
		}
		if i.texture != nil {
			if err := i.texture.Dispose(ui.GLContext()); err != nil {
				return err
			}
			i.texture = nil
		}
		i.disposed = true
		i.pixels = nil
		runtime.SetFinalizer(i, nil)
		return nil
	}

	if theDelayedImageTasks.add(f) {
		return nil
	}
	return f()
}

func (i *imageImpl) isDisposed() bool {
	return i.disposed
}

func (i *imageImpl) ReplacePixels(p []uint8) error {
	if l := 4 * i.width * i.height; len(p) != l {
		return fmt.Errorf("ebiten: p's length must be %d", l)
	}
	f := func() error {
		imageM.Lock()
		defer imageM.Unlock()
		// Don't set i.pixels here because i.pixels is used not every time.
		i.pixels = nil
		if i.isDisposed() {
			return errors.New("ebiten: image is already disposed")
		}
		return i.framebuffer.ReplacePixels(ui.GLContext(), i.texture, p)
	}
	if theDelayedImageTasks.add(f) {
		return nil
	}
	return f()
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
	eimg, err := theImages.add(image)
	if err != nil {
		return nil, err
	}
	f := func() error {
		imageM.Lock()
		defer imageM.Unlock()
		texture, err := graphics.NewTexture(ui.GLContext(), width, height, glFilter(ui.GLContext(), filter))
		if err != nil {
			return err
		}
		framebuffer, err := graphics.NewFramebufferFromTexture(ui.GLContext(), texture)
		if err != nil {
			// TODO: texture should be removed here?
			return err
		}
		image.framebuffer = framebuffer
		image.texture = texture
		runtime.SetFinalizer(image, (*imageImpl).Dispose)
		if err := image.framebuffer.Fill(ui.GLContext(), color.Transparent); err != nil {
			return err
		}
		return nil
	}
	if theDelayedImageTasks.add(f) {
		return eimg, nil
	}
	if err := f(); err != nil {
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
	img := &imageImpl{
		width:  w,
		height: h,
		filter: filter,
	}
	eimg, err := theImages.add(img)
	if err != nil {
		return nil, err
	}
	f := func() error {
		// Don't lock while manipulating an image.Image interface.
		rgbaImg, ok := source.(*image.RGBA)
		if !ok {
			origImg := source
			newImg := image.NewRGBA(origImg.Bounds())
			draw.Draw(newImg, newImg.Bounds(), origImg, origImg.Bounds().Min, draw.Src)
			rgbaImg = newImg
		}
		imageM.Lock()
		defer imageM.Unlock()
		texture, err := graphics.NewTextureFromImage(ui.GLContext(), rgbaImg, glFilter(ui.GLContext(), filter))
		if err != nil {
			return err
		}
		framebuffer, err := graphics.NewFramebufferFromTexture(ui.GLContext(), texture)
		if err != nil {
			// TODO: texture should be removed here?
			return err
		}
		img.framebuffer = framebuffer
		img.texture = texture
		runtime.SetFinalizer(img, (*imageImpl).Dispose)
		return nil
	}
	if theDelayedImageTasks.add(f) {
		return eimg, nil
	}
	if err := f(); err != nil {
		return nil, err
	}
	return eimg, nil
}

func newImageWithZeroFramebuffer(width, height int) (*Image, error) {
	img, err := newImageWithZeroFramebufferImpl(width, height)
	if err != nil {
		return nil, err
	}
	return img, nil
}

func newImageWithZeroFramebufferImpl(width, height int) (*Image, error) {
	imageM.Lock()
	defer imageM.Unlock()
	f, err := graphics.NewZeroFramebuffer(ui.GLContext(), width, height)
	if err != nil {
		return nil, err
	}
	img := &imageImpl{
		framebuffer:        f,
		texture:            nil,
		width:              width,
		height:             height,
		defaultFramebuffer: true,
	}
	eimg, err := theImages.add(img)
	if err != nil {
		return nil, err
	}
	runtime.SetFinalizer(img, (*imageImpl).Dispose)
	return eimg, nil
}
