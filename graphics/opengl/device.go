package opengl

// #cgo LDFLAGS: -framework OpenGL
//
// #include <OpenGL/gl.h>
// #include <stdlib.h>
import "C"
import (
	"github.com/hajimehoshi/go.ebiten/graphics"
	"github.com/hajimehoshi/go.ebiten/graphics/matrix"
)

type Device struct {
	screenWidth      int
	screenHeight     int
	screenScale      int
	graphicsContext  *GraphicsContext
	offscreenTexture graphics.Texture
	deviceUpdate     chan chan graphics.Drawable
	updating         chan chan func()
}

func NewDevice(screenWidth, screenHeight, screenScale int, updating chan chan func()) *Device {
	graphicsContext := newGraphicsContext(screenWidth, screenHeight, screenScale)

	device := &Device{
		screenWidth:     screenWidth,
		screenHeight:    screenHeight,
		screenScale:     screenScale,
		deviceUpdate:    make(chan chan graphics.Drawable),
		graphicsContext: graphicsContext,
		updating:        updating,
	}
	device.offscreenTexture =
		device.graphicsContext.NewTexture(screenWidth, screenHeight)

	go func() {
		for {
			ch := <-device.updating
			ch <- device.Update
		}
	}()

	return device
}

func (device *Device) Drawing() <-chan chan graphics.Drawable {
	return device.deviceUpdate
}

func (device *Device) OffscreenTexture() graphics.Texture {
	return device.offscreenTexture
}

func (device *Device) Update() {
	g := device.graphicsContext
	C.glEnable(C.GL_TEXTURE_2D)
	C.glTexParameteri(C.GL_TEXTURE_2D, C.GL_TEXTURE_MIN_FILTER, C.GL_NEAREST)
	C.glTexParameteri(C.GL_TEXTURE_2D, C.GL_TEXTURE_MAG_FILTER, C.GL_NEAREST)
	g.SetOffscreen(device.offscreenTexture.ID)
	g.Clear()

	ch := make(chan graphics.Drawable)
	device.deviceUpdate <- ch
	drawable := <-ch
	drawable.Draw(g, device.offscreenTexture)

	g.flush()

	C.glTexParameteri(C.GL_TEXTURE_2D, C.GL_TEXTURE_MIN_FILTER, C.GL_LINEAR)
	C.glTexParameteri(C.GL_TEXTURE_2D, C.GL_TEXTURE_MAG_FILTER, C.GL_LINEAR)
	g.resetOffscreen()
	g.Clear()

	scale := float64(g.screenScale)
	geometryMatrix := matrix.Geometry{
		[2][3]float64{
			{scale, 0, 0},
			{0, scale, 0},
		},
	}
	g.DrawTexture(device.offscreenTexture.ID,
		geometryMatrix, matrix.IdentityColor())
	g.flush()
}

func (device *Device) TextureFactory() graphics.TextureFactory {
	return device.graphicsContext
}
