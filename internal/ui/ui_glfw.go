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

// +build darwin freebsd linux windows
// +build !js
// +build !android
// +build !ios

package ui

import (
	"image"
	"math"
	"runtime"
	"sync"
	"time"

	"github.com/go-gl/glfw/v3.2/glfw"

	"github.com/hajimehoshi/ebiten/internal/devicescale"
	"github.com/hajimehoshi/ebiten/internal/opengl"
)

type userInterface struct {
	title       string
	window      *glfw.Window
	width       int
	windowWidth int
	height      int

	scale           float64
	fullscreenScale float64

	running              bool
	sizeChanged          bool
	origPosX             int
	origPosY             int
	runnableInBackground bool

	initFullscreen    bool
	initCursorVisible bool
	initIconImages    []image.Image

	funcs chan func()

	m sync.Mutex
}

var (
	currentUI = &userInterface{
		sizeChanged:       true,
		origPosX:          -1,
		origPosY:          -1,
		initCursorVisible: true,
	}
	currentUIInitialized = make(chan struct{})
)

func init() {
	runtime.LockOSThread()
}

func initialize() error {
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
	hideConsoleWindowOnWindows()
	currentUI.window = window
	currentUI.funcs = make(chan func())

	currentUI.window.MakeContextCurrent()

	mode := glfw.CursorNormal
	if !currentUI.isInitCursorVisible() {
		mode = glfw.CursorHidden
	}
	if i := currentUI.getInitIconImages(); i != nil {
		currentUI.window.SetIcon(i)
	}
	currentUI.window.SetInputMode(glfw.CursorMode, mode)
	currentUI.window.SetInputMode(glfw.StickyMouseButtonsMode, glfw.True)
	currentUI.window.SetInputMode(glfw.StickyKeysMode, glfw.True)
	return nil
}

func RunMainThreadLoop(ch <-chan error) error {
	// This must be called on the main thread.

	if err := initialize(); err != nil {
		return err
	}
	close(currentUIInitialized)

	// TODO: Check this is done on the main thread.
	currentUI.setRunning(true)
	defer func() {
		currentUI.setRunning(false)
	}()
	for {
		select {
		case f := <-currentUI.funcs:
			f()
		case err := <-ch:
			// ch returns a value not only when an error occur but also it is closed.
			return err
		}
	}
}

func (u *userInterface) isRunning() bool {
	u.m.Lock()
	v := u.running
	u.m.Unlock()
	return v
}

func (u *userInterface) setRunning(running bool) {
	u.m.Lock()
	u.running = running
	u.m.Unlock()
}

func (u *userInterface) isInitFullscreen() bool {
	u.m.Lock()
	v := u.initFullscreen
	u.m.Unlock()
	return v
}

func (u *userInterface) setInitFullscreen(initFullscreen bool) {
	u.m.Lock()
	u.initFullscreen = initFullscreen
	u.m.Unlock()
}

func (u *userInterface) isInitCursorVisible() bool {
	u.m.Lock()
	v := u.initCursorVisible
	u.m.Unlock()
	return v
}

func (u *userInterface) setInitCursorVisible(visible bool) {
	u.m.Lock()
	u.initCursorVisible = visible
	u.m.Unlock()
}

func (u *userInterface) isRunnableInBackground() bool {
	u.m.Lock()
	v := u.runnableInBackground
	u.m.Unlock()
	return v
}

func (u *userInterface) setRunnableInBackground(runnableInBackground bool) {
	u.m.Lock()
	u.runnableInBackground = runnableInBackground
	u.m.Unlock()
}

func (u *userInterface) getInitIconImages() []image.Image {
	u.m.Lock()
	i := u.initIconImages
	u.m.Unlock()
	return i
}

