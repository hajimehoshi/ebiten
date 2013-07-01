// Copyright 2013 Hajime Hoshi
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

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
	context          *Context
	offscreenTexture graphics.Texture
	drawing          chan chan func(graphics.Context, graphics.Texture)
	updating         chan chan func()
}

func NewDevice(screenWidth, screenHeight, screenScale int, updating chan chan func()) *Device {
	context := newContext(screenWidth, screenHeight, screenScale)

	device := &Device{
		screenWidth:  screenWidth,
		screenHeight: screenHeight,
		screenScale:  screenScale,
		drawing:      make(chan chan func(graphics.Context, graphics.Texture)),
		context:      context,
		updating:     updating,
	}
	device.offscreenTexture =
		device.context.NewTexture(screenWidth, screenHeight)

	go func() {
		for {
			ch := <-device.updating
			ch <- device.Update
		}
	}()

	return device
}

func (device *Device) Drawing() <-chan chan func(graphics.Context, graphics.Texture) {
	return device.drawing
}

func (device *Device) OffscreenTexture() graphics.Texture {
	return device.offscreenTexture
}

func (device *Device) Update() {
	g := device.context
	C.glEnable(C.GL_TEXTURE_2D)
	C.glTexParameteri(C.GL_TEXTURE_2D, C.GL_TEXTURE_MIN_FILTER, C.GL_NEAREST)
	C.glTexParameteri(C.GL_TEXTURE_2D, C.GL_TEXTURE_MAG_FILTER, C.GL_NEAREST)
	g.SetOffscreen(device.offscreenTexture.ID)
	g.Clear()

	ch := make(chan func(graphics.Context, graphics.Texture))
	device.drawing <- ch
	drawable := <-ch
	drawable(g, device.offscreenTexture)

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
	return device.context
}
