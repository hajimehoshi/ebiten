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
	window           *glfw.Window
	width            int
	height           int
	scale            float64
	deviceScale      float64
	framebufferScale float64
	context          *opengl.Context
	funcs            chan func()
	sizeChanged      bool
}

var currentUI *userInterface

func CurrentUI() UserInterface {
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

	u := &userInterface{
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

	return u.context, nil
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

func (u *userInterface) runOnMainThread(f func()) {
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

func (u *userInterface) SetScreenSize(width, height int) bool {
	r := false
	u.runOnMainThread(func() {
		r = u.setScreenSize(width, height, u.scale)
	})
	return r
}

func (u *userInterface) SetScreenScale(scale float64) bool {
	r := false
	u.runOnMainThread(func() {
		r = u.setScreenSize(u.width, u.height, scale)
	})
	return r
}

func (u *userInterface) ScreenScale() float64 {
	s := 0.0
	u.runOnMainThread(func() {
		s = u.scale
	})
	return s
}

func (u *userInterface) Start(width, height int, scale float64, title string) error {
	var err error
	u.runOnMainThread(func() {
		m := glfw.GetPrimaryMonitor()
		v := m.GetVideoMode()
		u.deviceScale = deviceScale()
		u.framebufferScale = 1
		if !u.setScreenSize(width, height, scale) {
			err = errors.New("ui: Fail to set the screen size")
			return
		}
		u.window.SetTitle(title)
		u.window.Show()

		x := (v.Width - int(float64(width)*u.windowScale())) / 2
		y := (v.Height - int(float64(height)*u.windowScale())) / 3
		u.window.SetPos(x, y)
	})
	return err
}

func (u *userInterface) windowScale() float64 {
	return u.scale * u.deviceScale
}

func (u *userInterface) actualScreenScale() float64 {
	return u.windowScale() * u.framebufferScale
}

func (u *userInterface) pollEvents() error {
	glfw.PollEvents()
	return currentInput.update(u.window, u.windowScale())
}

func (u *userInterface) Update() (interface{}, error) {
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
	// Dummy channel
	ch := make(chan struct{}, 1)
	return RenderEvent{ch}, nil
}

func (u *userInterface) Terminate() error {
	u.runOnMainThread(func() {
		glfw.Terminate()
	})
	close(u.funcs)
	u.funcs = nil
	return nil
}

func (u *userInterface) SwapBuffers() error {
	var err error
	u.runOnMainThread(func() {
		err = u.swapBuffers()
	})
	return err
}

func (u *userInterface) swapBuffers() error {
	// The bound framebuffer must be the default one (0) before swapping buffers.
	if err := u.context.BindScreenFramebuffer(); err != nil {
		return err
	}
	u.context.RunOnContextThread(func() error {
		u.window.SwapBuffers()
		return nil
	})
	return nil
}

func (u *userInterface) FinishRendering() error {
	return nil
}

func (u *userInterface) setScreenSize(width, height int, scale float64) bool {
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
	window.SetSize(int(float64(width)*u.windowScale()), int(float64(height)*u.windowScale()))

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
	u.framebufferScale = float64(fw) / float64(width) / u.windowScale()
	u.sizeChanged = true
	return true
}
