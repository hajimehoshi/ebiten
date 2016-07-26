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

package pixels

import (
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

// Pixels represents pixels of an image for restoring when GL context is lost.
type Pixels struct {
	image *graphics.Image

	// basePixels and baseColor are exclusive.
	basePixels       []uint8
	baseColor        color.Color
	drawImageHistory []*drawImageHistoryItem
}

func NewPixels(image *graphics.Image) *Pixels {
	return &Pixels{
		image: image,
	}
}

func (p *Pixels) Clear() {
	p.basePixels = nil
	p.baseColor = nil
	p.drawImageHistory = nil
}

func (p *Pixels) Fill(clr color.Color) {
	p.basePixels = nil
	p.baseColor = clr
	p.drawImageHistory = nil
}

func (p *Pixels) ReplacePixels(pixels []uint8) {
	if p.basePixels == nil {
		p.basePixels = make([]uint8, len(pixels))
	}
	copy(p.basePixels, pixels)
	p.baseColor = nil
	p.drawImageHistory = nil
}

func (p *Pixels) AppendDrawImageHistory(image *graphics.Image, vertices []int16, geom graphics.Matrix, colorm graphics.Matrix, mode opengl.CompositeMode) {
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
// This means Pixels members must match with acutal state in GPU.
func (p *Pixels) At(idx int, context *opengl.Context) (color.Color, error) {
	if p.basePixels == nil || p.drawImageHistory != nil {
		var err error
		p.basePixels, err = p.image.Pixels(context)
		if err != nil {
			return nil, err
		}
		p.baseColor = nil
		p.drawImageHistory = nil
	}
	r, g, b, a := p.basePixels[idx], p.basePixels[idx+1], p.basePixels[idx+2], p.basePixels[idx+3]
	return color.RGBA{r, g, b, a}, nil
}

func (p *Pixels) hasHistoryWith(target *graphics.Image) bool {
	for _, c := range p.drawImageHistory {
		if c.image == target {
			return true
		}
	}
	return false
}

func (p *Pixels) Reset(context *opengl.Context) error {
	var err error
	p.basePixels, err = p.image.Pixels(context)
	if err != nil {
		return err
	}
	p.baseColor = nil
	p.drawImageHistory = nil
	return nil
}

func (p *Pixels) NeedsReset(target *graphics.Image) bool {
	if p.drawImageHistory == nil {
		return false
	}
	if !p.hasHistoryWith(target) {
		return false
	}
	return true
}

func (p *Pixels) HasHistory() bool {
	return p.drawImageHistory != nil
}

// Restore restores the pixels using its history.
//
// restore is the only function that the pixel data is not present on GPU when this is called.
func (p *Pixels) Restore(context *opengl.Context, width, height int, filter opengl.Filter) (*graphics.Image, error) {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	if p.basePixels != nil {
		for j := 0; j < height; j++ {
			copy(img.Pix[j*img.Stride:], p.basePixels[j*width*4:(j+1)*width*4])
		}
	}
	gimg, err := graphics.NewImageFromImage(img, filter)
	if err != nil {
		return nil, err
	}
	if p.baseColor != nil {
		if p.basePixels != nil {
			panic("not reach")
		}
		if err := gimg.Fill(p.baseColor); err != nil {
			return nil, err
		}
	}
	for _, c := range p.drawImageHistory {
		// c.image.impl must be already restored.
		/*if c.image.impl.hasHistory() {
			panic("not reach")
		}*/
		if err := gimg.DrawImage(c.image, c.vertices, c.geom, c.colorm, c.mode); err != nil {
			return nil, err
		}
	}
	p.basePixels, err = gimg.Pixels(context)
	if err != nil {
		return nil, err
	}
	p.baseColor = nil
	p.drawImageHistory = nil
	return gimg, nil
}
