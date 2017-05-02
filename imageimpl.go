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
	"fmt"
	"image"
	"image/color"
	"runtime"
	"sync"

	"github.com/hajimehoshi/ebiten/internal/graphics"
	"github.com/hajimehoshi/ebiten/internal/opengl"
	"github.com/hajimehoshi/ebiten/internal/restorable"
)

type imageImpl struct {
	restorable *restorable.Image
	m          sync.Mutex
}

func newImageImpl(width, height int, filter Filter, volatile bool) *imageImpl {
	i := &imageImpl{
		restorable: restorable.NewImage(width, height, glFilter(filter), volatile),
	}
	runtime.SetFinalizer(i, (*imageImpl).Dispose)
	return i
}

func newImageImplFromImage(source image.Image, filter Filter) *imageImpl {
	size := source.Bounds().Size()
	w, h := size.X, size.Y

	// Don't lock while manipulating an image.Image interface.

	// It is necessary to copy the source image since the actual construction of
	// an image is delayed and we can't expect the source image is not modified
	// until the construction.
	rgbaImg := graphics.CopyImage(source)
	i := &imageImpl{
		restorable: restorable.NewImageFromImage(rgbaImg, w, h, glFilter(filter)),
	}
	runtime.SetFinalizer(i, (*imageImpl).Dispose)
	return i
}

func newScreenImageImpl(width, height int) *imageImpl {
	i := &imageImpl{
		restorable: restorable.NewScreenFramebufferImage(width, height),
	}
	runtime.SetFinalizer(i, (*imageImpl).Dispose)
	return i
}

func (i *imageImpl) Fill(clr color.Color) {
	i.m.Lock()
	defer i.m.Unlock()
	if i.restorable == nil {
		return
	}
	rgba := color.RGBAModel.Convert(clr).(color.RGBA)
	i.restorable.Fill(rgba)
}

func (i *imageImpl) DrawImage(image *Image, options *DrawImageOptions) {
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
	vs := vertices(parts, w, h, &options.GeoM.impl)
	if len(vs) == 0 {
		return
	}
	if i == image.impl {
		panic("ebiten: Image.DrawImage: image must be different from the receiver")
	}
	i.m.Lock()
	defer i.m.Unlock()
	if i.restorable == nil {
		return
	}
	mode := opengl.CompositeMode(options.CompositeMode)
	i.restorable.DrawImage(image.impl.restorable, vs, options.ColorM.impl, mode)
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
	// TODO: Error should be delayed until flushing. Do not panic here.
	clr, err := i.restorable.At(x, y, context)
	if err != nil {
		panic(err)
	}
	return clr
}

func (i *imageImpl) Dispose() {
	i.m.Lock()
	defer i.m.Unlock()
	if i.restorable == nil {
		return
	}
	i.restorable.Dispose()
	i.restorable = nil
	runtime.SetFinalizer(i, nil)
}

func (i *imageImpl) ReplacePixels(p []uint8) {
	w, h := i.restorable.Size()
	if l := 4 * w * h; len(p) != l {
		panic(fmt.Sprintf("ebiten: len(p) was %d but must be %d", len(p), l))
	}
	i.m.Lock()
	defer i.m.Unlock()
	if i.restorable == nil {
		return
	}
	w2, h2 := graphics.NextPowerOf2Int(w), graphics.NextPowerOf2Int(h)
	pix := make([]uint8, 4*w2*h2)
	for j := 0; j < h; j++ {
		copy(pix[j*w2*4:], p[j*w*4:(j+1)*w*4])
	}
	i.restorable.ReplacePixels(pix)
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
