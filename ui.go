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
	"errors"
	"fmt"
	glfw "github.com/go-gl/glfw3"
)

func init() {
	glfw.SetErrorCallback(func(err glfw.ErrorCode, desc string) {
		panic(fmt.Sprintf("%v: %v\n", err, desc))
	})
}

type ui struct {
	canvas *canvas
}

func (u *ui) Start(game Game, width, height, scale int, title string) error {
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
		scale:     scale,
		funcs:     make(chan func()),
		funcsDone: make(chan struct{}),
	}

	c.run(width, height, scale)

	// For retina displays, recalculate the scale with the framebuffer size.
	windowWidth, _ := window.GetFramebufferSize()
	realScale := windowWidth / width
	c.use(func() {
		c.graphicsContext, err = initialize(width, height, realScale)
	})
	if err != nil {
		return err
	}

	u.canvas = c

	return nil
}

func (u *ui) DoEvents() {
	glfw.PollEvents()
	u.canvas.update()
}

func (u *ui) Terminate() {
	glfw.Terminate()
}

func (u *ui) IsClosed() bool {
	return u.canvas.isClosed()
}

func (u *ui) DrawGame(game Game) error {
	return u.canvas.draw(game)
}
