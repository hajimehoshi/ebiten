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
	"errors"
	"fmt"
	glfw "github.com/go-gl/glfw3"
	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/internal/opengl"
)

func init() {
	glfw.SetErrorCallback(func(err glfw.ErrorCode, desc string) {
		panic(fmt.Sprintf("%v: %v\n", err, desc))
	})
}

type UI struct {
	canvas *canvas
}

func (u *UI) Start(width, height, scale int, title string) error {
	if !glfw.Init() {
		return errors.New("glfw.Init() fails")
	}
	glfw.WindowHint(glfw.Resizable, glfw.False)
	window, err := glfw.CreateWindow(width*scale, height*scale, title, nil, nil)
	if err != nil {
		return err
	}

	c := &canvas{
		window:    window,
		funcs:     make(chan func()),
		funcsDone: make(chan struct{}),
	}
	ebiten.SetInput(&c.input)
	ebiten.SetTextureFactory(c)

	c.run(width, height, scale)

	// For retina displays, recalculate the scale with the framebuffer size.
	windowWidth, _ := window.GetFramebufferSize()
	realScale := windowWidth / width
	c.use(func() {
		c.context, err = opengl.Initialize(width, height, realScale)
	})
	if err != nil {
		return err
	}

	u.canvas = c
	return nil
}

func (u *UI) DoEvents() {
	glfw.PollEvents()
	u.canvas.update()
}

func (u *UI) Terminate() {
	glfw.Terminate()
}

func (u *UI) Draw(drawer ebiten.GraphicsContextDrawer) error {
	return u.canvas.draw(drawer)
}

func (u *UI) IsClosed() bool {
	return u.canvas.isClosed()
}
