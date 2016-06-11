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

package graphics

import (
	"image"
	"image/draw"

	"github.com/hajimehoshi/ebiten/internal/graphics/opengl"
)

func adjustImageForTexture(img *image.RGBA) *image.RGBA {
	width, height := img.Bounds().Size().X, img.Bounds().Size().Y
	adjustedImageBounds := image.Rectangle{
		image.ZP,
		image.Point{
			int(NextPowerOf2Int32(int32(width))),
			int(NextPowerOf2Int32(int32(height))),
		},
	}
	if img.Bounds() == adjustedImageBounds {
		return img
	}

	adjustedImage := image.NewRGBA(adjustedImageBounds)
	dstBounds := image.Rectangle{
		image.ZP,
		img.Bounds().Size(),
	}
	draw.Draw(adjustedImage, dstBounds, img, img.Bounds().Min, draw.Src)
	return adjustedImage
}

type Texture struct {
	native opengl.Texture
	width  int
	height int
}

func NewImage(width, height int, filter opengl.Filter) (*Texture, *Framebuffer, error) {
	texture := &Texture{}
	framebuffer := &Framebuffer{}
	c := &newImageCommand{
		texture:     texture,
		framebuffer: framebuffer,
		width:       width,
		height:      height,
		filter:      filter,
	}
	theCommandQueue.Enqueue(c)
	return texture, framebuffer, nil
}

func NewImageFromImage(img *image.RGBA, filter opengl.Filter) (*Texture, *Framebuffer, error) {
	texture := &Texture{}
	framebuffer := &Framebuffer{}
	c := &newImageFromImageCommand{
		texture:     texture,
		framebuffer: framebuffer,
		img:         img,
		filter:      filter,
	}
	theCommandQueue.Enqueue(c)
	return texture, framebuffer, nil
}
