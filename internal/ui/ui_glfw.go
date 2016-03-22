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
	"errors"
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

func Start(width, height, scale int, title string) error {
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

func SetScreenSize(width, height int) bool {
	return currentUI.setScreenSize(width, height, currentUI.scale)
}

func SetScreenScale(scale int) bool {
	return currentUI.setScreenSize(currentUI.width, currentUI.height, scale)
}

func ActualScale() int {
	return currentUI.actualScale()
}

type userInterface struct {
	window           *glfw.Window
	width            int
	height           int
	scale            int
	deviceScale      float64
	framebufferScale int
	context          *opengl.Context
}

func (u *userInterface) start(width, height, scale int, title string) error {
	m := glfw.GetPrimaryMonitor()
	v := m.GetVideoMode()
	mw, _ := m.GetPhysicalSize()
	u.deviceScale = 1
	u.framebufferScale = 1
	// mw can be 0 on some environment like Linux VM
	if 0 < mw {
		dpi := float64(v.Width) * 25.4 / float64(mw)
		u.deviceScale = dpi / 96
		if u.deviceScale < 1 {
			u.deviceScale = 1
		}
	}

	if !u.setScreenSize(width, height, scale) {
		return errors.New("ui: Fail to set the screen size")
	}
	u.window.SetTitle(title)
	u.window.Show()

	x := (v.Width - width*u.windowScale()) / 2
	y := (v.Height - height*u.windowScale()) / 3
	u.window.SetPos(x, y)

	return nil
}

func (u *userInterface) windowScale() int {
	return u.scale * int(u.deviceScale)
}

func (u *userInterface) actualScale() int {
	return u.windowScale() * u.framebufferScale
}

func (u *userInterface) pollEvents() error {
	glfw.PollEvents()
	return updateInput(u.window, u.windowScale())
}

func (u *userInterface) doEvents() error {
	if err := u.pollEvents(); err != nil {
		return err
	}
	for u.window.GetAttrib(glfw.Focused) == 0 {
		// Wait for an arbitrary period to avoid busy loop.
		time.Sleep(time.Second / 60)
		if err := u.pollEvents(); err != nil {
			return err
		}
		if u.isClosed() {
			return nil
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
	// The bound framebuffer must be the default one (0) before swapping buffers.
	u.context.BindZeroFramebuffer()
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

	// u.scale should be set first since this affects windowScale().
	origScale := u.scale
	u.scale = scale

	// On Windows, giving a too small width doesn't call a callback (#165).
	// To prevent hanging up, return asap if the width is too small.
	// 252 is an arbitrary number and I guess this is small enough.
	const minWindowWidth = 252
	if width*u.actualScale() < minWindowWidth {
		u.scale = origScale
		return false
	}
	u.width = width
	u.height = height

	// To make sure the current existing framebuffers are rendered,
	// swap buffers here before SetSize is called.
	u.swapBuffers()

	ch := make(chan struct{})
	window := u.window
	window.SetFramebufferSizeCallback(func(_ *glfw.Window, width, height int) {
		window.SetFramebufferSizeCallback(nil)
		close(ch)
	})
	window.SetSize(width*u.windowScale(), height*u.windowScale())

event:
	for {
		glfw.PollEvents()
		select {
		case <-ch:
			break event
		default:
		}
	}
	// This is usually 1, but sometimes more than 1 (e.g. Retina Mac)
	fw, _ := window.GetFramebufferSize()
	u.framebufferScale = fw / width / u.windowScale()
	return true
}
