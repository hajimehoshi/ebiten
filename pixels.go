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

type pixels struct {
	imageImpl        *imageImpl
	pixels           []uint8
	baseColor        color.Color
	drawImageHistory []*drawImageHistoryItem
}

func (p *pixels) resetWithPixels(pixels []uint8) {
	if p.pixels == nil {
		p.pixels = make([]uint8, len(pixels))
	}
	copy(p.pixels, pixels)
	p.baseColor = nil
	p.drawImageHistory = nil
}

func (p *pixels) clear() {
	p.pixels = nil
	p.baseColor = nil
	p.drawImageHistory = nil
}

func (p *pixels) fill(clr color.Color) {
	p.pixels = nil
	p.baseColor = clr
	p.drawImageHistory = nil
}

func (p *pixels) appendDrawImageHistory(item *drawImageHistoryItem) {
	p.drawImageHistory = append(p.drawImageHistory, item)
}

func (p *pixels) at(x, y int, context *opengl.Context) (color.Color, error) {
	if p.pixels == nil || p.drawImageHistory != nil {
		var err error
		p.pixels, err = p.imageImpl.image.Pixels(context)
		if err != nil {
			return nil, err
		}
		p.baseColor = nil
		p.drawImageHistory = nil
	}
	idx := 4*x + 4*y*p.imageImpl.width
	r, g, b, a := p.pixels[idx], p.pixels[idx+1], p.pixels[idx+2], p.pixels[idx+3]
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

func (p *pixels) resetHistoryIfNeeded(target *Image, context *opengl.Context) error {
	if p.drawImageHistory == nil {
		return nil
	}
	if !p.hasHistoryWith(target) {
		return nil
	}
	var err error
	p.pixels, err = p.imageImpl.image.Pixels(context)
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

func (p *pixels) restore(context *opengl.Context) (*graphics.Image, error) {
	w, h := p.imageImpl.width, p.imageImpl.height
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	if p.pixels != nil {
		for j := 0; j < h; j++ {
			copy(img.Pix[j*img.Stride:], p.pixels[j*w*4:(j+1)*w*4])
		}
	} else if p.baseColor != nil {
		r32, g32, b32, a32 := p.baseColor.RGBA()
		r, g, b, a := uint8(r32), uint8(g32), uint8(b32), uint8(a32)
		for idx := 0; idx < len(img.Pix)/4; idx++ {
			img.Pix[4*idx] = r
			img.Pix[4*idx+1] = g
			img.Pix[4*idx+2] = b
			img.Pix[4*idx+3] = a
		}
	}
	gimg, err := graphics.NewImageFromImage(img, glFilter(p.imageImpl.filter))
	if err != nil {
		return nil, err
	}
	for _, c := range p.drawImageHistory {
		if c.image.impl.hasHistory() {
			panic("not reach")
		}
		if err := gimg.DrawImage(c.image.impl.image, c.vertices, &c.geom, &c.colorm, c.mode); err != nil {
			return nil, err
		}
	}
	p.pixels, err = gimg.Pixels(context)
	if err != nil {
		return nil, err
	}
	p.baseColor = nil
	p.drawImageHistory = nil
	return gimg, nil
}
