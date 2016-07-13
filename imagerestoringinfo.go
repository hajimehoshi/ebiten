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

type imageRestoringInfo struct {
	imageImpl        *imageImpl
	pixels           []uint8
	baseColor        color.Color
	drawImageHistory []*drawImageHistoryItem
}

func (i *imageRestoringInfo) resetWithPixels(pixels []uint8) {
	if i.pixels == nil {
		i.pixels = make([]uint8, len(pixels))
	}
	copy(i.pixels, pixels)
	i.baseColor = nil
	i.drawImageHistory = nil
}

func (i *imageRestoringInfo) clear() {
	i.pixels = nil
	i.baseColor = nil
	i.drawImageHistory = nil
}

func (i *imageRestoringInfo) fill(clr color.Color) {
	i.pixels = nil
	i.baseColor = clr
	i.drawImageHistory = nil
}

func (i *imageRestoringInfo) appendDrawImageHistory(item *drawImageHistoryItem) {
	i.drawImageHistory = append(i.drawImageHistory, item)
}

func (i *imageRestoringInfo) at(x, y int, context *opengl.Context) (color.Color, error) {
	if i.pixels == nil || i.drawImageHistory != nil {
		var err error
		i.pixels, err = i.imageImpl.image.Pixels(context)
		if err != nil {
			return nil, err
		}
		i.baseColor = nil
		i.drawImageHistory = nil
	}
	idx := 4*x + 4*y*i.imageImpl.width
	r, g, b, a := i.pixels[idx], i.pixels[idx+1], i.pixels[idx+2], i.pixels[idx+3]
	return color.RGBA{r, g, b, a}, nil
}

func (i *imageRestoringInfo) hasHistoryWith(target *Image) bool {
	for _, c := range i.drawImageHistory {
		if c.image == target {
			return true
		}
	}
	return false
}

func (i *imageRestoringInfo) resetHistoryIfNeeded(target *Image, context *opengl.Context) error {
	if i.drawImageHistory == nil {
		return nil
	}
	if !i.hasHistoryWith(target) {
		return nil
	}
	var err error
	i.pixels, err = i.imageImpl.image.Pixels(context)
	if err != nil {
		return err
	}
	i.baseColor = nil
	i.drawImageHistory = nil
	return nil
}

func (i *imageRestoringInfo) hasHistory() bool {
	return i.drawImageHistory != nil
}

func (i *imageRestoringInfo) restore(context *opengl.Context) (*graphics.Image, error) {
	w, h := i.imageImpl.width, i.imageImpl.height
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	if i.pixels != nil {
		for j := 0; j < h; j++ {
			copy(img.Pix[j*img.Stride:], i.pixels[j*w*4:(j+1)*w*4])
		}
	} else if i.baseColor != nil {
		r32, g32, b32, a32 := i.baseColor.RGBA()
		r, g, b, a := uint8(r32), uint8(g32), uint8(b32), uint8(a32)
		for idx := 0; idx < len(img.Pix)/4; idx++ {
			img.Pix[4*idx] = r
			img.Pix[4*idx+1] = g
			img.Pix[4*idx+2] = b
			img.Pix[4*idx+3] = a
		}
	}
	gimg, err := graphics.NewImageFromImage(img, glFilter(i.imageImpl.filter))
	if err != nil {
		return nil, err
	}
	for _, c := range i.drawImageHistory {
		if c.image.impl.hasHistory() {
			panic("not reach")
		}
		if err := gimg.DrawImage(c.image.impl.image, c.vertices, &c.geom, &c.colorm, c.mode); err != nil {
			return nil, err
		}
	}
	i.pixels, err = gimg.Pixels(context)
	if err != nil {
		return nil, err
	}
	i.baseColor = nil
	i.drawImageHistory = nil
	return gimg, nil
}
