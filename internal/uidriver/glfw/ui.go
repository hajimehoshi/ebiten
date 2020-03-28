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

package glfw

import (
	"context"
	"fmt"
	"image"
	"os"
	"runtime"
	"sync"
	"time"
	"unsafe"

	"github.com/hajimehoshi/ebiten/internal/devicescale"
	"github.com/hajimehoshi/ebiten/internal/driver"
	"github.com/hajimehoshi/ebiten/internal/glfw"
	"github.com/hajimehoshi/ebiten/internal/hooks"
	"github.com/hajimehoshi/ebiten/internal/thread"
)

type UserInterface struct {
	title  string
	window *glfw.Window

	// windowWidth and windowHeight represents a window size.
	// The unit is device-dependent pixels.
	windowWidth  int
	windowHeight int

	running             bool
	toChangeSize        bool
	origPosX            int
	origPosY            int
	runnableOnUnfocused bool
	vsync               bool

	lastDeviceScaleFactor float64

	initMonitor              *glfw.Monitor
	initTitle                string
	initFullscreenWidthInDP  int
	initFullscreenHeightInDP int
	initFullscreen           bool
	initCursorMode           driver.CursorMode
	initWindowDecorated      bool
	initWindowResizable      bool
	initWindowPositionXInDP  int
	initWindowPositionYInDP  int
	initWindowWidthInDP      int
	initWindowHeightInDP     int
	initWindowFloating       bool
	initWindowMaximized      bool
	initScreenTransparent    bool
	initIconImages           []image.Image

	reqWidth  int
	reqHeight int

	input   Input
	iwindow window

	t *thread.Thread
	m sync.RWMutex
}

const (
	maxInt     = int(^uint(0) >> 1)
	minInt     = -maxInt - 1
	invalidPos = minInt
)

var (
	theUI = &UserInterface{
		origPosX:                invalidPos,
		origPosY:                invalidPos,
		initCursorMode:          driver.CursorModeVisible,
		initWindowDecorated:     true,
		initWindowPositionXInDP: invalidPos,
		initWindowPositionYInDP: invalidPos,
		initWindowWidthInDP:     640,
		initWindowHeightInDP:    480,
		vsync:                   true,
	}
)

func init() {
	theUI.input.ui = theUI
	theUI.iwindow.ui = theUI
}

func Get() *UserInterface {
	return theUI
}

func init() {
	hideConsoleWindowOnWindows()
	if err := initialize(); err != nil {
		panic(err)
	}
	glfw.SetMonitorCallback(func(monitor *glfw.Monitor, event glfw.PeripheralEvent) {
		cacheMonitors()
	})
	cacheMonitors()
}

func initialize() error {
	if err := glfw.Init(); err != nil {
		return err
	}
	glfw.WindowHint(glfw.Visible, glfw.False)

	// Create a window to set the initial monitor.
	w, err := glfw.CreateWindow(16, 16, "", nil, nil)
	if err != nil {
		return err
	}
	if w == nil {
		// This can happen on Windows Remote Desktop (#903).
		panic("glfw: glfw.CreateWindow must not return nil")
	}

	// Create a window and leave it as it is: this affects the result of currentMonitorFromPosition.
	theUI.window = w
	theUI.initMonitor = theUI.currentMonitor()
	v := theUI.initMonitor.GetVideoMode()
	theUI.initFullscreenWidthInDP = int(theUI.toDeviceIndependentPixel(float64(v.Width)))
	theUI.initFullscreenHeightInDP = int(theUI.toDeviceIndependentPixel(float64(v.Height)))

	return nil
}

type cachedMonitor struct {
	m  *glfw.Monitor
	vm *glfw.VidMode
	// Pos of monitor in virtual coords
	x int
	y int
}

// monitors is the monitor list cache for desktop glfw compile targets.
// populated by 'cacheMonitors' which is called on init and every
// monitor config change event.
//
// monitors must be manipulated on the main thread.
var monitors []*cachedMonitor

func cacheMonitors() {
	monitors = nil
	ms := glfw.GetMonitors()
	for _, m := range ms {
		x, y := m.GetPos()
		monitors = append(monitors, &cachedMonitor{
			m:  m,
			vm: m.GetVideoMode(),
			x:  x,
			y:  y,
		})
	}
}

