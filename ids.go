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
	"image"
	"math"
	"sync"
)

type ids struct {
	textures              map[TextureID]*opengl.Texture
	renderTargets         map[RenderTargetID]*opengl.RenderTarget
	renderTargetToTexture map[RenderTargetID]TextureID
	lastId                int
	currentRenderTargetId RenderTargetID
	sync.RWMutex
}

var idsInstance = &ids{
	textures:              map[TextureID]*opengl.Texture{},
	renderTargets:         map[RenderTargetID]*opengl.RenderTarget{},
	renderTargetToTexture: map[RenderTargetID]TextureID{},
	currentRenderTargetId: -1,
}

func (i *ids) textureAt(id TextureID) *opengl.Texture {
	i.RLock()
	defer i.RUnlock()
	return i.textures[id]
}

func (i *ids) renderTargetAt(id RenderTargetID) *opengl.RenderTarget {
	i.RLock()
	defer i.RUnlock()
	return i.renderTargets[id]
}

func (i *ids) toTexture(id RenderTargetID) TextureID {
	i.RLock()
	defer i.RUnlock()
	return i.renderTargetToTexture[id]
}

func (i *ids) createTexture(img image.Image, filter int) (TextureID, error) {
	texture, err := opengl.CreateTextureFromImage(img, filter)
	if err != nil {
		return 0, err
	}

	i.Lock()
	defer i.Unlock()
	i.lastId++
	textureId := TextureID(i.lastId)
	i.textures[textureId] = texture
	return textureId, nil
}

func (i *ids) createRenderTarget(width, height int, filter int) (RenderTargetID, error) {
	texture, err := opengl.CreateTexture(width, height, filter)
	if err != nil {
		return 0, err
	}

	// The current binded framebuffer can be changed.
	i.currentRenderTargetId = -1
	r, err := opengl.NewRenderTargetFromTexture(texture)
	if err != nil {
		return 0, err
	}

	i.Lock()
	defer i.Unlock()
	i.lastId++
	textureId := TextureID(i.lastId)
	i.lastId++
	renderTargetId := RenderTargetID(i.lastId)

	i.textures[textureId] = texture
	i.renderTargets[renderTargetId] = r
	i.renderTargetToTexture[renderTargetId] = textureId

	return renderTargetId, nil
}

// NOTE: renderTarget can't be used as a texture.
func (i *ids) addRenderTarget(renderTarget *opengl.RenderTarget) RenderTargetID {
	i.Lock()
	defer i.Unlock()
	i.lastId++
	id := RenderTargetID(i.lastId)
	i.renderTargets[id] = renderTarget

	return id
}

func (i *ids) deleteRenderTarget(id RenderTargetID) {
	i.Lock()
	defer i.Unlock()

	renderTarget := i.renderTargets[id]
	textureId := i.renderTargetToTexture[id]
	texture := i.textures[textureId]

	renderTarget.Dispose()
	texture.Dispose()

	delete(i.renderTargets, id)
	delete(i.renderTargetToTexture, id)
	delete(i.textures, textureId)
}

func (i *ids) fillRenderTarget(id RenderTargetID, r, g, b uint8) error {
	if err := i.setViewportIfNeeded(id); err != nil {
		return err
	}
	const max = float64(math.MaxUint8)
	gl.ClearColor(gl.GLclampf(float64(r)/max), gl.GLclampf(float64(g)/max), gl.GLclampf(float64(b)/max), 1)
	gl.Clear(gl.COLOR_BUFFER_BIT)
	return nil
}

func (i *ids) drawTexture(target RenderTargetID, id TextureID, parts []TexturePart, geo GeometryMatrix, color ColorMatrix) error {
	texture := i.textureAt(id)
	if err := i.setViewportIfNeeded(target); err != nil {
		return err
	}
	r := i.renderTargetAt(target)
	projectionMatrix := r.ProjectionMatrix()
	quads := textureQuads(parts, texture.Width(), texture.Height())
	shader.DrawTexture(texture.Native(), projectionMatrix, quads, &geo, &color)
	return nil
}

func (i *ids) setViewportIfNeeded(id RenderTargetID) error {
	r := i.renderTargetAt(id)
	if i.currentRenderTargetId != id {
		if err := r.SetAsViewport(); err != nil {
			return err
		}
		i.currentRenderTargetId = id
	}
	return nil
}
