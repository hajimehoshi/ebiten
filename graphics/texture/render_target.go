package texture

type RenderTarget struct {
	framebuffer     interface{}
	offscreenWidth  int
	offscreenHeight int
}

func NewRenderTarget(framebuffer interface{}, width, height int) *RenderTarget {
	return &RenderTarget{
		framebuffer:     framebuffer,
		offscreenWidth:  AdjustSize(width),
		offscreenHeight: AdjustSize(height),
	}
}

type OffscreenSetter interface {
	Set(framebuffer interface{}, x, y, width, height int)
}

func (r *RenderTarget) SetAsOffscreen(setter OffscreenSetter) {
	setter.Set(r.framebuffer, 0, 0, r.offscreenWidth, r.offscreenHeight)
}

type RenderTargetDisposer interface {
	Dispose(framebuffer interface{})
}

func (r *RenderTarget) Dispose(disposer RenderTargetDisposer) {
	disposer.Dispose(r.framebuffer)
}