// getCachedMonitor returns a monitor for the given window x/y
// returns false if monitor is not found.
//
// getCachedMonitor must be called on the main thread.
func getCachedMonitor(wx, wy int) (*cachedMonitor, bool) {
	for _, m := range monitors {
		if m.x <= wx && wx < m.x+m.vm.Width && m.y <= wy && wy < m.y+m.vm.Height {
			return m, true
		}
	}
	return nil, false
}

func (u *UserInterface) isRunning() bool {
	u.m.RLock()
	v := u.running
	u.m.RUnlock()
	return v
}

func (u *UserInterface) setRunning(running bool) {
	u.m.Lock()
	u.running = running
	u.m.Unlock()
}

func (u *UserInterface) getInitTitle() string {
	u.m.RLock()
	v := u.initTitle
	u.m.RUnlock()
	return v
}

func (u *UserInterface) setInitTitle(title string) {
	u.m.RLock()
	u.initTitle = title
	u.m.RUnlock()
}

func (u *UserInterface) isInitFullscreen() bool {
	u.m.RLock()
	v := u.initFullscreen
	u.m.RUnlock()
	return v
}

func (u *UserInterface) setInitFullscreen(initFullscreen bool) {
	u.m.Lock()
	u.initFullscreen = initFullscreen
	u.m.Unlock()
}

func (u *UserInterface) getInitCursorMode() driver.CursorMode {
	u.m.RLock()
	v := u.initCursorMode
	u.m.RUnlock()
	return v
}

func (u *UserInterface) setInitCursorMode(mode driver.CursorMode) {
	u.m.Lock()
	u.initCursorMode = mode
	u.m.Unlock()
}

func (u *UserInterface) isInitWindowDecorated() bool {
	u.m.RLock()
	v := u.initWindowDecorated
	u.m.RUnlock()
	return v
}

func (u *UserInterface) setInitWindowDecorated(decorated bool) {
	u.m.Lock()
	u.initWindowDecorated = decorated
	u.m.Unlock()
}

func (u *UserInterface) isRunnableOnUnfocused() bool {
	u.m.RLock()
	v := u.runnableOnUnfocused
	u.m.RUnlock()
	return v
}

func (u *UserInterface) setRunnableOnUnfocused(runnableOnUnfocused bool) {
	u.m.Lock()
	u.runnableOnUnfocused = runnableOnUnfocused
	u.m.Unlock()
}

func (u *UserInterface) isInitWindowResizable() bool {
	u.m.RLock()
	v := u.initWindowResizable
	u.m.RUnlock()
	return v
}

func (u *UserInterface) setInitWindowResizable(resizable bool) {
	u.m.Lock()
	u.initWindowResizable = resizable
	u.m.Unlock()
}

func (u *UserInterface) isInitScreenTransparent() bool {
	u.m.RLock()
	v := u.initScreenTransparent
	u.m.RUnlock()
	return v
}

func (u *UserInterface) setInitScreenTransparent(transparent bool) {
	u.m.RLock()
	u.initScreenTransparent = transparent
	u.m.RUnlock()
}

func (u *UserInterface) getInitIconImages() []image.Image {
	u.m.RLock()
	i := u.initIconImages
	u.m.RUnlock()
	return i
}

func (u *UserInterface) setInitIconImages(iconImages []image.Image) {
	u.m.Lock()
	u.initIconImages = iconImages
	u.m.Unlock()
}

func (u *UserInterface) getInitWindowPosition() (int, int) {
	u.m.RLock()
	defer u.m.RUnlock()
	if u.initWindowPositionXInDP != invalidPos && u.initWindowPositionYInDP != invalidPos {
		return u.initWindowPositionXInDP, u.initWindowPositionYInDP
	}
	return invalidPos, invalidPos
}

func (u *UserInterface) setInitWindowPosition(x, y int) {
	u.m.Lock()
	defer u.m.Unlock()

	u.initWindowPositionXInDP = x
	u.initWindowPositionYInDP = y
}

func (u *UserInterface) getInitWindowSize() (int, int) {
	u.m.Lock()
	w, h := u.initWindowWidthInDP, u.initWindowHeightInDP
	u.m.Unlock()
	return w, h
}

