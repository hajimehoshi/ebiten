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

type imageImpl struct {
	image    *graphics.Image
	screen   bool
	disposed bool
	width    int
	height   int
	filter   Filter
	pixels   []uint8
	noSave   bool
	m        sync.Mutex
}

func (i *imageImpl) Fill(clr color.Color) error {
	i.m.Lock()
	defer i.m.Unlock()
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
	i.m.Lock()
	defer i.m.Unlock()
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
	if i.noSave {
		return nil
	}
	if i.disposed {
		return nil
	}
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

func (i *imageImpl) restorePixels() error {
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
		i.image, err = graphics.NewImageFromImage(img, glFilter(i.filter))
		if err != nil {
			return err
		}
		return nil
	}
	var err error
	i.image, err = graphics.NewImage(i.width, i.height, glFilter(i.filter))
	if err != nil {
		return err
	}
	return nil
}

func (i *imageImpl) Dispose() error {
	i.m.Lock()
	defer i.m.Unlock()
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
	i.m.Lock()
	defer i.m.Unlock()
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
