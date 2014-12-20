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
func DrawWholeTexture(r *RenderTarget, texture *Texture, geo GeometryMatrix, color ColorMatrix) error {
	w, h := texture.Size()
	parts := []TexturePart{
		{Rect{0, 0, float64(w), float64(h)}, Rect{0, 0, float64(w), float64(h)}},
	}
	return r.DrawTexture(texture, parts, geo, color)
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
