// Copyright 2015 Hajime Hoshi
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ui

import (
	"fmt"
	glfw "github.com/go-gl/glfw3"
	"github.com/hajimehoshi/ebiten/internal/opengl"
	"runtime"
)

var current *ui

func Use(f func(*opengl.Context)) {
	ch := make(chan struct{})
	current.funcs <- func() {
		defer close(ch)
		f(current.glContext)
	}
	<-ch
}

func ActualScale() int {
	return current.actualScale
}

func DoEvents() {
	current.doEvents()
}

func Terminate() {
	current.terminate()
}

func IsClosed() bool {
	return current.isClosed()
}

func SwapBuffers() {
	current.swapBuffers()
}

func init() {
	runtime.LockOSThread()

	glfw.SetErrorCallback(func(err glfw.ErrorCode, desc string) {
		panic(fmt.Sprintf("%v: %v\n", err, desc))
	})
	if !glfw.Init() {
		panic("glfw.Init() fails")
	}
	glfw.WindowHint(glfw.Visible, glfw.False)
	glfw.WindowHint(glfw.Resizable, glfw.False)

	window, err := glfw.CreateWindow(16, 16, "", nil, nil)
	if err != nil {
		panic(err)
	}

	u := &ui{
		window: window,
		funcs:  make(chan func()),
	}
	go func() {
		runtime.LockOSThread()
		u.window.MakeContextCurrent()
		u.glContext = opengl.NewContext()
		glfw.SwapInterval(1)
		for f := range u.funcs {
			f()
		}
	}()
	current = u
}

type ui struct {
	window      *glfw.Window
	scale       int
	actualScale int
	glContext   *opengl.Context
	input       input
	funcs       chan func()
}

func Start(width, height, scale int, title string) error {
	monitor, err := glfw.GetPrimaryMonitor()
	if err != nil {
		return err
	}
	videoMode, err := monitor.GetVideoMode()
	if err != nil {
		return err
	}
	x := (videoMode.Width - width*scale) / 2
	y := (videoMode.Height - height*scale) / 3

	ch := make(chan struct{})
	ui := current
	window := ui.window
	window.SetFramebufferSizeCallback(func(w *glfw.Window, width, height int) {
		close(ch)
	})
	window.SetSize(width*scale, height*scale)
	window.SetTitle(title)
	window.SetPosition(x, y)
	window.Show()

	for {
		done := false
		glfw.PollEvents()
		select {
		case <-ch:
			done = true
		default:
		}
		if done {
			break
		}
	}

	ui.scale = scale

	// For retina displays, recalculate the scale with the framebuffer size.
	windowWidth, _ := window.GetFramebufferSize()
	ui.actualScale = windowWidth / width

	return err
}

func (u *ui) doEvents() {
	glfw.PollEvents()
	u.input.update(u.window, u.scale)
}

func (u *ui) terminate() {
	glfw.Terminate()
}

func (u *ui) isClosed() bool {
	return u.window.ShouldClose()
}

func (u *ui) swapBuffers() {
	u.window.SwapBuffers()
}
