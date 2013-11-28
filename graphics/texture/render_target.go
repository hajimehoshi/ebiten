package texture

type RenderTarget struct {
	texture     *Texture
	framebuffer interface{}
}

func NewRenderTarget(texture *Texture, framebuffer interface{}) *RenderTarget {
	return &RenderTarget{
		texture:     texture,
		framebuffer: framebuffer,
	}
}

type OffscreenSetter interface {
	Set(framebuffer interface{}, x, y, width, height int)
}

func (renderTarget *RenderTarget) SetAsOffscreen(setter OffscreenSetter) {
	renderTarget.texture.SetAsViewport(func(x, y, width, height int) {
		setter.Set(renderTarget.framebuffer, x, y, width, height)
	})
}
