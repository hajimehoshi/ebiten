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

	"github.com/hajimehoshi/ebiten/internal/affine"
	"github.com/hajimehoshi/ebiten/internal/opengl"
)

func CopyImage(origImg image.Image) *image.RGBA {
	size := origImg.Bounds().Size()
	w, h := size.X, size.Y
	newImg := image.NewRGBA(image.Rect(0, 0, w, h))
	switch origImg := origImg.(type) {
	case *image.Paletted:
		b := origImg.Bounds()
		x0 := b.Min.X
		y0 := b.Min.Y
		x1 := b.Max.X
		y1 := b.Max.Y
		palette := make([]color.RGBA, len(origImg.Palette))
		for i, c := range origImg.Palette {
			palette[i] = color.RGBAModel.Convert(c).(color.RGBA)
		}
		index0 := y0*origImg.Stride + x0
		index1 := 0
		d0 := origImg.Stride - (x1 - x0)
		d1 := newImg.Stride - (x1-x0)*4
		for j := 0; j < y1-y0; j++ {
			for i := 0; i < x1-x0; i++ {
				p := origImg.Pix[index0]
				c := palette[p]
				newImg.Pix[index1] = c.R
				newImg.Pix[index1+1] = c.G
				newImg.Pix[index1+2] = c.B
				newImg.Pix[index1+3] = c.A
				index0++
				index1 += 4
			}
			index0 += d0
			index1 += d1
		}
	default:
		draw.Draw(newImg, newImg.Bounds(), origImg, origImg.Bounds().Min, draw.Src)
	}
	return newImg
}

type Image struct {
	texture     *texture
	framebuffer *framebuffer
	width       int
	height      int
}

const ImageMaxSize = viewportSize

func NewImage(width, height int, filter opengl.Filter) (*Image, error) {
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
	return i, nil
}

func NewImageFromImage(img *image.RGBA, filter opengl.Filter) (*Image, error) {
	s := img.Bounds().Size()
	i := &Image{
		width:  s.X,
		height: s.Y,
	}
	c := &newImageFromImageCommand{
		result: i,
		img:    img,
		filter: filter,
	}
	theCommandQueue.Enqueue(c)
	return i, nil
}

func NewScreenFramebufferImage(width, height int) (*Image, error) {
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
	return i, nil
}

func (i *Image) Dispose() error {
	c := &disposeCommand{
		target: i,
	}
	theCommandQueue.Enqueue(c)
	return nil
}

func (i *Image) Size() (int, int) {
	return i.width, i.height
}

func (i *Image) Fill(clr color.RGBA) error {
	// TODO: Need to clone clr value
	c := &fillCommand{
		dst:   i,
		color: clr,
	}
	theCommandQueue.Enqueue(c)
	return nil
}

func (i *Image) DrawImage(src *Image, vertices []float32, clr affine.ColorM, mode opengl.CompositeMode) error {
	c := &drawImageCommand{
		dst:      i,
		src:      src,
		vertices: vertices,
		color:    clr,
		mode:     mode,
	}
	theCommandQueue.Enqueue(c)
	return nil
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
	return context.FramebufferPixels(f.native, i.width, i.height)
}

func (i *Image) ReplacePixels(p []uint8) error {
	pixels := make([]uint8, len(p))
	copy(pixels, p)
	c := &replacePixelsCommand{
		dst:    i,
		pixels: pixels,
	}
	theCommandQueue.Enqueue(c)
	return nil
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
