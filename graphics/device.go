package graphics

// #cgo LDFLAGS: -framework OpenGL
//
// #include <OpenGL/gl.h>
// #include <stdlib.h>
import "C"

type Device struct {
	screenWidth int
	screenHeight int
	screenScale int
	graphicsContext *GraphicsContext
	offscreenTexture *Texture
	drawFunc func(*GraphicsContext, *Texture)
}

func NewDevice(screenWidth, screenHeight, screenScale int,
	drawFunc func(*GraphicsContext, *Texture)) *Device {
	device := &Device{
		screenWidth: screenWidth,
		screenHeight: screenHeight,
		screenScale: screenScale,
		graphicsContext: newGraphicsContext(screenWidth, screenHeight, screenScale),
		offscreenTexture: NewTexture(screenWidth, screenHeight),
		drawFunc: drawFunc,
	}
	return device
}

func (device *Device) Update() {
	g := device.graphicsContext
	C.glEnable(C.GL_TEXTURE_2D)
	C.glTexParameteri(C.GL_TEXTURE_2D, C.GL_TEXTURE_MIN_FILTER, C.GL_NEAREST)
	C.glTexParameteri(C.GL_TEXTURE_2D, C.GL_TEXTURE_MAG_FILTER, C.GL_NEAREST)
	g.SetOffscreen(device.offscreenTexture)
	g.Clear()
	device.drawFunc(g, device.offscreenTexture)
	g.flush()
	
	C.glTexParameteri(C.GL_TEXTURE_2D, C.GL_TEXTURE_MIN_FILTER, C.GL_LINEAR)
	C.glTexParameteri(C.GL_TEXTURE_2D, C.GL_TEXTURE_MAG_FILTER, C.GL_LINEAR)
	g.resetOffscreen()
	g.Clear()
	geometryMatrix := IdentityGeometryMatrix()
	geometryMatrix.SetA(AffineMatrixElement(g.screenScale))
	geometryMatrix.SetD(AffineMatrixElement(g.screenScale))
	g.DrawTexture(device.offscreenTexture,
		0, 0, device.screenWidth, device.screenHeight,
		geometryMatrix, IdentityColorMatrix())
	g.flush()
}
