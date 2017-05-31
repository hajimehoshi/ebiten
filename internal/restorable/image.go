// Copyright 2016 The Ebiten Authors
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

package restorable

import (
	"errors"
	"image"
	"image/color"
	"runtime"

	"github.com/hajimehoshi/ebiten/internal/affine"
	"github.com/hajimehoshi/ebiten/internal/graphics"
	"github.com/hajimehoshi/ebiten/internal/opengl"
)

const drawImageHistoryMax = 100

type drawImageHistoryItem struct {
	image    *Image
	vertices []float32
	colorm   affine.ColorM
	mode     opengl.CompositeMode
}

// Image represents an image that can be restored when GL context is lost.
type Image struct {
	image  *graphics.Image
	filter opengl.Filter

	// baseImage and baseColor are exclusive.
	basePixels       []uint8
	baseColor        color.RGBA
	drawImageHistory []*drawImageHistoryItem

	// stale indicates whether the image needs to be synced with GPU as soon as possible.
	stale bool

	// volatile indicates whether the image is cleared whenever a frame starts.
	volatile bool

	// screen indicates whether the image is used as an actual screen.
	screen bool
}

func NewImage(width, height int, filter opengl.Filter, volatile bool) *Image {
	i := &Image{
		image:    graphics.NewImage(width, height, filter),
		filter:   filter,
		volatile: volatile,
	}
	theImages.add(i)
	runtime.SetFinalizer(i, (*Image).Dispose)
	return i
}

func NewImageFromImage(source *image.RGBA, width, height int, filter opengl.Filter) *Image {
	w2, h2 := graphics.NextPowerOf2Int(width), graphics.NextPowerOf2Int(height)
	p := make([]uint8, 4*w2*h2)
	for j := 0; j < height; j++ {
		copy(p[j*w2*4:(j+1)*w2*4], source.Pix[j*source.Stride:])
	}
	i := &Image{
		image:      graphics.NewImageFromImage(source, width, height, filter),
		basePixels: p,
		filter:     filter,
	}
	theImages.add(i)
	runtime.SetFinalizer(i, (*Image).Dispose)
	return i
}

func NewScreenFramebufferImage(width, height int) *Image {
	i := &Image{
		image:    graphics.NewScreenFramebufferImage(width, height),
		volatile: true,
		screen:   true,
	}
	theImages.add(i)
	runtime.SetFinalizer(i, (*Image).Dispose)
	return i
}

func (p *Image) BasePixelsForTesting() []uint8 {
	return p.basePixels
}

func (p *Image) Size() (int, int) {
	return p.image.Size()
}

func (p *Image) makeStale() {
	p.basePixels = nil
	p.baseColor = color.RGBA{}
	p.drawImageHistory = nil
	p.stale = true
}

func (p *Image) clearIfVolatile() {
	if !p.volatile {
		return
	}
	p.basePixels = nil
	p.baseColor = color.RGBA{}
	p.drawImageHistory = nil
	p.stale = false
	if p.image == nil {
		panic("not reached")
	}
	p.image.Fill(color.RGBA{})
}

func (p *Image) Fill(clr color.RGBA) {
	theImages.resetPixelsIfDependingOn(p)
	p.basePixels = nil
	p.baseColor = clr
	p.drawImageHistory = nil
	p.stale = false
	p.image.Fill(clr)
}

func (p *Image) ReplacePixels(pixels []uint8) {
	theImages.resetPixelsIfDependingOn(p)
	p.image.ReplacePixels(pixels)
	p.basePixels = pixels
	p.baseColor = color.RGBA{}
	p.drawImageHistory = nil
	p.stale = false
}

func (p *Image) DrawImage(img *Image, vertices []float32, colorm *affine.ColorM, mode opengl.CompositeMode) {
	theImages.resetPixelsIfDependingOn(p)
	if img.stale || img.volatile {
		// TODO: What will happen if there are images depending on p?
		p.makeStale()
	} else {
		p.appendDrawImageHistory(img, vertices, colorm, mode)
	}
	p.image.DrawImage(img.image, vertices, colorm, mode)
}

