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

package glfw

import (
	glfw "github.com/go-gl/glfw3"
	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/internal/opengl"
	"image"
	"runtime"
)

type canvas struct {
	window          *glfw.Window
	graphicsContext *opengl.GraphicsContext
	input           input
	funcs           chan func()
	funcsDone       chan struct{}
}

func (c *canvas) draw(game ebiten.Game) (err error) {
	c.use(func() {
		c.graphicsContext.PreUpdate()
	})
	if err = game.Draw(&graphicsContext{c}); err != nil {
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

func (c *canvas) NewTextureID(img image.Image, filter ebiten.Filter) (ebiten.TextureID, error) {
	var id ebiten.TextureID
	var err error
	c.use(func() {
		id, err = opengl.NewTextureID(img, filter)
	})
	return id, err
}

func (c *canvas) NewRenderTargetID(width, height int, filter ebiten.Filter) (ebiten.RenderTargetID, error) {
	var id ebiten.RenderTargetID
	var err error
	c.use(func() {
		id, err = opengl.NewRenderTargetID(width, height, filter)
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
	c.input.update(c.window)
}

func (c *canvas) IsKeyPressed(key ebiten.Key) bool {
	return c.input.IsKeyPressed(key)
}

func (c *canvas) IsMouseButtonPressed(button ebiten.MouseButton) bool {
	return c.input.IsMouseButtonPressed(button)
}

func (c *canvas) CursorPosition() (x, y int) {
	return c.input.CursorPosition()
}
