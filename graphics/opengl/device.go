package opengl

// #cgo LDFLAGS: -framework OpenGL
//
// #include <OpenGL/gl.h>
// #include <stdlib.h>
import "C"
import (
	"github.com/hajimehoshi/go-ebiten/graphics"
	"github.com/hajimehoshi/go-ebiten/graphics/matrix"
	"runtime"
)

type Device struct {
	screenScale int
	context     *Context
}

func NewDevice(screenWidth, screenHeight, screenScale int) *Device {
	graphicsContext := newContext(screenWidth, screenHeight, screenScale)
	device := &Device{
		screenScale: screenScale,
		context:     graphicsContext,
	}
	return device
}

func (device *Device) Init() {
	device.context.Init()
}

func (device *Device) Update(draw func(graphics.Context)) {
	context := device.context
	C.glEnable(C.GL_TEXTURE_2D)
	context.ResetOffscreen()
	context.Clear()

	draw(context)

	context.flush()
	context.setMainFramebufferOffscreen()
	context.Clear()

	scale := float64(context.screenScale)
	geometryMatrix := matrix.Geometry{
		[2][3]float64{
			{scale, 0, 0},
			{0, scale, 0},
		},
	}
	context.DrawTexture(context.ToTexture(context.screenId),
		geometryMatrix, matrix.IdentityColor())
	context.flush()
}

func (device *Device) TextureFactory() graphics.TextureFactory {
	return device.context
}

func init() {
	runtime.LockOSThread()
}