func (u *UserInterface) setInitWindowSize(width, height int) {
	u.m.Lock()
	u.initWindowWidthInDP, u.initWindowHeightInDP = width, height
	u.m.Unlock()
}

func (u *UserInterface) isInitWindowFloating() bool {
	u.m.Lock()
	f := u.initWindowFloating
	u.m.Unlock()
	return f
}

func (u *UserInterface) setInitWindowFloating(floating bool) {
	u.m.Lock()
	u.initWindowFloating = floating
	u.m.Unlock()
}

func (u *UserInterface) isInitWindowMaximized() bool {
	u.m.Lock()
	f := u.initWindowMaximized
	u.m.Unlock()
	return f
}

func (u *UserInterface) setInitWindowMaximized(floating bool) {
	u.m.Lock()
	u.initWindowMaximized = floating
	u.m.Unlock()
}

// toDeviceIndependentPixel must be called from the main thread.
func (u *UserInterface) toDeviceIndependentPixel(x float64) float64 {
	return x / u.glfwScale()
}

// toDeviceDependentPixel must be called from the main thread.
func (u *UserInterface) toDeviceDependentPixel(x float64) float64 {
	return x * u.glfwScale()
}

func (u *UserInterface) ScreenSizeInFullscreen() (int, int) {
	if !u.isRunning() {
		return u.initFullscreenWidthInDP, u.initFullscreenHeightInDP
	}

	var w, h int
	_ = u.t.Call(func() error {
		v := u.currentMonitor().GetVideoMode()
		w = int(u.toDeviceIndependentPixel(float64(v.Width)))
		h = int(u.toDeviceIndependentPixel(float64(v.Height)))
		return nil
	})
	return w, h
}

// isFullscreen must be called from the main thread.
func (u *UserInterface) isFullscreen() bool {
	if !u.isRunning() {
		panic("glfw: isFullscreen can't be called before the main loop starts")
	}
	return u.window.GetMonitor() != nil
}

func (u *UserInterface) IsFullscreen() bool {
	if !u.isRunning() {
		return u.isInitFullscreen()
	}
	b := false
	_ = u.t.Call(func() error {
		b = u.isFullscreen()
		return nil
	})
	return b
}

func (u *UserInterface) SetFullscreen(fullscreen bool) {
	if !u.isRunning() {
		u.setInitFullscreen(fullscreen)
		return
	}

	var update bool
	_ = u.t.Call(func() error {
		update = u.isFullscreen() != fullscreen
		return nil
	})
	if !update {
		return
	}

	var w, h int
	_ = u.t.Call(func() error {
		w, h = u.windowWidth, u.windowHeight
		return nil
	})
	u.setWindowSize(w, h, fullscreen, u.vsync)
}

func (u *UserInterface) IsFocused() bool {
	if !u.isRunning() {
		return false
	}

	var focused bool
	_ = u.t.Call(func() error {
		focused = u.window.GetAttrib(glfw.Focused) == glfw.True
		return nil
	})
	return focused
}

func (u *UserInterface) SetRunnableOnUnfocused(runnableOnUnfocused bool) {
	u.setRunnableOnUnfocused(runnableOnUnfocused)
}

func (u *UserInterface) IsRunnableOnUnfocused() bool {
	return u.isRunnableOnUnfocused()
}

func (u *UserInterface) SetVsyncEnabled(enabled bool) {
	if !u.isRunning() {
		// In general, m is used for locking init* values.
		// m is not used for updating vsync in setWindowSize so far, but
		// it should be OK since any goroutines can't reach here when
		// the game already starts and setWindowSize can be called.
		u.m.Lock()
		u.vsync = enabled
		u.m.Unlock()
		return
	}
	var w, h int
	_ = u.t.Call(func() error {
		w, h = u.windowWidth, u.windowHeight
		return nil
	})
	u.setWindowSize(w, h, u.isFullscreen(), enabled)
}

func (u *UserInterface) IsVsyncEnabled() bool {
	u.m.RLock()
	r := u.vsync
	u.m.RUnlock()
	return r
}

