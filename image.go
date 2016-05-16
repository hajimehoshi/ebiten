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

// Image represents an image.
// The pixel format is alpha-premultiplied.
// Image implements image.Image.
type Image struct {
	framebuffer *graphics.Framebuffer
	texture     *graphics.Texture
	disposed    bool
	pixels      []uint8
	width       int
	height      int
}

// Size returns the size of the image.
//
// This function is concurrent-safe.
func (i *Image) Size() (width, height int) {
	return i.width, i.height
}

// Clear resets the pixels of the image into 0.
//
// This function is concurrent-safe.
func (i *Image) Clear() (err error) {
	return i.clear()
}

func (i *Image) clear() (err error) {
	return i.fill(color.Transparent)
}

// Fill fills the image with a solid color.
//
// This function is concurrent-safe.
func (i *Image) Fill(clr color.Color) (err error) {
	return i.fill(clr)
}

func (i *Image) fill(clr color.Color) (err error) {
	f := func() error {
		imageM.Lock()
		defer imageM.Unlock()
		if i.isDisposed() {
			return errors.New("ebiten: image is already disposed")
		}
		i.pixels = nil
		return i.framebuffer.Fill(glContext, clr)
	}
	if theDelayedImageTasks.add(f) {
		return nil
	}
	return f()

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

	if i == image {
		return errors.New("ebiten: Image.DrawImage: image should be different from the receiver")
	}
	f := func() error {
		imageM.Lock()
		defer imageM.Unlock()
		if i.isDisposed() {
			return errors.New("ebiten: image is already disposed")
		}
		i.pixels = nil
		m := opengl.CompositeMode(options.CompositeMode)
		return i.framebuffer.DrawTexture(glContext, image.texture, vertices[:16*n], &options.GeoM, &options.ColorM, m)
	}
	if theDelayedImageTasks.add(f) {
		return nil
	}
	return f()
}

// Bounds returns the bounds of the image.
//
// This function is concurrent-safe.
func (i *Image) Bounds() image.Rectangle {
	return image.Rect(0, 0, i.width, i.height)
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
	if !currentRunContext.isRunning() {
		panic("ebiten: At can't be called when the GL context is not initialized (this panic happens as of version 1.4.0-alpha)")
	}
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
	idx := 4*x + 4*y*i.width
	r, g, b, a := i.pixels[idx], i.pixels[idx+1], i.pixels[idx+2], i.pixels[idx+3]
	return color.RGBA{r, g, b, a}
}

// Dispose disposes the image data. After disposing, the image becomes invalid.
// This is useful to save memory.
//
// The behavior of any functions for a disposed image is undefined.
//
// This function is concurrent-safe.
func (i *Image) Dispose() error {
	f := func() error {
		imageM.Lock()
		defer imageM.Unlock()
		if i.isDisposed() {
			return errors.New("ebiten: image is already disposed")
		}
		if i.framebuffer != nil {
			if err := i.framebuffer.Dispose(glContext); err != nil {
				return err
			}
			i.framebuffer = nil
		}
		if i.texture != nil {
			if err := i.texture.Dispose(glContext); err != nil {
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

func (i *Image) isDisposed() bool {
	return i.disposed
}

// ReplacePixels replaces the pixels of the image with p.
//
// The given p must represent RGBA pre-multiplied alpha values. len(p) must equal to 4 * (image width) * (image height).
//
// This function may be slow (as for implementation, this calls glTexSubImage2D).
//
// This function is concurrent-safe.
func (i *Image) ReplacePixels(p []uint8) error {
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
		return i.framebuffer.ReplacePixels(glContext, i.texture, p)
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
	image := &Image{
		width:  width,
		height: height,
	}
	f := func() error {
		imageM.Lock()
		defer imageM.Unlock()
		texture, err := graphics.NewTexture(glContext, width, height, glFilter(glContext, filter))
		if err != nil {
			return err
		}
		framebuffer, err := graphics.NewFramebufferFromTexture(glContext, texture)
		if err != nil {
			// TODO: texture should be removed here?
			return err
		}
		image.framebuffer = framebuffer
		image.texture = texture
		runtime.SetFinalizer(image, (*Image).Dispose)
		if err := image.framebuffer.Fill(glContext, color.Transparent); err != nil {
			return err
		}
		return nil
	}
	if theDelayedImageTasks.add(f) {
		return image, nil
	}
	if err := f(); err != nil {
		return nil, err
	}
	return image, nil
}

// NewImageFromImage creates a new image with the given image (img).
//
// NewImageFromImage generates a new texture and a new framebuffer.
//
// This function is concurrent-safe.
func NewImageFromImage(img image.Image, filter Filter) (*Image, error) {
	size := img.Bounds().Size()
	w, h := size.X, size.Y
	// TODO: Return error when the image is too big!
	eimg := &Image{
		width:  w,
		height: h,
	}
	f := func() error {
		// Don't lock while manipulating an image.Image interface.
		rgbaImg, ok := img.(*image.RGBA)
		if !ok {
			origImg := img
			newImg := image.NewRGBA(origImg.Bounds())
			draw.Draw(newImg, newImg.Bounds(), origImg, origImg.Bounds().Min, draw.Src)
			rgbaImg = newImg
		}
		imageM.Lock()
		defer imageM.Unlock()
		texture, err := graphics.NewTextureFromImage(glContext, rgbaImg, glFilter(glContext, filter))
		if err != nil {
			return err
		}
		framebuffer, err := graphics.NewFramebufferFromTexture(glContext, texture)
		if err != nil {
			// TODO: texture should be removed here?
			return err
		}
		eimg.framebuffer = framebuffer
		eimg.texture = texture
		runtime.SetFinalizer(eimg, (*Image).Dispose)
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
	imageM.Lock()
	defer imageM.Unlock()
	f, err := graphics.NewZeroFramebuffer(glContext, width, height)
	if err != nil {
		return nil, err
	}
	img := &Image{
		framebuffer: f,
		texture:     nil,
		width:       width,
		height:      height,
	}
	runtime.SetFinalizer(img, (*Image).Dispose)
	return img, nil
}
