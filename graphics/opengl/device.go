package opengl

import (
	"github.com/hajimehoshi/go-ebiten/graphics"
	"github.com/hajimehoshi/go-ebiten/graphics/matrix"
	"image"
)

type Device struct {
	canvas      *Canvas
	screenScale int
}

func NewDevice(screenWidth, screenHeight, screenScale int) *Device {
	canvas := newCanvas(screenWidth, screenHeight, screenScale)
	return &Device{
		canvas:     canvas,
		screenScale: screenScale,
	}
}

func (d *Device) Update(draw func(graphics.Canvas)) {
	canvas := d.canvas
	canvas.Init()
	canvas.ResetOffscreen()
	canvas.Clear()

	draw(canvas)

	canvas.flush()
	canvas.setMainFramebufferOffscreen()
	canvas.Clear()

	scale := float64(d.screenScale)
	geometryMatrix := matrix.IdentityGeometry()
	geometryMatrix.Scale(scale, scale)
	canvas.DrawRenderTarget(canvas.screenId,
		geometryMatrix, matrix.IdentityColor())
	canvas.flush()
}

func (d *Device) CreateRenderTarget(width, height int) (graphics.RenderTargetId, error) {
	return d.canvas.CreateRenderTarget(width, height)
}

func (d *Device) CreateTexture(img image.Image) (graphics.TextureId, error) {
	return d.canvas.CreateTextureFromImage(img)
}