func (u *UserInterface) CursorMode() driver.CursorMode {
	if !u.isRunning() {
		return u.getInitCursorMode()
	}
	var v driver.CursorMode
	_ = u.t.Call(func() error {
		mode := u.window.GetInputMode(glfw.CursorMode)
		switch mode {
		case glfw.CursorNormal:
			v = driver.CursorModeVisible
		case glfw.CursorHidden:
			v = driver.CursorModeHidden
		case glfw.CursorDisabled:
			v = driver.CursorModeCaptured
		default:
			panic(fmt.Sprintf("invalid cursor mode: %d", mode))
		}
		return nil
	})
	return v
}

func (u *UserInterface) SetCursorMode(mode driver.CursorMode) {
	if !u.isRunning() {
		u.setInitCursorMode(mode)
		return
	}
	_ = u.t.Call(func() error {
		var c int
		switch mode {
		case driver.CursorModeVisible:
			c = glfw.CursorNormal
		case driver.CursorModeHidden:
			c = glfw.CursorHidden
		case driver.CursorModeCaptured:
			c = glfw.CursorDisabled
		default:
			panic(fmt.Sprintf("invalid cursor mode: %d", mode))
		}
		u.window.SetInputMode(glfw.CursorMode, c)
		return nil
	})
}

func (u *UserInterface) DeviceScaleFactor() float64 {
	if !u.isRunning() {
		return devicescale.GetAt(u.initMonitor.GetPos())
	}

	f := 0.0
	_ = u.t.Call(func() error {
		f = u.deviceScaleFactor()
		return nil
	})
	return f
}

// deviceScaleFactor must be called from the main thread.
func (u *UserInterface) deviceScaleFactor() float64 {
	// Before calling SetWindowPosition, the window's positin is not reliable.
	if u.iwindow.setPositionCalled {
		// Avoid calling monitor.GetPos if we have the monitor position cached already.
		if cm, ok := getCachedMonitor(u.window.GetPos()); ok {
			return devicescale.GetAt(cm.x, cm.y)
		}
	}
	return devicescale.GetAt(u.currentMonitor().GetPos())
}

func init() {
	// Lock the main thread.
	runtime.LockOSThread()
}

func (u *UserInterface) Run(uicontext driver.UIContext) error {
	// Initialize the main thread first so the thread is available at u.run (#809).
	u.t = thread.New()
	u.Graphics().SetThread(u.t)

	ctx, cancel := context.WithCancel(context.Background())

	ch := make(chan error, 1)
	go func() {
		defer cancel()
		defer close(ch)
		if err := u.run(uicontext); err != nil {
			ch <- err
		}
	}()

	u.setRunning(true)
	u.t.Loop(ctx)
	u.setRunning(false)
	return <-ch
}

func (u *UserInterface) RunWithoutMainLoop(context driver.UIContext) {
	panic("glfw: RunWithoutMainLoop is not implemented")
}

// createWindow creates a GLFW window.
//
// createWindow must be called from the main thread.
//
// createWindow does not set the position or size so far.
func (u *UserInterface) createWindow() error {
	if u.window != nil {
		panic("glfw: u.window must not exist at createWindow")
	}

	// As a start, create a window with temporary size to create OpenGL context thread.
	window, err := glfw.CreateWindow(16, 16, "", nil, nil)
	if err != nil {
		return err
	}
	u.window = window

	if u.Graphics().IsGL() {
		u.window.MakeContextCurrent()
	}

	u.window.SetInputMode(glfw.StickyMouseButtonsMode, glfw.True)
	u.window.SetInputMode(glfw.StickyKeysMode, glfw.True)

	mode := glfw.CursorNormal
	switch u.getInitCursorMode() {
	case driver.CursorModeHidden:
		mode = glfw.CursorHidden
	case driver.CursorModeCaptured:
		mode = glfw.CursorDisabled
	}
	u.window.SetInputMode(glfw.CursorMode, mode)
	u.window.SetTitle(u.title)
	// TODO: Set icons

	u.window.SetSizeCallback(func(_ *glfw.Window, width, height int) {
		if u.window.GetAttrib(glfw.Resizable) == glfw.False {
			return
		}
		if u.isFullscreen() {
			return
		}
		u.reqWidth = width
		u.reqHeight = height
	})

	return nil
}

