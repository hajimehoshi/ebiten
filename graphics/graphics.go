// Copyright 2013 Hajime Hoshi
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package graphics

import (
	"github.com/hajimehoshi/go.ebiten/graphics/matrix"
	"image"
	"image/color"
)

type Rect struct {
	X      int
	Y      int
	Width  int
	Height int
}

type TexturePart struct {
	LocationX int
	LocationY int
	Source    Rect
}

type Context interface {
	Screen() RenderTarget
	Clear()
	Fill(clr color.Color)
	DrawTexture(textureID TextureID,
		geometryMatrix matrix.Geometry,
		colorMatrix matrix.Color)
	DrawTextureParts(textureID TextureID,
		parts []TexturePart,
		geometryMatrix matrix.Geometry,
		colorMatrix matrix.Color)
	SetOffscreen(renderTargetID RenderTargetID)
}

type TextureFactory interface {
	NewRenderTarget(width, height int) RenderTarget
	NewTextureFromImage(img image.Image) (Texture, error)
}

type Texture interface {
	ID() TextureID
	Width() int
	Height() int
}

type TextureID int

type RenderTarget interface {
	Texture() Texture
	ID() RenderTargetID
	Width() int
	Height() int
}

type RenderTargetID int

