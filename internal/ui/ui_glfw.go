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

// +build !js

package ui

import (
	"fmt"
	"runtime"
	"time"

	"github.com/go-gl/glfw/v3.1/glfw"
	"github.com/hajimehoshi/ebiten/internal/graphics/opengl"
)

func Now() int64 {
	return time.Now().UnixNano()
}

var currentUI *userInterface

func Init() *opengl.Context {
	runtime.LockOSThread()

	err := glfw.Init()
	if err != nil {
		panic(fmt.Sprintf("glfw.Init() fails: %v", err))
	}
	glfw.WindowHint(glfw.Visible, glfw.False)
	glfw.WindowHint(glfw.Resizable, glfw.False)
	glfw.WindowHint(glfw.ContextVersionMajor, 2)
	glfw.WindowHint(glfw.ContextVersionMinor, 1)

	// As start, create an window with temporary size to create OpenGL context thread.
	window, err := glfw.CreateWindow(16, 16, "", nil, nil)
	if err != nil {
		panic(err)
	}

	u := &userInterface{
		window: window,
	}
	ch := make(chan struct{})
	go func() {
		runtime.LockOSThread()
		u.window.MakeContextCurrent()
		glfw.SwapInterval(1)
		u.context = opengl.NewContext()
		close(ch)
		u.context.Loop()
	}()
	currentUI = u
	<-ch
	u.context.Init()

	return u.context
}

func Start(width, height, scale int, title string) (actualScale int, err error) {
	return currentUI.start(width, height, scale, title)
}

func Terminate() {
	currentUI.terminate()
}

func DoEvents() error {
	return currentUI.doEvents()
}

func IsClosed() bool {
	return currentUI.isClosed()
}

func SwapBuffers() {
	currentUI.swapBuffers()
}

func SetScreenSize(width, height int) (bool, int) {
	result := currentUI.setScreenSize(width, height, currentUI.scale)
	return result, currentUI.actualScale()
}

func SetScreenScale(scale int) (bool, int) {
	result := currentUI.setScreenSize(currentUI.width, currentUI.height, scale)
	return result, currentUI.actualScale()
}

type userInterface struct {
	window            *glfw.Window
	width             int
	height            int
	scale             int
	deviceScaleFactor float64
	context           *opengl.Context
}

func (u *userInterface) start(width, height, scale int, title string) (actualScale int, err error) {
	m := glfw.GetPrimaryMonitor()
	mw, _ := m.GetPhysicalSize()
	v := m.GetVideoMode()
	dpi := float64(v.Width) * 25.4 / float64(mw)
	u.deviceScaleFactor = dpi / 96

	u.setScreenSize(width, height, scale)
	u.window.SetTitle(title)
	u.window.Show()

	x := (v.Width - width*u.actualScale()) / 2
	y := (v.Height - height*u.actualScale()) / 3
	u.window.SetPos(x, y)

	return u.actualScale(), nil
}

func (u *userInterface) actualScale() int {
	return int(float64(u.scale) * u.deviceScaleFactor)
}

func (u *userInterface) pollEvents() error {
	glfw.PollEvents()
	return updateInput(u.window, u.scale)
}

func (u *userInterface) doEvents() error {
	if err := u.pollEvents(); err != nil {
		return err
	}
	for u.window.GetAttrib(glfw.Focused) == 0 {
		time.Sleep(time.Second / 60)
		if err := u.pollEvents(); err != nil {
			return err
		}
	}
	return nil
}

func (u *userInterface) terminate() {
	glfw.Terminate()
}

func (u *userInterface) isClosed() bool {
	return u.window.ShouldClose()
}

func (u *userInterface) swapBuffers() {
	// Call glFinish before glfwSwapBuffer to make sure
	// all OpenGL tasks are executed.
	u.context.Finish()
	u.context.RunOnContextThread(func() {
		u.window.SwapBuffers()
	})
}

func (u *userInterface) setScreenSize(width, height, scale int) bool {
	if u.width == width && u.height == height && u.scale == scale {
		return false
	}
	u.scale = scale

	// To make sure the current existing framebuffers are rendered,
	// swap buffers here before SetSize is called.
	u.context.BindZeroFramebuffer()
	u.swapBuffers()

	ch := make(chan struct{})
	window := u.window
	window.SetFramebufferSizeCallback(func(w *glfw.Window, width, height int) {
		window.SetFramebufferSizeCallback(nil)
		close(ch)
	})
	window.SetSize(width*u.actualScale(), height*u.actualScale())

event:
	for {
		glfw.PollEvents()
		select {
		case <-ch:
			break event
		default:
		}
	}
	u.width = width
	u.height = height
	return true
}
