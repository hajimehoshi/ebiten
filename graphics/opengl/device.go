package opengl

import (
	"github.com/hajimehoshi/go-ebiten/graphics"
	"github.com/hajimehoshi/go-ebiten/graphics/matrix"
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

func (device *Device) Update(draw func(graphics.Canvas)) {
	context := device.context
	context.Init()
	context.ResetOffscreen()
	context.Clear()

	draw(context)

	context.flush()
	context.setMainFramebufferOffscreen()
	context.Clear()

	scale := float64(device.screenScale)
	geometryMatrix := matrix.IdentityGeometry()
	geometryMatrix.Scale(scale, scale)
	context.DrawTexture(context.ToTexture(context.screenId),
		geometryMatrix, matrix.IdentityColor())
	context.flush()
}

func (device *Device) TextureFactory() graphics.TextureFactory {
	return device.context
}
