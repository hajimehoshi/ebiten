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

package ebiten

import (
	"errors"
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/internal/graphics"
	"github.com/hajimehoshi/ebiten/internal/graphics/opengl"
)

type drawImageHistoryItem struct {
	image    *Image
	vertices []int16
	geom     GeoM
	colorm   ColorM
	mode     opengl.CompositeMode
}

// basePixels and baseColor are exclusive.

type pixels struct {
	basePixels       []uint8
	baseColor        color.Color
	drawImageHistory []*drawImageHistoryItem
}

func (p *pixels) resetWithPixels(pixels []uint8) {
	if p.basePixels == nil {
		p.basePixels = make([]uint8, len(pixels))
	}
	copy(p.basePixels, pixels)
	p.baseColor = nil
	p.drawImageHistory = nil
}

func (p *pixels) clear() {
	p.basePixels = nil
	p.baseColor = nil
	p.drawImageHistory = nil
}

func (p *pixels) isCleared() bool {
	return p.basePixels == nil && p.baseColor == nil && p.drawImageHistory == nil
}

func (p *pixels) fill(clr color.Color) {
	p.basePixels = nil
	p.baseColor = clr
	p.drawImageHistory = nil
}

func (p *pixels) appendDrawImageHistory(item *drawImageHistoryItem) {
	p.drawImageHistory = append(p.drawImageHistory, item)
}

func (p *pixels) at(image *graphics.Image, idx int, context *opengl.Context) (color.Color, error) {
	if p.basePixels == nil || p.drawImageHistory != nil {
		var err error
		p.basePixels, err = image.Pixels(context)
		if err != nil {
			return nil, err
		}
		p.baseColor = nil
		p.drawImageHistory = nil
	}
	r, g, b, a := p.basePixels[idx], p.basePixels[idx+1], p.basePixels[idx+2], p.basePixels[idx+3]
	return color.RGBA{r, g, b, a}, nil
}

func (p *pixels) hasHistoryWith(target *Image) bool {
	for _, c := range p.drawImageHistory {
		if c.image == target {
			return true
		}
	}
	return false
}

func (p *pixels) resetHistoryIfNeeded(image *graphics.Image, target *Image, context *opengl.Context) error {
	if p.drawImageHistory == nil {
		return nil
	}
	if !p.hasHistoryWith(target) {
		return nil
	}
	if context == nil {
		return errors.New("ebiten: OpenGL context is missing: before running the main loop, it is forbidden to manipulate an image that is used as a drawing source once.")
	}
	var err error
	p.basePixels, err = image.Pixels(context)
	if err != nil {
		return err
	}
	p.baseColor = nil
	p.drawImageHistory = nil
	return nil
}

func (p *pixels) hasHistory() bool {
	return p.drawImageHistory != nil
}

// restore restores the pixels using its history.
//
// restore is the only function that the pixel data is not present on GPU when this is called.
func (p *pixels) restore(context *opengl.Context, width, height int, filter Filter) (*graphics.Image, error) {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	if p.basePixels != nil {
		for j := 0; j < height; j++ {
			copy(img.Pix[j*img.Stride:], p.basePixels[j*width*4:(j+1)*width*4])
		}
	}
	gimg, err := graphics.NewImageFromImage(img, glFilter(filter))
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
		if c.image.impl.hasHistory() {
			panic("not reach")
		}
		if err := gimg.DrawImage(c.image.impl.image, c.vertices, &c.geom, &c.colorm, c.mode); err != nil {
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
