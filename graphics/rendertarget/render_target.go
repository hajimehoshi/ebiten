package rendertarget

import (
	"github.com/hajimehoshi/go-ebiten/graphics/texture"
)

type RenderTarget struct {
	texture     *texture.Texture
	framebuffer interface{}
}

func NewWithFramebuffer(texture *texture.Texture, framebuffer interface{}) *RenderTarget {
	return &RenderTarget{
		texture:     texture,
		framebuffer: framebuffer,
	}
}

func (renderTarget *RenderTarget) SetAsViewport(setter func(x, y, width, height int)) {
	renderTarget.texture.SetAsViewport(setter)
}

func (renderTarget *RenderTarget) SetAsOffscreen(setter func(framebuffer interface{})) {
	setter(renderTarget.framebuffer)
}
