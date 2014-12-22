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
)

type innerImage struct {
	renderTarget *opengl.Image
	texture      *opengl.Texture
}

func newInnerImage(texture *opengl.Texture, filter int) (*innerImage, error) {
	renderTarget, err := opengl.NewRenderTargetFromTexture(texture)
	if err != nil {
		return nil, err
	}
	return &innerImage{renderTarget, texture}, nil
}

func (i *innerImage) size() (width, height int) {
	return i.renderTarget.Size()
}

func (i *innerImage) Clear() error {
	return i.Fill(color.Transparent)
}

func (i *innerImage) Fill(clr color.Color) error {
	if err := i.renderTarget.SetAsViewport(); err != nil {
		return err
	}
	rf, gf, bf, af := internal.RGBA(clr)
	gl.ClearColor(gl.GLclampf(rf), gl.GLclampf(gf), gl.GLclampf(bf), gl.GLclampf(af))
	gl.Clear(gl.COLOR_BUFFER_BIT)
	return nil
}

func (i *innerImage) drawImage(image *innerImage, parts []ImagePart, geo GeometryMatrix, color ColorMatrix) error {
	if err := i.renderTarget.SetAsViewport(); err != nil {
		return err
	}
	w, h := image.texture.Size()
	quads := textureQuads(parts, w, h)
	targetNativeTexture := gl.Texture(0)
	if i.texture != nil {
		targetNativeTexture = i.texture.Native()
	}
	w2, h2 := i.size()
	projectionMatrix := i.renderTarget.ProjectionMatrix()
	shader.DrawTexture(image.texture.Native(), targetNativeTexture, w2, h2, projectionMatrix, quads, &geo, &color)
	return nil
}

func u(x float64, width int) float32 {
	return float32(x) / float32(internal.NextPowerOf2Int(width))
}

func v(y float64, height int) float32 {
	return float32(y) / float32(internal.NextPowerOf2Int(height))
}

func textureQuads(parts []ImagePart, width, height int) []shader.TextureQuad {
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

// Image represents an image.
// The pixel format is non alpha-premultiplied.
type Image struct {
	syncer syncer
	inner  *innerImage
}

// Size returns the size of the image.
func (i *Image) Size() (width, height int) {
	return i.inner.size()
}

// Clear resets the pixels of the image into 0.
func (i *Image) Clear() (err error) {
	i.syncer.Sync(func() {
		err = i.inner.Clear()
	})
	return
}

// Fill fills the image with a solid color.
func (i *Image) Fill(clr color.Color) (err error) {
	i.syncer.Sync(func() {
		err = i.inner.Fill(clr)
	})
	return
}

// DrawImage draws the given image on the receiver (i).
func (i *Image) DrawImage(image *Image, parts []ImagePart, geo GeometryMatrix, color ColorMatrix) (err error) {
	return i.drawImage(image.inner, parts, geo, color)
}

func (i *Image) drawImage(image *innerImage, parts []ImagePart, geo GeometryMatrix, color ColorMatrix) (err error) {
	i.syncer.Sync(func() {
		err = i.inner.drawImage(image, parts, geo, color)
	})
	return
}
