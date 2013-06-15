package graphics

// #cgo LDFLAGS: -framework OpenGL
//
// #include <OpenGL/gl.h>
// #include <stdlib.h>
import "C"

type device struct {
	screenWidth int
	screenHeight int
	screenScale int
	graphicsContext *GraphicsContext
	offscreenTexture *Texture
	drawFunc func(*GraphicsContext, *Texture)
}

// This method should be called on the UI thread??
func newDevice(screenWidth, screenHeight, screenScale int, drawFunc func(*GraphicsContext, *Texture)) *device {
	return &device{
		screenWidth: screenWidth,
		screenHeight: screenHeight,
		screenScale: screenScale,
		graphicsContext: newGraphicsContext(screenWidth, screenHeight, screenScale),
		offscreenTexture: NewTexture(screenWidth, screenHeight),
		drawFunc: drawFunc,
	}
}

// This method should be called on the UI thread??
func (d *device) Update() {
	g := d.graphicsContext
	// g.initialize()
	C.glEnable(C.GL_TEXTURE_2D)
	C.glTexParameteri(C.GL_TEXTURE_2D, C.GL_TEXTURE_MIN_FILTER, C.GL_NEAREST)
	C.glTexParameteri(C.GL_TEXTURE_2D, C.GL_TEXTURE_MAG_FILTER, C.GL_NEAREST)
	g.SetOffscreen(d.offscreenTexture)
	g.Clear()
	d.drawFunc(g, d.offscreenTexture)
	g.flush()
	C.glTexParameteri(C.GL_TEXTURE_2D, C.GL_TEXTURE_MIN_FILTER, C.GL_LINEAR)
	C.glTexParameteri(C.GL_TEXTURE_2D, C.GL_TEXTURE_MAG_FILTER, C.GL_LINEAR)
	g.resetOffscreen()
	g.Clear()

	geometryMatrix := IdentityGeometryMatrix()
	geometryMatrix.SetA(AffineMatrixElement(g.screenScale))
	geometryMatrix.SetD(AffineMatrixElement(g.screenScale))
	g.DrawTexture(d.offscreenTexture,
		0, 0, d.screenWidth, d.screenHeight,
		geometryMatrix, IdentityColorMatrix())
	g.flush()
}
