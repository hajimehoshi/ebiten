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

func (renderTarget *RenderTarget) SetAsOffscreen(
	setter func(framebuffer interface{}, x, y, width, height int)) {
	renderTarget.texture.SetAsViewport(func(x, y, width, height int) {
		setter(renderTarget.framebuffer, x, y, width, height)
	})
}
