package opengl

import (
	"github.com/go-gl/gl"
	"github.com/hajimehoshi/ebiten/graphics"
	"github.com/hajimehoshi/ebiten/graphics/matrix"
	"github.com/hajimehoshi/ebiten/graphics/opengl/internal/shader"
	"image"
	"math"
	"sync"
)

type ids struct {
	textures              map[graphics.TextureID]*texture
	renderTargets         map[graphics.RenderTargetID]*renderTarget
	renderTargetToTexture map[graphics.RenderTargetID]graphics.TextureID
	lastId                int
	currentRenderTargetId graphics.RenderTargetID
	sync.RWMutex
}

var idsInstance = &ids{
	textures:              map[graphics.TextureID]*texture{},
	renderTargets:         map[graphics.RenderTargetID]*renderTarget{},
	renderTargetToTexture: map[graphics.RenderTargetID]graphics.TextureID{},
	currentRenderTargetId: -1,
}

func NewRenderTargetID(width, height int, filter graphics.Filter) (graphics.RenderTargetID, error) {
	return idsInstance.newRenderTarget(width, height, filter)
}

func NewTextureID(img image.Image, filter graphics.Filter) (graphics.TextureID, error) {
	return idsInstance.newTexture(img, filter)
}

func (i *ids) textureAt(id graphics.TextureID) *texture {
	i.RLock()
	defer i.RUnlock()
	return i.textures[id]
}

func (i *ids) renderTargetAt(id graphics.RenderTargetID) *renderTarget {
	i.RLock()
	defer i.RUnlock()
	return i.renderTargets[id]
}

func (i *ids) toTexture(id graphics.RenderTargetID) graphics.TextureID {
	i.RLock()
	defer i.RUnlock()
	return i.renderTargetToTexture[id]
}

func (i *ids) newTexture(img image.Image, filter graphics.Filter) (graphics.TextureID, error) {
	texture, err := newTextureFromImage(img, filter)
	if err != nil {
		return 0, err
	}

	i.Lock()
	defer i.Unlock()
	i.lastId++
	textureId := graphics.TextureID(i.lastId)
	i.textures[textureId] = texture
	return textureId, nil
}

func (i *ids) newRenderTarget(width, height int, filter graphics.Filter) (graphics.RenderTargetID, error) {
	texture, err := newTexture(width, height, filter)
	if err != nil {
		return 0, err
	}
	framebuffer := newFramebuffer(gl.Texture(texture.native))
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
	textureId := graphics.TextureID(i.lastId)
	i.lastId++
	renderTargetId := graphics.RenderTargetID(i.lastId)

	i.textures[textureId] = texture
	i.renderTargets[renderTargetId] = r
	i.renderTargetToTexture[renderTargetId] = textureId

	return renderTargetId, nil
}

// NOTE: renderTarget can't be used as a texture.
func (i *ids) addRenderTarget(renderTarget *renderTarget) graphics.RenderTargetID {
	i.Lock()
	defer i.Unlock()
	i.lastId++
	id := graphics.RenderTargetID(i.lastId)
	i.renderTargets[id] = renderTarget

	return id
}

func (i *ids) deleteRenderTarget(id graphics.RenderTargetID) {
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

func (i *ids) fillRenderTarget(id graphics.RenderTargetID, r, g, b uint8) {
	i.setViewportIfNeeded(id)
	const max = float64(math.MaxUint8)
	gl.ClearColor(gl.GLclampf(float64(r)/max), gl.GLclampf(float64(g)/max), gl.GLclampf(float64(b)/max), 1)
	gl.Clear(gl.COLOR_BUFFER_BIT)
}

func (i *ids) drawTexture(
	target graphics.RenderTargetID,
	id graphics.TextureID,
	parts []graphics.TexturePart,
	geo matrix.Geometry,
	color matrix.Color) {
	texture := i.textureAt(id)
	i.setViewportIfNeeded(target)
	r := i.renderTargetAt(target)
	projectionMatrix := r.projectionMatrix()
	quads := shader.TextureQuads(parts, texture.width, texture.height)
	shader.DrawTexture(texture.native, projectionMatrix, quads, geo, color)
}

func (i *ids) setViewportIfNeeded(id graphics.RenderTargetID) {
	r := i.renderTargetAt(id)
	if i.currentRenderTargetId != id {
		r.setAsViewport()
		i.currentRenderTargetId = id
	}
}
