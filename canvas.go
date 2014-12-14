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
	glfw "github.com/go-gl/glfw3"
	"image"
	"runtime"
)

type canvas struct {
	window          *glfw.Window
	scale           int
	graphicsContext *graphicsContext
	input           input
	funcs           chan func()
	funcsDone       chan struct{}
}

func newCanvas(window *glfw.Window, width, height, scale int) (*canvas, error) {
	c := &canvas{
		window:    window,
		scale:     scale,
		funcs:     make(chan func()),
		funcsDone: make(chan struct{}),
	}

	c.run(width, height, scale)

	// For retina displays, recalculate the scale with the framebuffer size.
	windowWidth, _ := window.GetFramebufferSize()
	realScale := windowWidth / width
	var err error
	c.use(func() {
		c.graphicsContext, err = newGraphicsContext(width, height, realScale)
	})
	if err != nil {
		return nil, err
	}
	return c, nil
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

func (c *canvas) newTextureID(img image.Image, filter int) (TextureID, error) {
	var id TextureID
	var err error
	c.use(func() {
		id, err = newTextureID(img, filter)
	})
	return id, err
}

func (c *canvas) newRenderTargetID(width, height int, filter int) (RenderTargetID, error) {
	var id RenderTargetID
	var err error
	c.use(func() {
		id, err = newRenderTargetID(width, height, filter)
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
	c.input.update(c.window, c.scale)
}
