package rendertarget

import (
	"github.com/hajimehoshi/go-ebiten/graphics/texture"
)

type OffscreenSetter interface {
	Set(framebuffer interface{}, x, y, width, height int)
}

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

func (renderTarget *RenderTarget) SetAsOffscreen(setter OffscreenSetter) {
	renderTarget.texture.SetAsViewport(func(x, y, width, height int) {
		setter.Set(renderTarget.framebuffer, x, y, width, height)
	})
}
