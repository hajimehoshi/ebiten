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

	"github.com/hajimehoshi/ebiten/internal/graphics"
	"github.com/hajimehoshi/ebiten/internal/graphics/opengl"
)

type drawImageHistoryItem struct {
	image    *graphics.Image
	vertices []int16
	geom     graphics.Matrix
	colorm   graphics.Matrix
	mode     opengl.CompositeMode
}

// Image represents an image of an image for restoring when GL context is lost.
type Image struct {
	image *graphics.Image

	// baseImage and baseColor are exclusive.
	basePixels       []uint8
	baseColor        color.RGBA
	drawImageHistory []*drawImageHistoryItem
	stale            bool
}

func NewImage(width, height int, filter opengl.Filter) (*Image, error) {
	img, err := graphics.NewImage(width, height, filter)
	if err != nil {
		return nil, err
	}
	return &Image{
		image: img,
	}, nil
}

func NewImageFromImage(source *image.RGBA, filter opengl.Filter) (*Image, error) {
	img, err := graphics.NewImageFromImage(source, filter)
	if err != nil {
		// TODO: texture should be removed here?
		return nil, err
	}
	return &Image{
		image: img,
	}, nil
}

func NewScreenFramebufferImage(width, height int) (*Image, error) {
	img, err := graphics.NewScreenFramebufferImage(width, height)
	if err != nil {
		return nil, err
	}
	return &Image{
		image: img,
	}, nil
}

func (p *Image) MakeStale() {
	p.basePixels = nil
	p.baseColor = color.RGBA{}
	p.drawImageHistory = nil
	p.stale = true
}

func (p *Image) Clear() error {
	p.basePixels = nil
	p.baseColor = color.RGBA{}
	p.drawImageHistory = nil
	p.stale = false
	if p.image == nil {
		panic("not reach")
	}
	if err := p.image.Fill(color.RGBA{}); err != nil {
		return err
	}
	return nil
}

func (p *Image) Fill(clr color.RGBA) error {
	p.basePixels = nil
	p.baseColor = clr
	p.drawImageHistory = nil
	p.stale = false
	if err := p.image.Fill(clr); err != nil {
		return err
	}
	return nil
}

func (p *Image) ReplacePixels(pixels []uint8) {
	if p.basePixels == nil {
		p.basePixels = make([]uint8, len(pixels))
	}
	copy(p.basePixels, pixels)
	p.baseColor = color.RGBA{}
	p.drawImageHistory = nil
	p.stale = false
}

func (p *Image) DrawImage(img *Image, vertices []int16, geom graphics.Matrix, colorm graphics.Matrix, mode opengl.CompositeMode) error {
	if img.stale {
		p.MakeStale()
	} else {
		p.appendDrawImageHistory(img.image, vertices, geom, colorm, mode)
	}
	if err := p.image.DrawImage(img.image, vertices, geom, colorm, mode); err != nil {
		return err
	}
	return nil
}

func (p *Image) Image() *graphics.Image {
	// TODO: This function is temporary. Remove this.
	return p.image
}

func (p *Image) appendDrawImageHistory(image *graphics.Image, vertices []int16, geom graphics.Matrix, colorm graphics.Matrix, mode opengl.CompositeMode) {
	if p.stale {
		return
	}
	// All images must be resolved and not stale each after frame.
	// So we don't have to care if image is stale or not here.
	item := &drawImageHistoryItem{
		image:    image,
		vertices: vertices,
		geom:     geom,
		colorm:   colorm,
		mode:     mode,
	}
	p.drawImageHistory = append(p.drawImageHistory, item)
}

// At returns a color value at idx.
//
// Note that this must not be called until context is available.
// This means Pixels members must match with acutal state in VRAM.
func (p *Image) At(idx int, image *graphics.Image, context *opengl.Context) (color.RGBA, error) {
	if p.basePixels == nil || p.drawImageHistory != nil || p.stale {
		if err := p.readPixelsFromVRAM(image, context); err != nil {
			return color.RGBA{}, err
		}
	}
	r, g, b, a := p.basePixels[idx], p.basePixels[idx+1], p.basePixels[idx+2], p.basePixels[idx+3]
	return color.RGBA{r, g, b, a}, nil
}

func (p *Image) DependsOn(target *graphics.Image) bool {
	if p.stale {
		return false
	}
	// TODO: Performance is bad when drawImageHistory is too many.
	for _, c := range p.drawImageHistory {
		if c.image == target {
			return true
		}
	}
	return false
}

func (p *Image) readPixelsFromVRAM(image *graphics.Image, context *opengl.Context) error {
	var err error
	p.basePixels, err = image.Pixels(context)
	if err != nil {
		return err
	}
	p.baseColor = color.RGBA{}
	p.drawImageHistory = nil
	p.stale = false
	return nil
}

func (p *Image) ReadPixelsFromVRAMIfStale(image *graphics.Image, context *opengl.Context) error {
	if !p.stale {
		return nil
	}
	return p.readPixelsFromVRAM(image, context)
}

func (p *Image) HasDependency() bool {
	if p.stale {
		return false
	}
	return p.drawImageHistory != nil
}

// RestoreImage restores *graphics.Image from the pixels using its state.
func (p *Image) RestoreImage(context *opengl.Context, width, height int, filter opengl.Filter) error {
	if p.stale {
		return errors.New("restorable: pixels must not be stale when restoring")
	}
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	if p.basePixels != nil {
		for j := 0; j < height; j++ {
			copy(img.Pix[j*img.Stride:], p.basePixels[j*width*4:(j+1)*width*4])
		}
	}
	gimg, err := graphics.NewImageFromImage(img, filter)
	if err != nil {
		return err
	}
	if p.baseColor != (color.RGBA{}) {
		if p.basePixels != nil {
			panic("not reach")
		}
		if err := gimg.Fill(p.baseColor); err != nil {
			return err
		}
	}
	for _, c := range p.drawImageHistory {
		// c.image.impl must be already restored.
		/*if c.image.impl.hasHistory() {
			panic("not reach")
		}*/
		if err := gimg.DrawImage(c.image, c.vertices, c.geom, c.colorm, c.mode); err != nil {
			return err
		}
	}
	p.image = gimg

	p.basePixels, err = gimg.Pixels(context)
	if err != nil {
		return err
	}
	p.baseColor = color.RGBA{}
	p.drawImageHistory = nil
	p.stale = false
	return nil
}

func (p *Image) RestoreAsScreen(width, height int) error {
	// The screen image should also be recreated because framebuffer might
	// be changed.
	var err error
	p.image, err = graphics.NewScreenFramebufferImage(width, height)
	if err != nil {
		return err
	}
	// TODO: Reset other values?
	return nil
}

func (p *Image) Recreate(width, height int, filter opengl.Filter) error {
	var err error
	p.image, err = graphics.NewImage(width, height, filter)
	if err != nil {
		return err
	}
	// TODO: Reset other values?
	return nil
}

func (p *Image) Dispose() error {
	if err := p.image.Dispose(); err != nil {
		return err
	}
	p.image = nil
	p.basePixels = nil
	p.baseColor = color.RGBA{}
	p.drawImageHistory = nil
	p.stale = false
	return nil
}
