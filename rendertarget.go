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

type innerRenderTarget struct {
	glRenderTarget *opengl.RenderTarget
	texture        *Texture
}

func newInnerRenderTarget(width, height int, filter int) (*innerRenderTarget, error) {
	glTexture, err := opengl.NewTexture(width, height, filter)
	if err != nil {
		return nil, err
	}

	glRenderTarget, err := opengl.NewRenderTargetFromTexture(glTexture)
	if err != nil {
		return nil, err
	}

	texture := &Texture{glTexture}
	return &innerRenderTarget{glRenderTarget, texture}, nil
}

func (r *innerRenderTarget) size() (width, height int) {
	return r.glRenderTarget.Size()
}

func (r *innerRenderTarget) Clear() error {
	return r.Fill(color.RGBA{0, 0, 0, 0})
}

func (r *innerRenderTarget) Fill(clr color.Color) error {
	if err := r.glRenderTarget.SetAsViewport(); err != nil {
		return err
	}
	const max = math.MaxUint16
	cr, cg, cb, ca := clr.RGBA()
	rf := gl.GLclampf(float64(cr) / max)
	gf := gl.GLclampf(float64(cg) / max)
	bf := gl.GLclampf(float64(cb) / max)
	af := gl.GLclampf(float64(ca) / max)
	gl.ClearColor(rf, gf, bf, af)
	gl.Clear(gl.COLOR_BUFFER_BIT)
	return nil
}

func (r *innerRenderTarget) DrawTexture(texture *Texture, parts []TexturePart, geo GeometryMatrix, color ColorMatrix) error {
	if err := r.glRenderTarget.SetAsViewport(); err != nil {
		return err
	}
	glTexture := texture.glTexture
	w, h := glTexture.Size()
	quads := textureQuads(parts, w, h)
	targetNativeTexture := gl.Texture(0)
	if r.texture != nil {
		targetNativeTexture = r.texture.glTexture.Native()
	}
	w2, h2 := r.size()
	projectionMatrix := r.glRenderTarget.ProjectionMatrix()
	shader.DrawTexture(glTexture.Native(), targetNativeTexture, w2, h2, projectionMatrix, quads, &geo, &color)
	return nil
}

func u(x float64, width int) float32 {
	return float32(x) / float32(internal.NextPowerOf2Int(width))
}

func v(y float64, height int) float32 {
	return float32(y) / float32(internal.NextPowerOf2Int(height))
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

type syncer interface {
	Sync(func())
}

type RenderTarget struct {
	syncer syncer
	inner  *innerRenderTarget
}

func (r *RenderTarget) Texture() *Texture {
	return r.inner.texture
}

func (r *RenderTarget) Size() (width, height int) {
	return r.inner.size()
}

func (r *RenderTarget) Clear() (err error) {
	r.syncer.Sync(func() {
		err = r.inner.Clear()
	})
	return
}

func (r *RenderTarget) Fill(clr color.Color) (err error) {
	r.syncer.Sync(func() {
		err = r.inner.Fill(clr)
	})
	return
}

func (r *RenderTarget) DrawTexture(texture *Texture, parts []TexturePart, geo GeometryMatrix, color ColorMatrix) (err error) {
	r.syncer.Sync(func() {
		err = r.inner.DrawTexture(texture, parts, geo, color)
	})
	return
}
