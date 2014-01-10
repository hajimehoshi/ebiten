package opengl

import (
	"github.com/hajimehoshi/go-ebiten/graphics"
	"github.com/hajimehoshi/go-ebiten/graphics/matrix"
	"github.com/hajimehoshi/go-ebiten/graphics/opengl/rendertarget"
	"github.com/hajimehoshi/go-ebiten/graphics/opengl/texture"
	"image"
	"sync"
)

type ids struct {
	lock                  sync.RWMutex
	textures              map[graphics.TextureId]*texture.Texture
	renderTargets         map[graphics.RenderTargetId]*rendertarget.RenderTarget
	renderTargetToTexture map[graphics.RenderTargetId]graphics.TextureId
	counts                chan int
}

func newIds() *ids {
	ids := &ids{
		textures:              map[graphics.TextureId]*texture.Texture{},
		renderTargets:         map[graphics.RenderTargetId]*rendertarget.RenderTarget{},
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

func (i *ids) textureAt(id graphics.TextureId) *texture.Texture {
	i.lock.RLock()
	defer i.lock.RUnlock()
	return i.textures[id]
}

func (i *ids) renderTargetAt(id graphics.RenderTargetId) *rendertarget.RenderTarget {
	i.lock.RLock()
	defer i.lock.RUnlock()
	return i.renderTargets[id]
}

func (i *ids) toTexture(id graphics.RenderTargetId) graphics.TextureId {
	i.lock.RLock()
	defer i.lock.RUnlock()
	return i.renderTargetToTexture[id]
}

func (i *ids) CreateTexture(img image.Image, filter graphics.Filter) (
	graphics.TextureId, error) {
	texture, err := texture.CreateFromImage(img, filter)
	if err != nil {
		return 0, err
	}
	textureId := graphics.TextureId(<-i.counts)

	i.lock.Lock()
	defer i.lock.Unlock()
	i.textures[textureId] = texture
	return textureId, nil
}

func (i *ids) CreateRenderTarget(width, height int, filter graphics.Filter) (
	graphics.RenderTargetId, error) {

	texture, err := texture.Create(width, height, filter)
	if err != nil {
		return 0, err
	}
	renderTarget := texture.CreateRenderTarget()

	textureId := graphics.TextureId(<-i.counts)
	renderTargetId := graphics.RenderTargetId(<-i.counts)

	i.lock.Lock()
	defer i.lock.Unlock()
	i.textures[textureId] = texture
	i.renderTargets[renderTargetId] = renderTarget
	i.renderTargetToTexture[renderTargetId] = textureId

	return renderTargetId, nil
}

// NOTE: renderTarget can't be used as a texture.
func (i *ids) AddRenderTarget(renderTarget *rendertarget.RenderTarget) graphics.RenderTargetId {
	id := graphics.RenderTargetId(<-i.counts)

	i.lock.Lock()
	defer i.lock.Unlock()
	i.renderTargets[id] = renderTarget

	return id
}

func (i *ids) DeleteRenderTarget(id graphics.RenderTargetId) {
	i.lock.Lock()
	defer i.lock.Unlock()

	renderTarget := i.renderTargets[id]
	textureId := i.renderTargetToTexture[id]
	texture := i.textures[textureId]

	renderTarget.Dispose()
	texture.Dispose()

	delete(i.renderTargets, id)
	delete(i.renderTargetToTexture, id)
	delete(i.textures, textureId)
}

func (i *ids) FillRenderTarget(id graphics.RenderTargetId, r, g, b uint8) {
	i.renderTargetAt(id).Fill(r, g, b)
}

func (i *ids) DrawTexture(target graphics.RenderTargetId, id graphics.TextureId,
	geo matrix.Geometry, color matrix.Color) {
	texture := i.textureAt(id)
	i.renderTargetAt(target).DrawTexture(texture, geo, color)
}

func (i *ids) DrawTextureParts(target graphics.RenderTargetId, id graphics.TextureId,
	parts []graphics.TexturePart, geo matrix.Geometry, color matrix.Color) {
	texture := i.textureAt(id)
	i.renderTargetAt(target).DrawTextureParts(texture, parts, geo, color)
}

func (i *ids) DrawRenderTarget(target graphics.RenderTargetId, id graphics.RenderTargetId,
	geo matrix.Geometry, color matrix.Color) {
	i.DrawTexture(target, i.toTexture(id), geo, color)
}

func (i *ids) DrawRenderTargetParts(target graphics.RenderTargetId, id graphics.RenderTargetId,
	parts []graphics.TexturePart, geo matrix.Geometry, color matrix.Color) {
	i.DrawTextureParts(target, i.toTexture(id), parts, geo, color)
}