func (p *Image) appendDrawImageHistory(image *Image, vertices []float32, colorm *affine.ColorM, mode opengl.CompositeMode) {
	if p.stale {
		return
	}
	if len(p.drawImageHistory)+1 > drawImageHistoryMax {
		p.makeStale()
		return
	}
	// All images must be resolved and not stale each after frame.
	// So we don't have to care if image is stale or not here.
	item := &drawImageHistoryItem{
		image:    image,
		vertices: vertices,
		colorm:   *colorm,
		mode:     mode,
	}
	p.drawImageHistory = append(p.drawImageHistory, item)
}

// At returns a color value at (x, y).
//
// Note that this must not be called until context is available.
func (p *Image) At(x, y int) (color.RGBA, error) {
	w, h := p.image.Size()
	w2, h2 := graphics.NextPowerOf2Int(w), graphics.NextPowerOf2Int(h)
	if x < 0 || y < 0 || w2 <= x || h2 <= y {
		return color.RGBA{}, nil
	}
	if p.basePixels == nil || p.drawImageHistory != nil || p.stale {
		if err := p.readPixelsFromGPU(p.image); err != nil {
			return color.RGBA{}, err
		}
	}
	idx := 4*x + 4*y*w2
	r, g, b, a := p.basePixels[idx], p.basePixels[idx+1], p.basePixels[idx+2], p.basePixels[idx+3]
	return color.RGBA{r, g, b, a}, nil
}

func (p *Image) makeStaleIfDependingOn(target *Image) {
	if p.stale {
		return
	}
	for _, c := range p.drawImageHistory {
		if c.image == target {
			p.makeStale()
			return
		}
	}
}

func (p *Image) readPixelsFromGPU(image *graphics.Image) error {
	var err error
	p.basePixels, err = image.Pixels()
	if err != nil {
		return err
	}
	p.baseColor = color.RGBA{}
	p.drawImageHistory = nil
	p.stale = false
	return nil
}

func (p *Image) resolveStalePixels() error {
	if p.volatile {
		return nil
	}
	if !p.stale {
		return nil
	}
	return p.readPixelsFromGPU(p.image)
}

func (p *Image) hasDependency() bool {
	if p.stale {
		return false
	}
	return len(p.drawImageHistory) > 0
}

// Restore restores *graphics.Image from the pixels using its state.
func (p *Image) restore() error {
	w, h := p.image.Size()
	if p.screen {
		// The screen image should also be recreated because framebuffer might
		// be changed.
		p.image = graphics.NewScreenFramebufferImage(w, h)
		p.basePixels = nil
		p.baseColor = color.RGBA{}
		p.drawImageHistory = nil
		p.stale = false
		return nil
	}
	if p.volatile {
		p.image = graphics.NewImage(w, h, p.filter)
		p.basePixels = nil
		p.baseColor = color.RGBA{}
		p.drawImageHistory = nil
		p.stale = false
		return nil
	}
	if p.stale {
		// TODO: panic here?
		return errors.New("restorable: pixels must not be stale when restoring")
	}
	w2, h2 := graphics.NextPowerOf2Int(w), graphics.NextPowerOf2Int(h)
	img := image.NewRGBA(image.Rect(0, 0, w2, h2))
	if p.basePixels != nil {
		for j := 0; j < h; j++ {
			copy(img.Pix[j*img.Stride:], p.basePixels[j*w2*4:(j+1)*w2*4])
		}
	}
	gimg := graphics.NewImageFromImage(img, w, h, p.filter)
	if p.baseColor != (color.RGBA{}) {
		if p.basePixels != nil {
			panic("not reached")
		}
		gimg.Fill(p.baseColor)
	}
	for _, c := range p.drawImageHistory {
		// c.image.image must be already restored.
		if c.image.hasDependency() {
			panic("not reached")
		}
		gimg.DrawImage(c.image.image, c.vertices, &c.colorm, c.mode)
	}
	p.image = gimg

	var err error
	p.basePixels, err = gimg.Pixels()
	if err != nil {
		return err
	}
	p.baseColor = color.RGBA{}
	p.drawImageHistory = nil
	p.stale = false
	return nil
}

func (p *Image) Dispose() {
	theImages.resetPixelsIfDependingOn(p)
	p.image.Dispose()
	p.image = nil
	p.basePixels = nil
	p.baseColor = color.RGBA{}
	p.drawImageHistory = nil
	p.stale = false
	theImages.remove(p)
	runtime.SetFinalizer(p, nil)
}

func (p *Image) IsInvalidated() bool {
	return p.image.IsInvalidated()
}
