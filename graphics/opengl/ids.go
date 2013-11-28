package opengl

import (
	"github.com/hajimehoshi/go-ebiten/graphics"
	"github.com/hajimehoshi/go-ebiten/graphics/opengl/rendertarget"
	"github.com/hajimehoshi/go-ebiten/graphics/opengl/texture"
	gtexture "github.com/hajimehoshi/go-ebiten/graphics/texture"
	"image"
)

type ids struct {
	textures              map[graphics.TextureId]*gtexture.Texture
	renderTargets         map[graphics.RenderTargetId]*gtexture.RenderTarget
	renderTargetToTexture map[graphics.RenderTargetId]graphics.TextureId
	counts                chan int
}

func newIds() *ids {
	ids := &ids{
		textures:              map[graphics.TextureId]*gtexture.Texture{},
		renderTargets:         map[graphics.RenderTargetId]*gtexture.RenderTarget{},
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

func (i *ids) TextureAt(id graphics.TextureId) *gtexture.Texture {
	return i.textures[id]
}

func (i *ids) RenderTargetAt(id graphics.RenderTargetId) *gtexture.RenderTarget {
	return i.renderTargets[id]
}

func (i *ids) ToTexture(id graphics.RenderTargetId) graphics.TextureId {
	return i.renderTargetToTexture[id]
}

func (i *ids) CreateTextureFromImage(img image.Image) (
	graphics.TextureId, error) {
	texture, err := texture.NewFromImage(img)
	if err != nil {
		return 0, err
	}
	textureId := graphics.TextureId(<-i.counts)
	i.textures[textureId] = texture
	return textureId, nil
}

func (i *ids) CreateRenderTarget(width, height int, filter texture.Filter) (
	graphics.RenderTargetId, error) {
	renderTarget, texture, err := rendertarget.New(width, height, filter)
	if err != nil {
		return 0, err
	}
	renderTargetId := graphics.RenderTargetId(<-i.counts)
	textureId := graphics.TextureId(<-i.counts)
	i.renderTargets[renderTargetId] = renderTarget
	i.textures[textureId] = texture
	i.renderTargetToTexture[renderTargetId] = textureId

	return renderTargetId, nil
}
