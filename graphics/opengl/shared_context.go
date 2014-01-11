package opengl

import (
	"github.com/hajimehoshi/go-ebiten/graphics"
	"image"
)

type SharedContext struct {
	ids *ids
}

var sharedContext *SharedContext = nil

func Initialize() *SharedContext {
	if sharedContext != nil {
		panic("OpenGL is already initialized")
	}
	sharedContext = &SharedContext{
		ids: newIds(),
	}
	return sharedContext
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
