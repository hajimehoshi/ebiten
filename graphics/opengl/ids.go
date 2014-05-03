package opengl

import (
	"github.com/hajimehoshi/go-ebiten/graphics"
	"github.com/hajimehoshi/go-ebiten/graphics/matrix"
	"image"
	"sync"
)

var idsInstance *ids = newIds()

func NewContext(screenWidth, screenHeight, screenScale int) *Context {
	return newContext(idsInstance, screenWidth, screenHeight, screenScale)
}

func CreateRenderTarget(
	width, height int,
	filter graphics.Filter) (graphics.RenderTargetId, error) {
	return idsInstance.createRenderTarget(width, height, filter)
}

func CreateTexture(
	img image.Image,
	filter graphics.Filter) (graphics.TextureId, error) {
	return idsInstance.createTexture(img, filter)
}

type ids struct {
	textures              map[graphics.TextureId]*Texture
	renderTargets         map[graphics.RenderTargetId]*RenderTarget
	renderTargetToTexture map[graphics.RenderTargetId]graphics.TextureId
	counts                chan int
	sync.RWMutex
}

func newIds() *ids {
	ids := &ids{
		textures:              map[graphics.TextureId]*Texture{},
		renderTargets:         map[graphics.RenderTargetId]*RenderTarget{},
		renderTargetToTexture: map[graphics.RenderTargetId]graphics.TextureId{},
		counts:                make(chan int),
	}
	go func() {
		for i := 1; ; i++ {
			ids.counts <- i
		}
	}()
	return ids
}

func (i *ids) textureAt(id graphics.TextureId) *Texture {
	i.RLock()
	defer i.RUnlock()
	return i.textures[id]
}

func (i *ids) renderTargetAt(id graphics.RenderTargetId) *RenderTarget {
	i.RLock()
	defer i.RUnlock()
	return i.renderTargets[id]
}

func (i *ids) toTexture(id graphics.RenderTargetId) graphics.TextureId {
	i.RLock()
	defer i.RUnlock()
	return i.renderTargetToTexture[id]
}

func (i *ids) createTexture(img image.Image, filter graphics.Filter) (
	graphics.TextureId, error) {
	texture, err := createTextureFromImage(img, filter)
	if err != nil {
		return 0, err
	}
	textureId := graphics.TextureId(<-i.counts)

	i.Lock()
	defer i.Unlock()
	i.textures[textureId] = texture
	return textureId, nil
}

func (i *ids) createRenderTarget(width, height int, filter graphics.Filter) (
	graphics.RenderTargetId, error) {

	texture, err := createTexture(width, height, filter)
	if err != nil {
		return 0, err
	}
	framebuffer := createFramebuffer(texture.native)
	renderTarget := &RenderTarget{framebuffer, texture.width, texture.height, false}

	textureId := graphics.TextureId(<-i.counts)
	renderTargetId := graphics.RenderTargetId(<-i.counts)

	i.Lock()
	defer i.Unlock()
	i.textures[textureId] = texture
	i.renderTargets[renderTargetId] = renderTarget
	i.renderTargetToTexture[renderTargetId] = textureId

	return renderTargetId, nil
}

// NOTE: renderTarget can't be used as a texture.
func (i *ids) addRenderTarget(renderTarget *RenderTarget) graphics.RenderTargetId {
	id := graphics.RenderTargetId(<-i.counts)

	i.Lock()
	defer i.Unlock()
	i.renderTargets[id] = renderTarget

	return id
}

func (i *ids) deleteRenderTarget(id graphics.RenderTargetId) {
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

func (i *ids) fillRenderTarget(id graphics.RenderTargetId, r, g, b uint8) {
	i.renderTargetAt(id).fill(r, g, b)
}

func (i *ids) drawTexture(
	target graphics.RenderTargetId,
	id graphics.TextureId,
	parts []graphics.TexturePart, geo matrix.Geometry, color matrix.Color) {
	texture := i.textureAt(id)
	i.renderTargetAt(target).drawTexture(texture, parts, geo, color)
}
