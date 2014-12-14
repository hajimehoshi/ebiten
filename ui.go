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

func (u *ui) start(game Game, width, height, scale int, title string) error {
	if !glfw.Init() {
		return errors.New("glfw.Init() fails")
	}
	glfw.WindowHint(glfw.Resizable, glfw.False)
	window, err := glfw.CreateWindow(width*scale, height*scale, title, nil, nil)
	if err != nil {
		return err
	}

	c, err := newCanvas(window, width, height, scale)
	if err != nil {
		return err
	}
	u.canvas = c

	return nil
}

func (u *ui) doEvents() {
	glfw.PollEvents()
	u.canvas.update()
}

func (u *ui) terminate() {
	glfw.Terminate()
}

func (u *ui) isClosed() bool {
	return u.canvas.isClosed()
}

func (u *ui) drawGame(game Game) error {
	return u.canvas.draw(game)
}
