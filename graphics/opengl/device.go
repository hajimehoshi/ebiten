package opengl

// #cgo LDFLAGS: -framework OpenGL
//
// #include <OpenGL/gl.h>
// #include <stdlib.h>
import "C"
import (
	"github.com/hajimehoshi/go.ebiten/graphics"
)

type Device struct {
	screenWidth      int
	screenHeight     int
	screenScale      int
	graphicsContext  *GraphicsContext
	offscreenTexture graphics.Texture
	drawFunc         func(graphics.GraphicsContext, graphics.TextureID)
}

func NewDevice(screenWidth, screenHeight, screenScale int,
	drawFunc func(graphics.GraphicsContext, graphics.TextureID)) *Device {
	device := &Device{
		screenWidth:     screenWidth,
		screenHeight:    screenHeight,
		screenScale:     screenScale,
		graphicsContext: newGraphicsContext(screenWidth, screenHeight, screenScale),
		drawFunc:        drawFunc,
	}
	device.offscreenTexture =
		device.graphicsContext.NewTexture(screenWidth, screenHeight)
	return device
}

func (device *Device) Update() {
	g := device.graphicsContext
	C.glEnable(C.GL_TEXTURE_2D)
	C.glTexParameteri(C.GL_TEXTURE_2D, C.GL_TEXTURE_MIN_FILTER, C.GL_NEAREST)
	C.glTexParameteri(C.GL_TEXTURE_2D, C.GL_TEXTURE_MAG_FILTER, C.GL_NEAREST)
	g.SetOffscreen(device.offscreenTexture.ID)
	g.Clear()
	device.drawFunc(g, device.offscreenTexture.ID)
	g.flush()

	C.glTexParameteri(C.GL_TEXTURE_2D, C.GL_TEXTURE_MIN_FILTER, C.GL_LINEAR)
	C.glTexParameteri(C.GL_TEXTURE_2D, C.GL_TEXTURE_MAG_FILTER, C.GL_LINEAR)
	g.resetOffscreen()
	g.Clear()
	geometryMatrix := graphics.IdentityGeometryMatrix()
	geometryMatrix.SetA(float64(g.screenScale))
	geometryMatrix.SetD(float64(g.screenScale))
	g.DrawTexture(device.offscreenTexture.ID,
		0, 0, device.screenWidth, device.screenHeight,
		geometryMatrix, graphics.IdentityColorMatrix())
	g.flush()
}

func (device *Device) TextureFactory() graphics.TextureFactory {
	return device.graphicsContext
}
