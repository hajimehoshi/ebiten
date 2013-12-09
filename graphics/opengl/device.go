package opengl

import (
	"github.com/hajimehoshi/go-ebiten/graphics"
	"github.com/hajimehoshi/go-ebiten/graphics/matrix"
	"image"
)

type Device struct {
	context     *Context
	screenScale int
}

func NewDevice(screenWidth, screenHeight, screenScale int) *Device {
	context := newContext(screenWidth, screenHeight, screenScale)
	return &Device{
		context:     context,
		screenScale: screenScale,
	}
}

func (d *Device) Update(draw func(graphics.Canvas)) {
	context := d.context
	context.Init()
	context.ResetOffscreen()
	context.Clear()

	draw(context)

	context.flush()
	context.setMainFramebufferOffscreen()
	context.Clear()

	scale := float64(d.screenScale)
	geometryMatrix := matrix.IdentityGeometry()
	geometryMatrix.Scale(scale, scale)
	context.DrawRenderTarget(context.screenId,
		geometryMatrix, matrix.IdentityColor())
	context.flush()
}

func (d *Device) CreateRenderTarget(width, height int) (graphics.RenderTargetId, error) {
	return d.context.CreateRenderTarget(width, height)
}

func (d *Device) CreateTexture(img image.Image) (graphics.TextureId, error) {
	return d.context.CreateTextureFromImage(img)
}
