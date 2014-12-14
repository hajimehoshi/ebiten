/*
Copyright 2014 Hajime Hoshi

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package ebiten

import (
	"github.com/go-gl/gl"
	glfw "github.com/go-gl/glfw3"
	"image"
	"runtime"
)

type canvas struct {
	window          *glfw.Window
	scale           int
	graphicsContext *graphicsContext
	input           Input
	funcs           chan func()
	funcsDone       chan struct{}
}

func (c *canvas) draw(game Game) (err error) {
	c.use(func() {
		c.graphicsContext.PreUpdate()
	})
	if err = game.Draw(&syncGraphicsContext{c}); err != nil {
		return
	}
	c.use(func() {
		c.graphicsContext.PostUpdate()
		c.window.SwapBuffers()
	})
	return
}

func (c *canvas) isClosed() bool {
	return c.window.ShouldClose()
}

func (c *canvas) NewTextureID(img image.Image, filter Filter) (TextureID, error) {
	var id TextureID
	var err error
	c.use(func() {
		glFilter := 0
		switch filter {
		case FilterNearest:
			glFilter = gl.NEAREST
		case FilterLinear:
			glFilter = gl.LINEAR
		default:
			panic("not reached")
		}
		id, err = newTextureID(img, glFilter)
	})
	return id, err
}

func (c *canvas) NewRenderTargetID(width, height int, filter Filter) (RenderTargetID, error) {
	var id RenderTargetID
	var err error
	c.use(func() {
		glFilter := 0
		switch filter {
		case FilterNearest:
			glFilter = gl.NEAREST
		case FilterLinear:
			glFilter = gl.LINEAR
		default:
			panic("not reached")
		}
		id, err = newRenderTargetID(width, height, glFilter)
	})
	return id, err
}

func (c *canvas) run(width, height, scale int) {
	go func() {
		runtime.LockOSThread()
		c.window.MakeContextCurrent()
		glfw.SwapInterval(1)
		for {
			(<-c.funcs)()
			c.funcsDone <- struct{}{}
		}
	}()
}

func (c *canvas) use(f func()) {
	c.funcs <- f
	<-c.funcsDone
}

func (c *canvas) update() {
	c.input.Update(c.window, c.scale)
}

func (c *canvas) IsKeyPressed(key Key) bool {
	return c.input.IsKeyPressed(key)
}

func (c *canvas) IsMouseButtonPressed(button MouseButton) bool {
	return c.input.IsMouseButtonPressed(button)
}

func (c *canvas) CursorPosition() (x, y int) {
	return c.input.CursorPosition()
}
