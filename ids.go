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
	"github.com/hajimehoshi/ebiten/internal/opengl"
	"github.com/hajimehoshi/ebiten/internal/opengl/internal/shader"
	"math"
)

type ids struct {
	currentRenderTarget *RenderTarget
}

var idsInstance = &ids{}

func (i *ids) toTexture(renderTarget *RenderTarget) *Texture {
	return renderTarget.texture
}

func (i *ids) createRenderTarget(width, height int, filter int) (*RenderTarget, error) {
	glTexture, err := opengl.CreateTexture(width, height, filter)
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
	renderTarget := &RenderTarget{glRenderTarget, texture}

	return renderTarget, nil
}

// NOTE: renderTarget can't be used as a texture.
func (i *ids) addRenderTarget(glRenderTarget *opengl.RenderTarget) *RenderTarget {
	return &RenderTarget{glRenderTarget, nil}
}

func (i *ids) deleteRenderTarget(renderTarget *RenderTarget) {

	glRenderTarget := renderTarget.glRenderTarget
	texture := renderTarget.texture
	glTexture := texture.glTexture

	glRenderTarget.Dispose()
	glTexture.Dispose()
}

func (i *ids) fillRenderTarget(renderTarget *RenderTarget, r, g, b uint8) error {
	if err := i.setViewportIfNeeded(renderTarget); err != nil {
		return err
	}
	const max = float64(math.MaxUint8)
	gl.ClearColor(gl.GLclampf(float64(r)/max), gl.GLclampf(float64(g)/max), gl.GLclampf(float64(b)/max), 1)
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
	shader.DrawTexture(glTexture.Native(), projectionMatrix, quads, &geo, &color)
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
