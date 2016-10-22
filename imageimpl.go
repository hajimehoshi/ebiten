// Copyright 2016 Hajime Hoshi
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

	"github.com/hajimehoshi/ebiten/internal/graphics/opengl"
	"github.com/hajimehoshi/ebiten/internal/restorable"
)

type imageImpl struct {
	restorable *restorable.Image
	m          sync.Mutex
}

func newImageImpl(width, height int, filter Filter, volatile bool) (*imageImpl, error) {
	img, err := restorable.NewImage(width, height, glFilter(filter), volatile)
	if err != nil {
		return nil, err
	}
	i := &imageImpl{
		restorable: img,
	}
	runtime.SetFinalizer(i, (*imageImpl).Dispose)
	return i, nil
}

func newImageImplFromImage(source image.Image, filter Filter) (*imageImpl, error) {
	size := source.Bounds().Size()
	w, h := size.X, size.Y
	// TODO: Return error when the image is too big!
	// Don't lock while manipulating an image.Image interface.

	// It is necessary to copy the source image since the actual construction of
	// an image is delayed and we can't expect the source image is not modified
	// until the construction.
	origImg := source
	newImg := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(newImg, newImg.Bounds(), origImg, origImg.Bounds().Min, draw.Src)
	rgbaImg := newImg
	p := make([]uint8, 4*w*h)
	for j := 0; j < h; j++ {
		copy(p[j*w*4:(j+1)*w*4], rgbaImg.Pix[j*rgbaImg.Stride:])
	}
	img, err := restorable.NewImageFromImage(rgbaImg, glFilter(filter))
	if err != nil {
		return nil, err
	}
	i := &imageImpl{
		restorable: img,
	}
	i.restorable.ReplacePixels(p)
	runtime.SetFinalizer(i, (*imageImpl).Dispose)
	return i, nil
}

func newScreenImageImpl(width, height int) (*imageImpl, error) {
	img, err := restorable.NewScreenFramebufferImage(width, height)
	if err != nil {
		return nil, err
	}
	i := &imageImpl{
		restorable: img,
	}
	runtime.SetFinalizer(i, (*imageImpl).Dispose)
	return i, nil
}

func (i *imageImpl) Fill(clr color.Color) error {
	i.m.Lock()
	defer i.m.Unlock()
	if i.restorable == nil {
		return errors.New("ebiten: image is already disposed")
	}
	rgba := color.RGBAModel.Convert(clr).(color.RGBA)
	if err := i.restorable.Fill(rgba); err != nil {
		return err
	}
	return nil
}

func (i *imageImpl) clearIfVolatile() error {
	i.m.Lock()
	defer i.m.Unlock()
	if i.restorable == nil {
		return nil
	}
	if err := i.restorable.ClearIfVolatile(); err != nil {
		return err
	}
	return nil
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
			w, h := image.impl.restorable.Size()
			parts = &wholeImage{w, h}
		}
	}
	w, h := image.impl.restorable.Size()
	quads := &textureQuads{parts: parts, width: w, height: h}
	vertices := quads.vertices()
	if len(vertices) == 0 {
		return nil
	}
	if i == image.impl {
		return errors.New("ebiten: Image.DrawImage: image should be different from the receiver")
	}
	i.m.Lock()
	defer i.m.Unlock()
	if i.restorable == nil {
		return errors.New("ebiten: image is already disposed")
	}
	geom := options.GeoM
	colorm := options.ColorM
	mode := opengl.CompositeMode(options.CompositeMode)
	if err := i.restorable.DrawImage(image.impl.restorable, vertices, &geom, &colorm, mode); err != nil {
		return err
	}
	return nil
}

func (i *imageImpl) At(x, y int, context *opengl.Context) color.Color {
	if context == nil {
		panic("ebiten: At can't be called when the GL context is not initialized (this panic happens as of version 1.4.0-alpha)")
	}
	i.m.Lock()
	defer i.m.Unlock()
	if i.restorable == nil {
		return color.Transparent
	}
	w, _ := i.restorable.Size()
	idx := 4*x + 4*y*w
	clr, err := i.restorable.At(idx, context)
	if err != nil {
		panic(err)
	}
	return clr
}

func (i *imageImpl) resolveStalePixels(context *opengl.Context) error {
	i.m.Lock()
	defer i.m.Unlock()
	if i.restorable == nil {
		return nil
	}
	if err := i.restorable.ReadPixelsFromVRAMIfStale(context); err != nil {
		return err
	}
	return nil
}

func (i *imageImpl) resetPixelsIfDependingOn(target *imageImpl, context *opengl.Context) error {
	i.m.Lock()
	defer i.m.Unlock()
	if i == target {
		return nil
	}
	if i.restorable == nil {
		return nil
	}
	if target.isDisposed() {
		return errors.New("ebiten: target is already disposed")
	}
	// target is an image that is about to be tried mutating.
	// If pixels object is related to that image, the pixels must be reset.
	i.restorable.MakeStaleIfDependingOn(target.restorable)
	return nil
}

func (i *imageImpl) hasDependency() bool {
	i.m.Lock()
	defer i.m.Unlock()
	return i.restorable.HasDependency()
}

func (i *imageImpl) restore(context *opengl.Context) error {
	i.m.Lock()
	defer i.m.Unlock()
	if i.restorable == nil {
		return nil
	}
	if err := i.restorable.Restore(context); err != nil {
		return err
	}
	return nil
}

func (i *imageImpl) Dispose() error {
	i.m.Lock()
	defer i.m.Unlock()
	if i.restorable == nil {
		return errors.New("ebiten: image is already disposed")
	}
	if err := i.restorable.Dispose(); err != nil {
		return err
	}
	i.restorable = nil
	runtime.SetFinalizer(i, nil)
	return nil
}

func (i *imageImpl) ReplacePixels(p []uint8) error {
	w, h := i.restorable.Size()
	if l := 4 * w * h; len(p) != l {
		return fmt.Errorf("ebiten: p's length must be %d", l)
	}
	i.m.Lock()
	defer i.m.Unlock()
	if i.restorable == nil {
		return errors.New("ebiten: image is already disposed")
	}
	if err := i.restorable.ReplacePixels(p); err != nil {
		return err
	}
	return nil
}

func (i *imageImpl) isDisposed() bool {
	i.m.Lock()
	defer i.m.Unlock()
	return i.restorable == nil
}

func (i *imageImpl) isInvalidated(context *opengl.Context) bool {
	i.m.Lock()
	defer i.m.Unlock()
	return i.restorable.IsInvalidated(context)
}