func (u *UserInterface) run(context driver.UIContext) error {
	if err := u.t.Call(func() error {
		// The window is created at initialize().
		u.window.Destroy()
		u.window = nil

		if u.Graphics().IsGL() {
			glfw.WindowHint(glfw.ContextVersionMajor, 2)
			glfw.WindowHint(glfw.ContextVersionMinor, 1)
		} else {
			glfw.WindowHint(glfw.ClientAPI, glfw.NoAPI)
		}

		decorated := glfw.False
		if u.isInitWindowDecorated() {
			decorated = glfw.True
		}
		glfw.WindowHint(glfw.Decorated, decorated)

		transparent := glfw.False
		if u.isInitScreenTransparent() {
			transparent = glfw.True
		}
		glfw.WindowHint(glfw.TransparentFramebuffer, transparent)
		u.Graphics().SetTransparent(u.isInitScreenTransparent())

		resizable := glfw.False
		if u.isInitWindowResizable() {
			resizable = glfw.True
		}
		glfw.WindowHint(glfw.Resizable, resizable)

		floating := glfw.False
		if u.isInitWindowFloating() {
			floating = glfw.True
		}
		glfw.WindowHint(glfw.Floating, floating)

		// Set the window visible explicitly or the application freezes on Wayland (#974).
		if os.Getenv("WAYLAND_DISPLAY") != "" {
			glfw.WindowHint(glfw.Visible, glfw.True)
		}

		if err := u.createWindow(); err != nil {
			return err
		}

		if i := u.getInitIconImages(); i != nil {
			u.window.SetIcon(i)
		}
		return nil
	}); err != nil {
		return err
	}

	setPosition := func() {
		u.iwindow.SetPosition(u.getInitWindowPosition())
	}
	setSize := func() {
		ww, wh := u.getInitWindowSize()
		ww = int(u.toDeviceDependentPixel(float64(ww)))
		wh = int(u.toDeviceDependentPixel(float64(wh)))
		u.setWindowSize(ww, wh, u.isFullscreen(), u.vsync)
	}

	// Set the window size and the window position in this order on Linux or other UNIX using X (#1118),
	// but this should be inverted on Windows. This is very tricky, but there is no obvious way to solve this.
	// This doesn't matter on macOS.
	if runtime.GOOS == "windows" {
		setPosition()
		setSize()
	} else {
		setSize()
		setPosition()
	}

	// Maximizing a window requires a proper size and position. Call Maximize here (#1117).
	if u.isInitWindowMaximized() {
		_ = u.t.Call(func() error {
			u.window.Maximize()
			return nil
		})
	}

	_ = u.t.Call(func() error {
		u.title = u.getInitTitle()
		u.window.SetTitle(u.title)
		u.window.Show()
		return nil
	})

	var w unsafe.Pointer
	_ = u.t.Call(func() error {
		w = u.nativeWindow()
		return nil
	})
	if g, ok := u.Graphics().(interface{ SetWindow(unsafe.Pointer) }); ok {
		g.SetWindow(w)
	}
	return u.loop(context)
}

func (u *UserInterface) updateSize(context driver.UIContext) {
	var w, h int
	_ = u.t.Call(func() error {
		w, h = u.windowWidth, u.windowHeight
		return nil
	})
	u.setWindowSize(w, h, u.isFullscreen(), u.vsync)

	sizeChanged := false
	_ = u.t.Call(func() error {
		if !u.toChangeSize {
			return nil
		}

		u.toChangeSize = false
		sizeChanged = true
		return nil
	})
	if sizeChanged {
		var w, h float64
		_ = u.t.Call(func() error {
			var ww, wh int
			if u.isFullscreen() {
				v := u.currentMonitor().GetVideoMode()
				ww = v.Width
				wh = v.Height
			} else {
				ww, wh = u.windowWidth, u.windowHeight
			}
			w = u.toDeviceIndependentPixel(float64(ww))
			h = u.toDeviceIndependentPixel(float64(wh))
			return nil
		})
		context.Layout(w, h)
	}
}

