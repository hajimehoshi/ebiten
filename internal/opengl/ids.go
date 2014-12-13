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

package opengl

import (
	"github.com/go-gl/gl"
	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/internal/opengl/internal/shader"
	"image"
	"math"
	"sync"
)

type ids struct {
	textures              map[ebiten.TextureID]*texture
	renderTargets         map[ebiten.RenderTargetID]*renderTarget
	renderTargetToTexture map[ebiten.RenderTargetID]ebiten.TextureID
	lastId                int
	currentRenderTargetId ebiten.RenderTargetID
	sync.RWMutex
}

var idsInstance = &ids{
	textures:              map[ebiten.TextureID]*texture{},
	renderTargets:         map[ebiten.RenderTargetID]*renderTarget{},
	renderTargetToTexture: map[ebiten.RenderTargetID]ebiten.TextureID{},
	currentRenderTargetId: -1,
}

func NewRenderTargetID(width, height int, filter int) (ebiten.RenderTargetID, error) {
	return idsInstance.createRenderTarget(width, height, filter)
}

func NewTextureID(img image.Image, filter int) (ebiten.TextureID, error) {
	return idsInstance.createTexture(img, filter)
}

func (i *ids) textureAt(id ebiten.TextureID) *texture {
	i.RLock()
	defer i.RUnlock()
	return i.textures[id]
}

func (i *ids) renderTargetAt(id ebiten.RenderTargetID) *renderTarget {
	i.RLock()
	defer i.RUnlock()
	return i.renderTargets[id]
}

func (i *ids) toTexture(id ebiten.RenderTargetID) ebiten.TextureID {
	i.RLock()
	defer i.RUnlock()
	return i.renderTargetToTexture[id]
}

func (i *ids) createTexture(img image.Image, filter int) (ebiten.TextureID, error) {
	texture, err := createTextureFromImage(img, filter)
	if err != nil {
		return 0, err
	}

	i.Lock()
	defer i.Unlock()
	i.lastId++
	textureId := ebiten.TextureID(i.lastId)
	i.textures[textureId] = texture
	return textureId, nil
}

func (i *ids) createRenderTarget(width, height int, filter int) (ebiten.RenderTargetID, error) {
	texture, err := createTexture(width, height, filter)
	if err != nil {
		return 0, err
	}
	framebuffer := createFramebuffer(gl.Texture(texture.native))
	// The current binded framebuffer can be changed.
	i.currentRenderTargetId = -1
	r := &renderTarget{
		framebuffer: framebuffer,
		width:       texture.width,
		height:      texture.height,
	}

	i.Lock()
	defer i.Unlock()
	i.lastId++
	textureId := ebiten.TextureID(i.lastId)
	i.lastId++
	renderTargetId := ebiten.RenderTargetID(i.lastId)

	i.textures[textureId] = texture
	i.renderTargets[renderTargetId] = r
	i.renderTargetToTexture[renderTargetId] = textureId

	return renderTargetId, nil
}

// NOTE: renderTarget can't be used as a texture.
func (i *ids) addRenderTarget(renderTarget *renderTarget) ebiten.RenderTargetID {
	i.Lock()
	defer i.Unlock()
	i.lastId++
	id := ebiten.RenderTargetID(i.lastId)
	i.renderTargets[id] = renderTarget

	return id
}

func (i *ids) deleteRenderTarget(id ebiten.RenderTargetID) {
	i.Lock()
	defer i.Unlock()

	renderTarget := i.renderTargets[id]
	textureId := i.renderTargetToTexture[id]
	texture := i.textures[textureId]

	renderTarget.dispose()
	texture.dispose()

	delete(i.renderTargets, id)
	delete(i.renderTargetToTexture, id)
	delete(i.textures, textureId)
}

func (i *ids) fillRenderTarget(id ebiten.RenderTargetID, r, g, b uint8) {
	i.setViewportIfNeeded(id)
	const max = float64(math.MaxUint8)
	gl.ClearColor(gl.GLclampf(float64(r)/max), gl.GLclampf(float64(g)/max), gl.GLclampf(float64(b)/max), 1)
	gl.Clear(gl.COLOR_BUFFER_BIT)
}

func (i *ids) drawTexture(target ebiten.RenderTargetID, id ebiten.TextureID, parts []ebiten.TexturePart, geo ebiten.GeometryMatrix, color ebiten.ColorMatrix) {
	texture := i.textureAt(id)
	i.setViewportIfNeeded(target)
	r := i.renderTargetAt(target)
	projectionMatrix := r.projectionMatrix()
	quads := textureQuads(parts, texture.width, texture.height)
	shader.DrawTexture(texture.native, projectionMatrix, quads, &geo, &color)
}

func (i *ids) setViewportIfNeeded(id ebiten.RenderTargetID) {
	r := i.renderTargetAt(id)
	if i.currentRenderTargetId != id {
		r.setAsViewport()
		i.currentRenderTargetId = id
	}
}
