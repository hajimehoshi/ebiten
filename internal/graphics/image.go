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

package graphics

import (
	"image"
	"image/color"
	"image/draw"
	"runtime"

	"github.com/hajimehoshi/ebiten/internal/affine"
	"github.com/hajimehoshi/ebiten/internal/opengl"
)

func CopyImage(origImg image.Image) *image.RGBA {
	size := origImg.Bounds().Size()
	w, h := size.X, size.Y
	newImg := image.NewRGBA(image.Rect(0, 0, NextPowerOf2Int(w), NextPowerOf2Int(h)))
	switch origImg := origImg.(type) {
	case *image.Paletted:
		b := origImg.Bounds()
		x0 := b.Min.X
		y0 := b.Min.Y
		x1 := b.Max.X
		y1 := b.Max.Y
		palette := make([]uint8, len(origImg.Palette)*4)
		for i, c := range origImg.Palette {
			rgba := color.RGBAModel.Convert(c).(color.RGBA)
			palette[4*i] = rgba.R
			palette[4*i+1] = rgba.G
			palette[4*i+2] = rgba.B
			palette[4*i+3] = rgba.A
		}
		index0 := 0
		index1 := 0
		d0 := origImg.Stride - (x1 - x0)
		d1 := newImg.Stride - (x1-x0)*4
		pix0 := origImg.Pix
		pix1 := newImg.Pix
		for j := 0; j < y1-y0; j++ {
			for i := 0; i < x1-x0; i++ {
				p := int(pix0[index0])
				pix1[index1] = palette[4*p]
				pix1[index1+1] = palette[4*p+1]
				pix1[index1+2] = palette[4*p+2]
				pix1[index1+3] = palette[4*p+3]
				index0++
				index1 += 4
			}
			index0 += d0
			index1 += d1
		}
	default:
		draw.Draw(newImg, image.Rect(0, 0, w, h), origImg, origImg.Bounds().Min, draw.Src)
	}
	runtime.Gosched()
	return newImg
}

type Image struct {
	texture     *texture
	framebuffer *framebuffer
	width       int
	height      int
}

const MaxImageSize = viewportSize

func NewImage(width, height int, filter opengl.Filter) *Image {
	i := &Image{
		width:  width,
		height: height,
	}
	c := &newImageCommand{
		result: i,
		width:  width,
		height: height,
		filter: filter,
	}
	theCommandQueue.Enqueue(c)
	return i
}

func NewImageFromImage(img *image.RGBA, width, height int, filter opengl.Filter) *Image {
	i := &Image{
		width:  width,
		height: height,
	}
	c := &newImageFromImageCommand{
		result: i,
		img:    img,
		filter: filter,
	}
	theCommandQueue.Enqueue(c)
	return i
}

func NewScreenFramebufferImage(width, height int) *Image {
	i := &Image{
		width:  width,
		height: height,
	}
	c := &newScreenFramebufferImageCommand{
		result: i,
		width:  width,
		height: height,
	}
	theCommandQueue.Enqueue(c)
	return i
}

func (i *Image) Dispose() {
	c := &disposeCommand{
		target: i,
	}
	theCommandQueue.Enqueue(c)
}

func (i *Image) Size() (int, int) {
	return i.width, i.height
}

func (i *Image) Fill(clr color.RGBA) {
	// TODO: Need to clone clr value
	c := &fillCommand{
		dst:   i,
		color: clr,
	}
	theCommandQueue.Enqueue(c)
}

func (i *Image) DrawImage(src *Image, vertices []float32, clr affine.ColorM, mode opengl.CompositeMode) {
	theCommandQueue.EnqueueDrawImageCommand(i, src, vertices, clr, mode)
}

func (i *Image) Pixels(context *opengl.Context) ([]uint8, error) {
	// Flush the enqueued commands so that pixels are certainly read.
	if err := theCommandQueue.Flush(context); err != nil {
		return nil, err
	}
	f, err := i.createFramebufferIfNeeded(context)
	if err != nil {
		return nil, err
	}
	return context.FramebufferPixels(f.native, NextPowerOf2Int(i.width), NextPowerOf2Int(i.height))
}

func (i *Image) ReplacePixels(p []uint8) {
	pixels := make([]uint8, len(p))
	copy(pixels, p)
	c := &replacePixelsCommand{
		dst:    i,
		pixels: pixels,
	}
	theCommandQueue.Enqueue(c)
}

func (i *Image) IsInvalidated(context *opengl.Context) bool {
	return !context.IsTexture(i.texture.native)
}

func (i *Image) createFramebufferIfNeeded(context *opengl.Context) (*framebuffer, error) {
	if i.framebuffer != nil {
		return i.framebuffer, nil
	}
	f, err := newFramebufferFromTexture(context, i.texture)
	if err != nil {
		return nil, err
	}
	i.framebuffer = f
	return i.framebuffer, nil
}
