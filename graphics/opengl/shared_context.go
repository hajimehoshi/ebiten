package opengl

import (
	"github.com/hajimehoshi/go-ebiten/graphics"
	"image"
)

type SharedContext struct {
	ids *ids
}

func NewSharedContext() *SharedContext {
	return &SharedContext{
		ids: newIds(),
	}
}

func (s *SharedContext) CreateContext(screenWidth, screenHeight, screenScale int) *Context {
	return newContext(s.ids, screenWidth, screenHeight, screenScale)
}

func (s *SharedContext) CreateRenderTarget(width, height int) (graphics.RenderTargetId, error) {
	return s.ids.CreateRenderTarget(width, height, graphics.FilterLinear)
}

func (s *SharedContext) CreateTexture(img image.Image, filter graphics.Filter) (graphics.TextureId, error) {
	return s.ids.CreateTexture(img, filter)
}