func (u *userInterface) setInitIconImages(iconImages []image.Image) {
	u.m.Lock()
	u.initIconImages = iconImages
	u.m.Unlock()
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

func SetScreenSize(width, height int) bool {
	u := currentUI
	if !u.isRunning() {
		panic("ui: Run is not called yet")
	}
	r := false
	_ = u.runOnMainThread(func() error {
		r = u.setScreenSize(width, height, u.scale, u.fullscreen())
		return nil
	})
	return r
}

func SetScreenScale(scale float64) bool {
	u := currentUI
	if !u.isRunning() {
		panic("ui: Run is not called yet")
	}
	r := false
	_ = u.runOnMainThread(func() error {
		r = u.setScreenSize(u.width, u.height, scale, u.fullscreen())
		return nil
	})
	return r
}

func ScreenScale() float64 {
	u := currentUI
	if !u.isRunning() {
		return 0
	}
	s := 0.0
	_ = u.runOnMainThread(func() error {
		s = u.scale
		return nil
	})
	return s
}

func (u *userInterface) fullscreen() bool {
	if !u.isRunning() {
		panic("not reached")
	}
	return u.window.GetMonitor() != nil
}

func IsFullscreen() bool {
	u := currentUI
	if !u.isRunning() {
		return u.isInitFullscreen()
	}
	return u.fullscreen()
}

func SetFullscreen(fullscreen bool) {
	u := currentUI
	if !u.isRunning() {
		u.setInitFullscreen(fullscreen)
		return
	}
	_ = u.runOnMainThread(func() error {
		u := currentUI
		u.setScreenSize(u.width, u.height, u.scale, fullscreen)
		return nil
	})
}

func SetRunnableInBackground(runnableInBackground bool) {
	currentUI.setRunnableInBackground(runnableInBackground)
}

func IsRunnableInBackground() bool {
	return currentUI.isRunnableInBackground()
}

func SetWindowIcon(iconImages []image.Image) {
	if !currentUI.isRunning() {
		currentUI.setInitIconImages(iconImages)
		return
	}
	_ = currentUI.runOnMainThread(func() error {
		currentUI.window.SetIcon(iconImages)
		return nil
	})
}

func ScreenOffset() (float64, float64) {
	u := currentUI
	if !u.isRunning() {
		return 0, 0
	}
	if !IsFullscreen() {
		if u.width == u.windowWidth {
			return 0, 0
		}
		return (float64(u.windowWidth)*u.actualScreenScale() - float64(u.width)*u.actualScreenScale()) / 2, 0
	}
	ox := 0.0
	oy := 0.0
	m := glfw.GetPrimaryMonitor()
	v := m.GetVideoMode()
	d := devicescale.DeviceScale()
	_ = u.runOnMainThread(func() error {
		ox = (float64(v.Width)*d/glfwScale() - float64(u.width)*u.actualScreenScale()) / 2
		oy = (float64(v.Height)*d/glfwScale() - float64(u.height)*u.actualScreenScale()) / 2
		return nil
	})
	return ox, oy
}

func adjustCursorPosition(x, y int) (int, int) {
	u := currentUI
	if !u.isRunning() {
		return x, y
	}
	ox, oy := ScreenOffset()
	s := 0.0
	_ = currentUI.runOnMainThread(func() error {
		s = currentUI.actualScreenScale()
		return nil
	})
	return x - int(ox/s), y - int(oy/s)
}

func IsCursorVisible() bool {
	u := currentUI
	if !u.isRunning() {
		return u.isInitCursorVisible()
	}
	v := false
	_ = currentUI.runOnMainThread(func() error {
		v = currentUI.window.GetInputMode(glfw.CursorMode) == glfw.CursorNormal
		return nil
	})
	return v
}

func SetCursorVisible(visible bool) {
	u := currentUI
	if !u.isRunning() {
		u.setInitCursorVisible(visible)
		return
	}
	_ = currentUI.runOnMainThread(func() error {
		c := glfw.CursorNormal
		if !visible {
			c = glfw.CursorHidden
		}
		currentUI.window.SetInputMode(glfw.CursorMode, c)
		return nil
	})
}

func Run(width, height int, scale float64, title string, g GraphicsContext) error {
	<-currentUIInitialized

	u := currentUI
	// GLContext must be created before setting the screen size, which requires
	// swapping buffers.
	opengl.Init(currentUI.runOnMainThread)
	_ = u.runOnMainThread(func() error {
		m := glfw.GetPrimaryMonitor()
		v := m.GetVideoMode()

		// The game is in window mode (not fullscreen mode) at the first state.
		// Don't refer u.initFullscreen here to avoid some GLFW problems.
		u.setScreenSize(width, height, scale, false)
		u.title = title
		u.window.SetTitle(title)
		u.window.Show()

		w, h := u.glfwSize()
		x := (v.Width - w) / 2
		y := (v.Height - h) / 3
		x, y = adjustWindowPosition(x, y)
		u.window.SetPos(x, y)
		return nil
	})
	return u.loop(g)
}

func (u *userInterface) glfwSize() (int, int) {
	w := int(float64(u.windowWidth) * u.getScale() * glfwScale())
	h := int(float64(u.height) * u.getScale() * glfwScale())
	return w, h
}

func (u *userInterface) getScale() float64 {
	if !u.fullscreen() {
		return u.scale
	}
	if u.fullscreenScale == 0 {
		m := glfw.GetPrimaryMonitor()
		v := m.GetVideoMode()
		sw := float64(v.Width) / glfwScale() / float64(u.width)
		sh := float64(v.Height) / glfwScale() / float64(u.height)
		s := sw
		if s > sh {
			s = sh
		}
		u.fullscreenScale = s
	}
	return u.fullscreenScale
}

func (u *userInterface) actualScreenScale() float64 {
	return u.getScale() * devicescale.DeviceScale()
}

func (u *userInterface) pollEvents() {
	glfw.PollEvents()
	currentInput.update(u.window, u.getScale()*glfwScale())
}

func (u *userInterface) update(g GraphicsContext) error {
	shouldClose := false
	_ = u.runOnMainThread(func() error {
		shouldClose = u.window.ShouldClose()
		return nil
	})
	if shouldClose {
		return &RegularTermination{}
	}

	_ = u.runOnMainThread(func() error {
		if u.isInitFullscreen() {
			u := currentUI
			u.setScreenSize(u.width, u.height, u.scale, true)
			u.setInitFullscreen(false)
		}
		return nil
	})

	actualScale := 0.0
	sizeChanged := false
	_ = u.runOnMainThread(func() error {
		if !u.sizeChanged {
			return nil
		}
		u.sizeChanged = false
		actualScale = u.actualScreenScale()
		sizeChanged = true
		return nil
	})
	if sizeChanged {
		g.SetSize(u.width, u.height, actualScale)
	}

	_ = u.runOnMainThread(func() error {
		u.pollEvents()
		for !u.isRunnableInBackground() && u.window.GetAttrib(glfw.Focused) == 0 {
			// Wait for an arbitrary period to avoid busy loop.
			time.Sleep(time.Second / 60)
			u.pollEvents()
			if u.window.ShouldClose() {
				return nil
			}
		}
		return nil
	})
	if err := g.Update(func() {
		currentInput.runeBuffer = currentInput.runeBuffer[:0]
	}); err != nil {
		return err
	}
	return nil
}

func (u *userInterface) loop(g GraphicsContext) error {
	defer func() {
		_ = u.runOnMainThread(func() error {
			glfw.Terminate()
			return nil
		})
	}()
	for {
		if err := u.update(g); err != nil {
			return err
		}
		// The bound framebuffer must be the original screen framebuffer
		// before swapping buffers.
		opengl.GetContext().BindScreenFramebuffer()
		_ = u.runOnMainThread(func() error {
			u.swapBuffers()
			return nil
		})
	}
}

func (u *userInterface) swapBuffers() {
	u.window.SwapBuffers()
}

func (u *userInterface) setScreenSize(width, height int, scale float64, fullscreen bool) bool {
	if u.width == width && u.height == height && u.scale == scale && u.fullscreen() == fullscreen {
		return false
	}

	// On Windows, giving a too small width doesn't call a callback (#165).
	// To prevent hanging up, return asap if the width is too small.
	// 252 is an arbitrary number and I guess this is small enough.
	const minWindowWidth = 252

	u.width = width
	u.windowWidth = width
	s := scale * devicescale.DeviceScale()
	if int(float64(width)*s) < minWindowWidth {
		u.windowWidth = int(math.Ceil(minWindowWidth / s))
	}
	u.height = height
	u.scale = scale
	u.fullscreenScale = 0

	// To make sure the current existing framebuffers are rendered,
	// swap buffers here before SetSize is called.
	u.swapBuffers()

	if fullscreen {
		if u.origPosX < 0 && u.origPosY < 0 {
			u.origPosX, u.origPosY = u.window.GetPos()
		}
		m := glfw.GetPrimaryMonitor()
		v := m.GetVideoMode()
		u.window.SetMonitor(m, 0, 0, v.Width, v.Height, v.RefreshRate)
	} else {
		if u.origPosX >= 0 && u.origPosY >= 0 {
			x := u.origPosX
			y := u.origPosY
			u.window.SetMonitor(nil, x, y, 16, 16, 0)
			u.origPosX = -1
			u.origPosY = -1
		}

		oldW, oldH := u.window.GetSize()
		newW, newH := u.glfwSize()
		if oldW != newW || oldH != newH {
			ch := make(chan struct{})
			u.window.SetFramebufferSizeCallback(func(_ *glfw.Window, _, _ int) {
				u.window.SetFramebufferSizeCallback(nil)
				close(ch)
			})
			u.window.SetSize(u.glfwSize())
		event:
			for {
				glfw.PollEvents()
				select {
				case <-ch:
					break event
				default:
				}
			}
		}
		// Window title might be lost on macOS after coming back from fullscreen.
		u.window.SetTitle(u.title)
	}
	// SwapInterval is affected by the current monitor of the window.
	// This needs to be called at least after SetMonitor.
	// Without SwapInterval after SetMonitor, vsynch doesn't work (#375).
	//
	// TODO: (#405) If triple buffering is needed, SwapInterval(0) should be called,
	// but is this correct? If glfw.SwapInterval(0) and the driver doesn't support triple
	// buffering, what will happen?
	glfw.SwapInterval(1)

	// TODO: Rename this variable?
	u.sizeChanged = true
	return true
}