func (u *UserInterface) update(context driver.UIContext) error {
	shouldClose := false
	_ = u.t.Call(func() error {
		shouldClose = u.window.ShouldClose()
		return nil
	})
	if shouldClose {
		return driver.RegularTermination
	}

	if u.isInitFullscreen() {
		var w, h int
		_ = u.t.Call(func() error {
			w, h = u.window.GetSize()
			return nil
		})
		u.setWindowSize(w, h, true, u.vsync)
		u.setInitFullscreen(false)
	}

	// This call is needed for initialization.
	u.updateSize(context)

	_ = u.t.Call(func() error {
		glfw.PollEvents()
		return nil
	})
	u.input.update(u.window, context)
	_ = u.t.Call(func() error {
		defer hooks.ResumeAudio()

		for !u.isRunnableOnUnfocused() && u.window.GetAttrib(glfw.Focused) == 0 {
			hooks.SuspendAudio()
			// Wait for an arbitrary period to avoid busy loop.
			time.Sleep(time.Second / 60)
			glfw.PollEvents()
			if u.window.ShouldClose() {
				return nil
			}
		}
		return nil
	})
	if err := context.Update(func() {
		// The offscreens must be updated every frame (#490).
		u.updateSize(context)
	}); err != nil {
		return err
	}

	// Update the screen size when the window is resizable.
	var w, h int
	_ = u.t.Call(func() error {
		w, h = u.reqWidth, u.reqHeight
		return nil
	})
	if w != 0 || h != 0 {
		u.setWindowSize(w, h, u.isFullscreen(), u.vsync)
	}
	_ = u.t.Call(func() error {
		u.reqWidth = 0
		u.reqHeight = 0
		return nil
	})
	return nil
}

func (u *UserInterface) loop(context driver.UIContext) error {
	defer func() {
		_ = u.t.Call(func() error {
			glfw.Terminate()
			return nil
		})
	}()
	for {
		var unfocused bool

		// On Windows, the focusing state might be always false (#987).
		// On Windows, even if a window is in another workspace, vsync seems to work.
		// Then let's assume the window is always 'focused' as a workaround.
		if runtime.GOOS != "windows" {
			unfocused = u.window.GetAttrib(glfw.Focused) == glfw.False
		}

		var t1, t2 time.Time

		if unfocused {
			t1 = time.Now()
		}
		if err := u.update(context); err != nil {
			return err
		}

		_ = u.t.Call(func() error {
			u.swapBuffers()
			return nil
		})
		if unfocused {
			t2 = time.Now()
		}

		// When a window is not focused, SwapBuffers might return immediately and CPU might be busy.
		// Mitigate this by sleeping (#982).
		if unfocused {
			d := t2.Sub(t1)
			const wait = time.Second / 60
			if d < wait {
				time.Sleep(wait - d)
			}
		}
	}
}

// swapBuffers must be called from the main thread.
func (u *UserInterface) swapBuffers() {
	if u.Graphics().IsGL() {
		u.window.SwapBuffers()
	}
}

