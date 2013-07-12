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
	"runtime"
)

type Device struct {
	screenScale int
	context     *Context
	drawing     chan chan func(graphics.Context)
	updating    chan chan func()
}

func NewDevice(screenWidth, screenHeight, screenScale int, updating chan chan func()) *Device {
	context := newContext(screenWidth, screenHeight, screenScale)

	device := &Device{
		screenScale: screenScale,
		drawing:     make(chan chan func(graphics.Context)),
		context:     context,
		updating:    updating,
	}

	go func() {
		for {
			ch := <-device.updating
			ch <- device.Update
		}
	}()

	return device
}

func (device *Device) Drawing() <-chan chan func(graphics.Context) {
	return device.drawing
}

func (device *Device) Update() {
	context := device.context
	C.glEnable(C.GL_TEXTURE_2D)
	C.glTexParameteri(C.GL_TEXTURE_2D, C.GL_TEXTURE_MIN_FILTER, C.GL_NEAREST)
	C.glTexParameteri(C.GL_TEXTURE_2D, C.GL_TEXTURE_MAG_FILTER, C.GL_NEAREST)
	context.SetOffscreen(context.Screen().ID())
	context.Clear()

	ch := make(chan func(graphics.Context))
	device.drawing <- ch
	drawable := <-ch
	drawable(context)

	context.flush()

	C.glTexParameteri(C.GL_TEXTURE_2D, C.GL_TEXTURE_MIN_FILTER, C.GL_LINEAR)
	C.glTexParameteri(C.GL_TEXTURE_2D, C.GL_TEXTURE_MAG_FILTER, C.GL_LINEAR)
	context.resetOffscreen()
	context.Clear()

	scale := float64(context.screenScale)
	geometryMatrix := matrix.Geometry{
		[2][3]float64{
			{scale, 0, 0},
			{0, scale, 0},
		},
	}
	context.DrawTexture(context.Screen().ID(),
		geometryMatrix, matrix.IdentityColor())
	context.flush()
}

func (device *Device) TextureFactory() graphics.TextureFactory {
	return device.context
}

func init() {
	runtime.LockOSThread()
}
