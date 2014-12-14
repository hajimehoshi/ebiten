/*
Copyright 2014 Hajime Hoshi

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package ebiten

import (
	"github.com/hajimehoshi/ebiten/internal/opengl"
	"github.com/hajimehoshi/ebiten/internal/opengl/internal/shader"
)

// A Rect represents a rectangle.
type Rect struct {
	X      int
	Y      int
	Width  int
	Height int
}

// A TexturePart represents a part of a texture.
type TexturePart struct {
	LocationX int
	LocationY int
	Source    Rect
}

// A Drawer is the interface that draws itself.
type Drawer interface {
	Draw(parts []TexturePart, geo GeometryMatrix, color ColorMatrix)
}

// DrawWhole draws the whole texture.
func DrawWhole(drawer Drawer, width, height int, geo GeometryMatrix, color ColorMatrix) {
	parts := []TexturePart{
		{0, 0, Rect{0, 0, width, height}},
	}
	drawer.Draw(parts, geo, color)
}

// A Context is the interface that means a context of rendering.
type GraphicsContext interface {
	Clear()
	Fill(r, g, b uint8)
	Texture(id TextureID) Drawer
	RenderTarget(id RenderTargetID) Drawer
	// TODO: ScreenRenderTarget() Drawer
	PushRenderTarget(id RenderTargetID)
	PopRenderTarget()
}

// Filter represents the type of filter to be used when a texture or a render
// target is maginified or minified.
type Filter int

const (
	FilterNearest Filter = iota
	FilterLinear
)

// TextureID represents an ID of a texture.
type TextureID int

// IsNil returns true if the texture is nil.
func (i TextureID) IsNil() bool {
	return i == 0
}

// RenderTargetID represents an ID of a render target.
// A render target is essentially same as a texture, but it is assumed that the
// all alpha of a render target is maximum.
type RenderTargetID int

// IsNil returns true if the render target is nil.
func (i RenderTargetID) IsNil() bool {
	return i == 0
}

func u(x int, width int) float32 {
	return float32(x) / float32(opengl.AdjustSizeForTexture(width))
}

func v(y int, height int) float32 {
	return float32(y) / float32(opengl.AdjustSizeForTexture(height))
}

func textureQuads(parts []TexturePart, width, height int) []shader.TextureQuad {
	quads := make([]shader.TextureQuad, 0, len(parts))
	for _, part := range parts {
		x1 := float32(part.LocationX)
		x2 := float32(part.LocationX + part.Source.Width)
		y1 := float32(part.LocationY)
		y2 := float32(part.LocationY + part.Source.Height)
		u1 := u(part.Source.X, width)
		u2 := u(part.Source.X+part.Source.Width, width)
		v1 := v(part.Source.Y, height)
		v2 := v(part.Source.Y+part.Source.Height, height)
		quad := shader.TextureQuad{x1, x2, y1, y2, u1, u2, v1, v2}
		quads = append(quads, quad)
	}
	return quads
}