func (u *UserInterface) setWindowSize(width, height int, fullscreen bool, vsync bool) {
	windowRecreated := false

	_ = u.t.Call(func() error {
		if u.windowWidth == width && u.windowHeight == height && u.isFullscreen() == fullscreen && u.vsync == vsync && u.lastDeviceScaleFactor == u.deviceScaleFactor() {
			return nil
		}

		if width < 1 {
			width = 1
		}
		if height < 1 {
			height = 1
		}

		u.vsync = vsync
		u.lastDeviceScaleFactor = u.deviceScaleFactor()

		// To make sure the current existing framebuffers are rendered,
		// swap buffers here before SetSize is called.
		u.swapBuffers()

		if fullscreen {
			if u.origPosX == invalidPos || u.origPosY == invalidPos {
				u.origPosX, u.origPosY = u.window.GetPos()
			}
			m := u.currentMonitor()
			v := m.GetVideoMode()
			u.window.SetMonitor(m, 0, 0, v.Width, v.Height, v.RefreshRate)

			// Swapping buffer is necesary to prevent the image lag (#1004).
			// TODO: This might not work when vsync is disabled.
			if u.Graphics().IsGL() {
				glfw.PollEvents()
				u.swapBuffers()
			}
		} else {
			if u.window.GetMonitor() != nil {
				if u.Graphics().IsGL() {
					// When OpenGL is used, swapping buffer is enough to solve the image-lag
					// issue (#1004). Rather, recreating window destroys GPU resources.
					// TODO: This might not work when vsync is disabled.
					u.window.SetMonitor(nil, 0, 0, 16, 16, 0)
					glfw.PollEvents()
					u.swapBuffers()
				} else {
					// Recreate the window since an image lag remains after coming back from
					// fullscreen (#1004).
					if u.window != nil {
						u.window.Destroy()
						u.window = nil
					}
					if err := u.createWindow(); err != nil {
						// TODO: This should return an error.
						panic(fmt.Sprintf("glfw: failed to recreate window: %v", err))
					}
					u.window.Show()
					windowRecreated = true
				}
			}

			// On Windows, giving a too small width doesn't call a callback (#165).
			// To prevent hanging up, return asap if the width is too small.
			// 126 is an arbitrary number and I guess this is small enough.
			minWindowWidth := int(u.toDeviceDependentPixel(126))
			if u.window.GetAttrib(glfw.Decorated) == glfw.False {
				minWindowWidth = 1
			}
			if width < minWindowWidth {
				width = minWindowWidth
			}

			if u.origPosX != invalidPos && u.origPosY != invalidPos {
				x := u.origPosX
				y := u.origPosY
				u.window.SetPos(x, y)
				// Dirty hack for macOS (#703). Rendering doesn't work correctly with one SetPos, but
				// work with two or more SetPos.
				if runtime.GOOS == "darwin" {
					u.window.SetPos(x+1, y)
					u.window.SetPos(x, y)
				}
				u.origPosX = invalidPos
				u.origPosY = invalidPos
			}

			// Set the window size after the position. The order matters.
			// In the opposite order, the window size might not be correct when going back from fullscreen with multi monitors.
			oldW, oldH := u.window.GetSize()
			newW := width
			newH := height
			if oldW != newW || oldH != newH {
				ch := make(chan struct{})
				u.window.SetFramebufferSizeCallback(func(_ *glfw.Window, _, _ int) {
					u.window.SetFramebufferSizeCallback(nil)
					close(ch)
				})
				u.window.SetSize(newW, newH)
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

		// As width might be updated, update windowWidth/Height here.
		u.windowWidth = width
		u.windowHeight = height

		if u.Graphics().IsGL() {
			// SwapInterval is affected by the current monitor of the window.
			// This needs to be called at least after SetMonitor.
			// Without SwapInterval after SetMonitor, vsynch doesn't work (#375).
			//
			// TODO: (#405) If triple buffering is needed, SwapInterval(0) should be called,
			// but is this correct? If glfw.SwapInterval(0) and the driver doesn't support triple
			// buffering, what will happen?
			if u.vsync {
				glfw.SwapInterval(1)
			} else {
				glfw.SwapInterval(0)
			}
		}
		u.Graphics().SetVsyncEnabled(vsync)

		u.toChangeSize = true
		return nil
	})

	if windowRecreated {
		if g, ok := u.Graphics().(interface{ SetWindow(unsafe.Pointer) }); ok {
			g.SetWindow(u.nativeWindow())
		}
	}
}

// currentMonitor returns the monitor most suitable with the current window.
//
// currentMonitor must be called on the main thread.
func (u *UserInterface) currentMonitor() *glfw.Monitor {
	if w := u.window; w != nil {
		// GetMonitor is available only on fullscreen.
		if m := w.GetMonitor(); m != nil {
			return m
		}
	}
	// Get the monitor which the current window belongs to. This requires OS API.
	return u.currentMonitorFromPosition()
}

func (u *UserInterface) SetScreenTransparent(transparent bool) {
	if !u.isRunning() {
		u.setInitScreenTransparent(transparent)
		return
	}
	panic("glfw: SetScreenTransparent can't be called after the main loop starts")
}

func (u *UserInterface) IsScreenTransparent() bool {
	if !u.isRunning() {
		return u.isInitScreenTransparent()
	}
	val := false
	_ = u.t.Call(func() error {
		val = u.window.GetAttrib(glfw.TransparentFramebuffer) == glfw.True
		return nil
	})
	return val
}

func (u *UserInterface) Input() driver.Input {
	return &u.input
}

func (u *UserInterface) Window() driver.Window {
	return &u.iwindow
}
