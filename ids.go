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
	"github.com/go-gl/gl"
	"github.com/hajimehoshi/ebiten/internal"
	"github.com/hajimehoshi/ebiten/internal/opengl"
	"github.com/hajimehoshi/ebiten/internal/opengl/internal/shader"
	"image/color"
	"math"
)

// ids manages the current render target to be used.
// TODO: Change this name. `ids` is not appropriate for now.
type ids struct {
	currentRenderTarget *RenderTarget
}

var idsInstance = &ids{}

func (i *ids) createRenderTarget(width, height int, filter int) (*RenderTarget, error) {
	glTexture, err := opengl.NewTexture(width, height, filter)
	if err != nil {
		return nil, err
	}

	// The current binded framebuffer can be changed.
	i.currentRenderTarget = nil
	glRenderTarget, err := opengl.NewRenderTargetFromTexture(glTexture)
	if err != nil {
		return nil, err
	}

	texture := &Texture{glTexture}
	// TODO: Is |texture| necessary?
	renderTarget := &RenderTarget{glRenderTarget, texture}

	return renderTarget, nil
}

func (i *ids) fillRenderTarget(renderTarget *RenderTarget, clr color.Color) error {
	if err := i.setViewportIfNeeded(renderTarget); err != nil {
		return err
	}
	const max = math.MaxUint16
	r, g, b, a := clr.RGBA()
	gl.ClearColor(gl.GLclampf(float64(r)/max), gl.GLclampf(float64(g)/max), gl.GLclampf(float64(b)/max), gl.GLclampf(float64(a)/max))
	gl.Clear(gl.COLOR_BUFFER_BIT)
	return nil
}

func (i *ids) drawTexture(target *RenderTarget, texture *Texture, parts []TexturePart, geo GeometryMatrix, color ColorMatrix) error {
	glTexture := texture.glTexture
	if err := i.setViewportIfNeeded(target); err != nil {
		return err
	}
	projectionMatrix := target.glRenderTarget.ProjectionMatrix()
	quads := textureQuads(parts, glTexture.Width(), glTexture.Height())
	w, h := target.Size()
	shader.DrawTexture(glTexture.Native(), target.texture.glTexture.Native(), w, h, projectionMatrix, quads, &geo, &color)
	gl.Flush()
	return nil
}

func (i *ids) setViewportIfNeeded(renderTarget *RenderTarget) error {
	if i.currentRenderTarget != renderTarget {
		if err := renderTarget.glRenderTarget.SetAsViewport(); err != nil {
			return err
		}
		i.currentRenderTarget = renderTarget
	}
	return nil
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
