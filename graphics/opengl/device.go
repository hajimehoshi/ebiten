package opengl

import (
	"github.com/hajimehoshi/go-ebiten/graphics"
	"image"
)

type Device struct {
	ids *ids
}

func NewDevice() *Device {
	device := &Device{
		ids: newIds(),
	}
	return device
}

// called from window
func (d *Device) CreateContext(screenWidth, screenHeight, screenScale int) *Context {
	return newContext(d.ids, screenWidth, screenHeight, screenScale)
}

// called from window
func (d *Device) Update(context *Context, draw func(graphics.Context)) {
	context.update(draw)
}

// called from ui
func (d *Device) CreateRenderTarget(width, height int) (graphics.RenderTargetId, error) {
	return d.ids.CreateRenderTarget(width, height, graphics.FilterLinear)
}

// called from ui
func (d *Device) CreateTexture(img image.Image, filter graphics.Filter) (graphics.TextureId, error) {
	return d.ids.CreateTexture(img, filter)
}
