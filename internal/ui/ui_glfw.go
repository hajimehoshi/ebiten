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
	"runtime"
	"time"

	"github.com/go-gl/glfw/v3.1/glfw"
	"github.com/hajimehoshi/ebiten/internal/graphics"
	"github.com/hajimehoshi/ebiten/internal/graphics/opengl"
)

type UserInterface struct {
	window           *glfw.Window
	width            int
	height           int
	scale            int
	deviceScale      float64
	framebufferScale int
	context          *opengl.Context
	funcs            chan func()
	sizeChanged      bool
}

var currentUI *UserInterface

func CurrentUI() *UserInterface {
	return currentUI
}

func initialize() (*opengl.Context, error) {
	runtime.LockOSThread()

	if err := glfw.Init(); err != nil {
		return nil, err
	}
	glfw.WindowHint(glfw.Visible, glfw.False)
	glfw.WindowHint(glfw.Resizable, glfw.False)
	glfw.WindowHint(glfw.ContextVersionMajor, 2)
	glfw.WindowHint(glfw.ContextVersionMinor, 1)

	// As start, create an window with temporary size to create OpenGL context thread.
	window, err := glfw.CreateWindow(16, 16, "", nil, nil)
	if err != nil {
		return nil, err
	}

	u := &UserInterface{
		window:      window,
		funcs:       make(chan func()),
		sizeChanged: true,
	}
	ch := make(chan error)
	go func() {
		runtime.LockOSThread()
		u.window.MakeContextCurrent()
		glfw.SwapInterval(1)
		var err error
		u.context, err = opengl.NewContext()
		if err != nil {
			ch <- err
		}
		close(ch)
		u.context.Loop()
	}()
	currentUI = u
	if err := <-ch; err != nil {
		return nil, err
	}
	if err := u.context.Init(); err != nil {
		return nil, err
	}
	if err := graphics.Initialize(u.context); err != nil {
		return nil, err
	}

	return u.context, nil
}

func Main() error {
	return currentUI.main()
}

func (u *UserInterface) main() error {
	// TODO: Check this is done on the main thread.
	for f := range u.funcs {
		f()
	}
	return nil
}

func (u *UserInterface) runOnMainThread(f func()) {
	if u.funcs == nil {
		// already closed
		return
	}
	ch := make(chan struct{})
	u.funcs <- func() {
		f()
		close(ch)
	}
	<-ch
}

func (u *UserInterface) SetScreenSize(width, height int) bool {
	r := false
	u.runOnMainThread(func() {
		r = u.setScreenSize(width, height, u.scale)
	})
	return r
}

func (u *UserInterface) SetScreenScale(scale int) bool {
	r := false
	u.runOnMainThread(func() {
		r = u.setScreenSize(u.width, u.height, scale)
	})
	return r
}

func (u *UserInterface) ScreenScale() int {
	s := 0
	u.runOnMainThread(func() {
		s = u.scale
	})
	return s
}

func (u *UserInterface) Start(width, height, scale int, title string) error {
	var err error
	u.runOnMainThread(func() {
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
			err = errors.New("ui: Fail to set the screen size")
			return
		}
		u.window.SetTitle(title)
		u.window.Show()

		x := (v.Width - width*u.windowScale()) / 2
		y := (v.Height - height*u.windowScale()) / 3
		u.window.SetPos(x, y)
	})
	return err
}

func (u *UserInterface) windowScale() int {
	return u.scale * int(u.deviceScale)
}

func (u *UserInterface) actualScreenScale() int {
	return u.windowScale() * u.framebufferScale
}

func (u *UserInterface) pollEvents() error {
	glfw.PollEvents()
	return currentInput.update(u.window, u.windowScale())
}

func (u *UserInterface) Update() (interface{}, error) {
	shouldClose := false
	u.runOnMainThread(func() {
		shouldClose = u.window.ShouldClose()
	})
	if shouldClose {
		return CloseEvent{}, nil
	}

	var screenSizeEvent *ScreenSizeEvent
	u.runOnMainThread(func() {
		if !u.sizeChanged {
			return
		}
		u.sizeChanged = false
		screenSizeEvent = &ScreenSizeEvent{
			Width:       u.width,
			Height:      u.height,
			Scale:       u.scale,
			ActualScale: u.actualScreenScale(),
		}
	})
	if screenSizeEvent != nil {
		return *screenSizeEvent, nil
	}

	var ferr error
	u.runOnMainThread(func() {
		if err := u.pollEvents(); err != nil {
			ferr = err
			return
		}
		for u.window.GetAttrib(glfw.Focused) == 0 {
			// Wait for an arbitrary period to avoid busy loop.
			time.Sleep(time.Second / 60)
			if err := u.pollEvents(); err != nil {
				ferr = err
				return
			}
			if u.window.ShouldClose() {
				return
			}
		}
	})
	if ferr != nil {
		return nil, ferr
	}
	return RenderEvent{}, nil
}

func (u *UserInterface) Terminate() {
	u.runOnMainThread(func() {
		glfw.Terminate()
	})
	close(u.funcs)
	u.funcs = nil
}

func (u *UserInterface) SwapBuffers() {
	u.runOnMainThread(func() {
		u.swapBuffers()
	})
}

func (u *UserInterface) swapBuffers() {
	// The bound framebuffer must be the default one (0) before swapping buffers.
	u.context.BindZeroFramebuffer()
	u.context.RunOnContextThread(func() error {
		u.window.SwapBuffers()
		return nil
	})
}

func (u *UserInterface) setScreenSize(width, height, scale int) bool {
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
	if width*u.actualScreenScale() < minWindowWidth {
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
	u.sizeChanged = true
	return true
}
