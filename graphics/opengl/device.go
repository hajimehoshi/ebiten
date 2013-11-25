package opengl

import (
	"github.com/hajimehoshi/go-ebiten/graphics"
	"github.com/hajimehoshi/go-ebiten/graphics/matrix"
)

type Device struct {
	context *Context
}

func NewDevice(screenWidth, screenHeight, screenScale int) *Device {
	context := newContext(screenWidth, screenHeight, screenScale)
	return &Device{
		context: context,
	}
}

func (device *Device) Update(draw func(graphics.Context)) {
	context := device.context
	context.Init()
	context.ResetOffscreen()
	context.Clear()

	draw(context)

	context.flush()
	context.setMainFramebufferOffscreen()
	context.Clear()

	scale := float64(context.screenScale)
	geometryMatrix := matrix.IdentityGeometry()
	geometryMatrix.Scale(scale, scale)
	context.DrawTexture(context.ToTexture(context.screenId),
		geometryMatrix, matrix.IdentityColor())
	context.flush()
}

func (device *Device) TextureFactory() graphics.TextureFactory {
	return device.context
}
