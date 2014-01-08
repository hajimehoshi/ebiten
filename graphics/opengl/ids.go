package opengl

import (
	"github.com/hajimehoshi/go-ebiten/graphics"
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

func (i *ids) TextureAt(id graphics.TextureId) *texture.Texture {
	i.lock.RLock()
	defer i.lock.RUnlock()
	return i.textures[id]
}

func (i *ids) RenderTargetAt(id graphics.RenderTargetId) *rendertarget.RenderTarget {
	i.lock.RLock()
	defer i.lock.RUnlock()
	return i.renderTargets[id]
}

func (i *ids) ToTexture(id graphics.RenderTargetId) graphics.TextureId {
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
	renderTarget, texture, err := rendertarget.Create(width, height, filter)
	if err != nil {
		return 0, err
	}
	renderTargetId := graphics.RenderTargetId(<-i.counts)
	textureId := graphics.TextureId(<-i.counts)

	i.lock.Lock()
	defer i.lock.Unlock()
	i.renderTargets[renderTargetId] = renderTarget
	i.textures[textureId] = texture
	i.renderTargetToTexture[renderTargetId] = textureId

	return renderTargetId, nil
}

func (i *ids) DeleteRenderTarget(id graphics.RenderTargetId) {
	renderTarget := i.renderTargets[id]
	renderTarget.Dispose()
	delete(i.renderTargets, id)
}
