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

// +build darwin,!arm,!arm64 linux windows
// +build !js
// +build !android
// +build !ios

package ui

import (
	"errors"
	"runtime"
	"time"

	"github.com/go-gl/glfw/v3.1/glfw"
	"github.com/hajimehoshi/ebiten/internal/graphics/opengl"
)

type userInterface struct {
	window      *glfw.Window
	width       int
	height      int
	scale       float64
	funcs       chan func()
	sizeChanged bool
}

var currentUI *userInterface

func CurrentUI() UserInterface {
	return currentUI
}

func init() {
	if err := initialize(); err != nil {
		panic(err)
	}
}

func initialize() error {
	runtime.LockOSThread()

	if err := glfw.Init(); err != nil {
		return err
	}
	glfw.WindowHint(glfw.Visible, glfw.False)
	glfw.WindowHint(glfw.Resizable, glfw.False)
	glfw.WindowHint(glfw.ContextVersionMajor, 2)
	glfw.WindowHint(glfw.ContextVersionMinor, 1)

	// As start, create an window with temporary size to create OpenGL context thread.
	window, err := glfw.CreateWindow(16, 16, "", nil, nil)
	if err != nil {
		return err
	}
	u := &userInterface{
		window:      window,
		funcs:       make(chan func()),
		sizeChanged: true,
	}
	u.window.MakeContextCurrent()
	glfw.SwapInterval(1)
	currentUI = u
	return nil
}

func Main() error {
	return currentUI.main()
}

func (u *userInterface) main() error {
	// TODO: Check this is done on the main thread.
	for f := range u.funcs {
		f()
	}
	return nil
}

func (u *userInterface) runOnMainThread(f func() error) error {
	if u.funcs == nil {
		// already closed
		return nil
	}
	ch := make(chan struct{})
	var err error
	u.funcs <- func() {
		err = f()
		close(ch)
	}
	<-ch
	return err
}

func (u *userInterface) SetScreenSize(width, height int) bool {
	r := false
	u.runOnMainThread(func() error {
		r = u.setScreenSize(width, height, u.scale)
		return nil
	})
	return r
}

func (u *userInterface) SetScreenScale(scale float64) bool {
	r := false
	u.runOnMainThread(func() error {
		r = u.setScreenSize(u.width, u.height, scale)
		return nil
	})
	return r
}

func (u *userInterface) ScreenScale() float64 {
	s := 0.0
	u.runOnMainThread(func() error {
		s = u.scale
		return nil
	})
	return s
}

func (u *userInterface) Start(width, height int, scale float64, title string) error {
	// GLContext must be created before setting the screen size, which requires
	// swapping buffers.
	var err error
	glContext, err = opengl.NewContext(currentUI.runOnMainThread)
	if err != nil {
		return err
	}
	if err := u.runOnMainThread(func() error {
		m := glfw.GetPrimaryMonitor()
		v := m.GetVideoMode()
		if !u.setScreenSize(width, height, scale) {
			return errors.New("ui: Fail to set the screen size")
		}
		u.window.SetTitle(title)
		u.window.Show()

		w, h := u.glfwSize()
		x := (v.Width - w) / 2
		y := (v.Height - h) / 3
		u.window.SetPos(x, y)
		return nil
	}); err != nil {
		return err
	}
	return nil
}

func (u *userInterface) glfwSize() (int, int) {
	return int(float64(u.width) * u.scale * glfwScale()), int(float64(u.height) * u.scale * glfwScale())
}

func (u *userInterface) actualScreenScale() float64 {
	return u.scale * deviceScale()
}

func (u *userInterface) pollEvents() error {
	glfw.PollEvents()
	return currentInput.update(u.window, u.scale*glfwScale())
}

func (u *userInterface) Update() (interface{}, error) {
	shouldClose := false
	u.runOnMainThread(func() error {
		shouldClose = u.window.ShouldClose()
		return nil
	})
	if shouldClose {
		return CloseEvent{}, nil
	}

	var screenSizeEvent *ScreenSizeEvent
	u.runOnMainThread(func() error {
		if !u.sizeChanged {
			return nil
		}
		u.sizeChanged = false
		screenSizeEvent = &ScreenSizeEvent{
			Width:       u.width,
			Height:      u.height,
			ActualScale: u.actualScreenScale(),
		}
		return nil
	})
	if screenSizeEvent != nil {
		return *screenSizeEvent, nil
	}

	if err := u.runOnMainThread(func() error {
		if err := u.pollEvents(); err != nil {
			return err
		}
		for u.window.GetAttrib(glfw.Focused) == 0 {
			// Wait for an arbitrary period to avoid busy loop.
			time.Sleep(time.Second / 60)
			if err := u.pollEvents(); err != nil {
				return err
			}
			if u.window.ShouldClose() {
				return nil
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}
	// Dummy channel
	ch := make(chan struct{}, 1)
	return RenderEvent{ch}, nil
}

func (u *userInterface) Terminate() error {
	u.runOnMainThread(func() error {
		glfw.Terminate()
		return nil
	})
	close(u.funcs)
	u.funcs = nil
	return nil
}

func (u *userInterface) SwapBuffers() error {
	// The bound framebuffer must be the default one (0) before swapping buffers.
	if err := glContext.BindScreenFramebuffer(); err != nil {
		return err
	}
	if err := u.runOnMainThread(func() error {
		return u.swapBuffers()
	}); err != nil {
		return err
	}
	return nil
}

func (u *userInterface) swapBuffers() error {
	u.window.SwapBuffers()
	return nil
}

func (u *userInterface) FinishRendering() error {
	return nil
}

func (u *userInterface) setScreenSize(width, height int, scale float64) bool {
	if u.width == width && u.height == height && u.scale == scale {
		return false
	}

	origScale := u.scale
	u.scale = scale

	// On Windows, giving a too small width doesn't call a callback (#165).
	// To prevent hanging up, return asap if the width is too small.
	// 252 is an arbitrary number and I guess this is small enough.
	const minWindowWidth = 252
	if int(float64(width)*u.actualScreenScale()) < minWindowWidth {
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
	w, h := u.glfwSize()
	window.SetSize(w, h)

event:
	for {
		glfw.PollEvents()
		select {
		case <-ch:
			break event
		default:
		}
	}
	u.sizeChanged = true
	return true
}
