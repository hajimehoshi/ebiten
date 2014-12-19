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
	"github.com/hajimehoshi/ebiten/internal"
	"github.com/hajimehoshi/ebiten/internal/opengl"
	"github.com/hajimehoshi/ebiten/internal/opengl/internal/shader"
)

// A Rect represents a rectangle.
type Rect struct {
	X      float64
	Y      float64
	Width  float64
	Height float64
}

// A TexturePart represents a part of a texture.
type TexturePart struct {
	Dst Rect
	Src Rect
}

// DrawWholeTexture draws the whole texture.
func DrawWholeTexture(g GraphicsContext, texture *Texture, geo GeometryMatrix, color ColorMatrix) error {
	w, h := texture.Size()
	parts := []TexturePart{
		{Rect{0, 0, float64(w), float64(h)}, Rect{0, 0, float64(w), float64(h)}},
	}
	return g.DrawTexture(texture, parts, geo, color)
}

// A GraphicsContext is the interface that means a context of rendering.
type GraphicsContext interface {
	Clear() error
	Fill(r, g, b uint8) error
	DrawTexture(texture *Texture, parts []TexturePart, geo GeometryMatrix, color ColorMatrix) error
	// TODO: ScreenRenderTarget() Drawer
	PushRenderTarget(id *RenderTarget)
	PopRenderTarget()
}

// Filter represents the type of filter to be used when a texture is maginified or minified.
type Filter int

// Filters
const (
	FilterNearest Filter = iota
	FilterLinear
)

// Texture represents a texture.
type Texture struct {
	glTexture *opengl.Texture
}

// Size returns the size of the texture.
func (t *Texture) Size() (width int, height int) {
	return t.glTexture.Width(), t.glTexture.Height()
}

// RenderTarget represents a render target.
// A render target is essentially same as a texture, but it is assumed that the
// all alpha values of a render target is maximum.
type RenderTarget struct {
	glRenderTarget *opengl.RenderTarget
	texture        *Texture
}

// Texture returns the texture of the render target.
func (r *RenderTarget) Texture() *Texture {
	return r.texture
}

// Size returns the size of the render target.
func (r *RenderTarget) Size() (width int, height int) {
	return r.glRenderTarget.Width(), r.glRenderTarget.Height()
}

func u(x float64, width int) float32 {
	return float32(x) / float32(internal.AdjustSizeForTexture(width))
}

func v(y float64, height int) float32 {
	return float32(y) / float32(internal.AdjustSizeForTexture(height))
}

func textureQuads(parts []TexturePart, width, height int) []shader.TextureQuad {
	quads := make([]shader.TextureQuad, 0, len(parts))
	for _, part := range parts {
		x1 := float32(part.Dst.X)
		x2 := float32(part.Dst.X + part.Dst.Width)
		y1 := float32(part.Dst.Y)
		y2 := float32(part.Dst.Y + part.Dst.Height)
		u1 := u(part.Src.X, width)
		u2 := u(part.Src.X+part.Src.Width, width)
		v1 := v(part.Src.Y, height)
		v2 := v(part.Src.Y+part.Src.Height, height)
		quad := shader.TextureQuad{x1, x2, y1, y2, u1, u2, v1, v2}
		quads = append(quads, quad)
	}
	return quads
}
