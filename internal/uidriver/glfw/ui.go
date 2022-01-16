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

//go:build !android && !js && !ios
// +build !android,!js,!ios

package glfw

import (
	"fmt"
	"image"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hajimehoshi/ebiten/v2/internal/devicescale"
	"github.com/hajimehoshi/ebiten/v2/internal/driver"
	"github.com/hajimehoshi/ebiten/v2/internal/glfw"
	"github.com/hajimehoshi/ebiten/v2/internal/hooks"
	"github.com/hajimehoshi/ebiten/v2/internal/thread"
)

func driverCursorModeToGLFWCursorMode(mode driver.CursorMode) int {
	switch mode {
	case driver.CursorModeVisible:
		return glfw.CursorNormal
	case driver.CursorModeHidden:
		return glfw.CursorHidden
	case driver.CursorModeCaptured:
		return glfw.CursorDisabled
	default:
		panic(fmt.Sprintf("glfw: invalid driver.CursorMode: %d", mode))
	}
}

type UserInterface struct {
	context driver.UIContext
	title   string
	window  *glfw.Window

	// windowWidthInDIP and windowHeightInDIP represents a window size.
	// The units are device-independent pixels.
	windowWidthInDIP  int
	windowHeightInDIP int

	// The units are device-independent pixels.
	minWindowWidthInDIP  int
	minWindowHeightInDIP int
	maxWindowWidthInDIP  int
	maxWindowHeightInDIP int

	running              uint32
	origPosX             int
	origPosY             int
	runnableOnUnfocused  bool
	fpsMode              driver.FPSMode
	iconImages           []image.Image
	cursorShape          driver.CursorShape
	windowClosingHandled bool
	windowBeingClosed    bool

	// setSizeCallbackEnabled must be accessed from the main thread.
	setSizeCallbackEnabled bool

	// err must be accessed from the main thread.
	err error

	lastDeviceScaleFactor float64

	// These values are not changed after initialized.
	// TODO: the fullscreen size should be updated when the initial window position is changed?
	initMonitor               *glfw.Monitor
	initFullscreenWidthInDIP  int
	initFullscreenHeightInDIP int

	initFullscreen           bool
	initCursorMode           driver.CursorMode
	initWindowDecorated      bool
	initWindowResizable      bool
	initWindowPositionXInDIP int
	initWindowPositionYInDIP int
	initWindowWidthInDIP     int
	initWindowHeightInDIP    int
	initWindowFloating       bool
	initWindowMaximized      bool
	initScreenTransparent    bool
	initFocused              bool

	fpsModeInited bool

	input   Input
	iwindow window

	sizeCallback              glfw.SizeCallback
	closeCallback             glfw.CloseCallback
	framebufferSizeCallback   glfw.FramebufferSizeCallback
	framebufferSizeCallbackCh chan struct{}

	t thread.Thread
	m sync.RWMutex
}

const (
	maxInt     = int(^uint(0) >> 1)
	minInt     = -maxInt - 1
	invalidPos = minInt
)

var (
	theUI = &UserInterface{
		runnableOnUnfocused:      true,
		minWindowWidthInDIP:      glfw.DontCare,
		minWindowHeightInDIP:     glfw.DontCare,
		maxWindowWidthInDIP:      glfw.DontCare,
		maxWindowHeightInDIP:     glfw.DontCare,
		origPosX:                 invalidPos,
		origPosY:                 invalidPos,
		initCursorMode:           driver.CursorModeVisible,
		initWindowDecorated:      true,
		initWindowPositionXInDIP: invalidPos,
		initWindowPositionYInDIP: invalidPos,
		initWindowWidthInDIP:     640,
		initWindowHeightInDIP:    480,
		initFocused:              true,
		fpsMode:                  driver.FPSModeVsyncOn,
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
		updateMonitors()
	})
	updateMonitors()
}

var glfwSystemCursors = map[driver.CursorShape]*glfw.Cursor{}

func initialize() error {
	if err := glfw.Init(); err != nil {
		return err
	}

	glfw.WindowHint(glfw.Visible, glfw.False)
	glfw.WindowHint(glfw.ClientAPI, glfw.NoAPI)

	// Create a window to set the initial monitor.
	w, err := glfw.CreateWindow(16, 16, "", nil, nil)
	if err != nil {
		return err
	}
	if w == nil {
		// This can happen on Windows Remote Desktop (#903).
		panic("glfw: glfw.CreateWindow must not return nil")
	}
	defer w.Destroy()
	initializeWindowAfterCreation(w)
	theUI.waitForFramebufferSizeCallback(w, nil)

	m := initialMonitor(w)
	theUI.initMonitor = m
	v := m.GetVideoMode()
	theUI.initFullscreenWidthInDIP = int(theUI.dipFromGLFWMonitorPixel(float64(v.Width), m))
	theUI.initFullscreenHeightInDIP = int(theUI.dipFromGLFWMonitorPixel(float64(v.Height), m))

	// Create system cursors. These cursors are destroyed at glfw.Terminate().
	glfwSystemCursors[driver.CursorShapeDefault] = nil
	glfwSystemCursors[driver.CursorShapeText] = glfw.CreateStandardCursor(glfw.IBeamCursor)
	glfwSystemCursors[driver.CursorShapeCrosshair] = glfw.CreateStandardCursor(glfw.CrosshairCursor)
	glfwSystemCursors[driver.CursorShapePointer] = glfw.CreateStandardCursor(glfw.HandCursor)
	glfwSystemCursors[driver.CursorShapeEWResize] = glfw.CreateStandardCursor(glfw.HResizeCursor)
	glfwSystemCursors[driver.CursorShapeNSResize] = glfw.CreateStandardCursor(glfw.VResizeCursor)

	return nil
}

type monitor struct {
	m  *glfw.Monitor
	vm *glfw.VidMode
	// Pos of monitor in virtual coords
	x int
	y int
}

// monitors is the monitor list cache for desktop glfw compile targets.
// populated by 'updateMonitors' which is called on init and every
// monitor config change event.
//
// monitors must be manipulated on the main thread.
var monitors []*monitor

func updateMonitors() {
	monitors = nil
	ms := glfw.GetMonitors()
	for _, m := range ms {
		x, y := m.GetPos()
		monitors = append(monitors, &monitor{
			m:  m,
			vm: m.GetVideoMode(),
			x:  x,
			y:  y,
		})
	}
	clearVideoModeScaleCache()
	devicescale.ClearCache()
}

func ensureMonitors() []*monitor {
	if len(monitors) == 0 {
		updateMonitors()
	}
	return monitors
}

// getMonitorFromPosition returns a monitor for the given window x/y,
// or returns nil if monitor is not found.
//
// getMonitorFromPosition must be called on the main thread.
func getMonitorFromPosition(wx, wy int) *monitor {
	for _, m := range ensureMonitors() {
		// TODO: Fix incorrectness in the cases of https://github.com/glfw/glfw/issues/1961.
		// See also internal/devicescale/impl_desktop.go for a maybe better way of doing this.
		if m.x <= wx && wx < m.x+m.vm.Width && m.y <= wy && wy < m.y+m.vm.Height {
			return m
		}
	}
	return nil
}

func (u *UserInterface) isRunning() bool {
	return atomic.LoadUint32(&u.running) != 0
}

func (u *UserInterface) setRunning(running bool) {
	if running {
		atomic.StoreUint32(&u.running, 1)
	} else {
		atomic.StoreUint32(&u.running, 0)
	}
}

func (u *UserInterface) getWindowSizeLimitsInDIP() (minw, minh, maxw, maxh int) {
	u.m.RLock()
	defer u.m.RUnlock()
	return u.minWindowWidthInDIP, u.minWindowHeightInDIP, u.maxWindowWidthInDIP, u.maxWindowHeightInDIP
}

func (u *UserInterface) setWindowSizeLimitsInDIP(minw, minh, maxw, maxh int) bool {
	u.m.RLock()
	defer u.m.RUnlock()
	if u.minWindowWidthInDIP == minw && u.minWindowHeightInDIP == minh && u.maxWindowWidthInDIP == maxw && u.maxWindowHeightInDIP == maxh {
		return false
	}
	u.minWindowWidthInDIP = minw
	u.minWindowHeightInDIP = minh
	u.maxWindowWidthInDIP = maxw
	u.maxWindowHeightInDIP = maxh
	return true
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

func (u *UserInterface) getCursorShape() driver.CursorShape {
	u.m.RLock()
	v := u.cursorShape
	u.m.RUnlock()
	return v
}

func (u *UserInterface) setCursorShape(shape driver.CursorShape) driver.CursorShape {
	u.m.Lock()
	old := u.cursorShape
	u.cursorShape = shape
	u.m.Unlock()
	return old
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

func (u *UserInterface) getIconImages() []image.Image {
	u.m.RLock()
	i := u.iconImages
	u.m.RUnlock()
	return i
}

func (u *UserInterface) setIconImages(iconImages []image.Image) {
	u.m.Lock()
	u.iconImages = iconImages
	u.m.Unlock()
}

func (u *UserInterface) getInitWindowPositionInDIP() (int, int) {
	u.m.RLock()
	defer u.m.RUnlock()
	if u.initWindowPositionXInDIP != invalidPos && u.initWindowPositionYInDIP != invalidPos {
		return u.initWindowPositionXInDIP, u.initWindowPositionYInDIP
	}
	return invalidPos, invalidPos
}

func (u *UserInterface) setInitWindowPositionInDIP(x, y int) {
	u.m.Lock()
	defer u.m.Unlock()

	u.initWindowPositionXInDIP = x
	u.initWindowPositionYInDIP = y
}

func (u *UserInterface) getInitWindowSizeInDIP() (int, int) {
	u.m.Lock()
	w, h := u.initWindowWidthInDIP, u.initWindowHeightInDIP
	u.m.Unlock()
	return w, h
}

func (u *UserInterface) setInitWindowSizeInDIP(width, height int) {
	u.m.Lock()
	u.initWindowWidthInDIP, u.initWindowHeightInDIP = width, height
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
	m := u.initWindowMaximized
	u.m.Unlock()
	return m
}

func (u *UserInterface) setInitWindowMaximized(maximized bool) {
	u.m.Lock()
	u.initWindowMaximized = maximized
	u.m.Unlock()
}

func (u *UserInterface) isWindowClosingHandled() bool {
	u.m.Lock()
	v := u.windowClosingHandled
	u.m.Unlock()
	return v
}

func (u *UserInterface) setWindowClosingHandled(handled bool) {
	u.m.Lock()
	u.windowClosingHandled = handled
	u.m.Unlock()
}

func (u *UserInterface) isWindowBeingClosed() bool {
	u.m.Lock()
	v := u.windowBeingClosed
	u.m.Unlock()
	return v
}

func (u *UserInterface) isInitFocused() bool {
	u.m.Lock()
	v := u.initFocused
	u.m.Unlock()
	return v
}

func (u *UserInterface) setInitFocused(focused bool) {
	u.m.Lock()
	u.initFocused = focused
	u.m.Unlock()
}

func (u *UserInterface) ScreenSizeInFullscreen() (int, int) {
	if !u.isRunning() {
		return u.initFullscreenWidthInDIP, u.initFullscreenHeightInDIP
	}

	var w, h int
	u.t.Call(func() {
		m := u.currentMonitor()
		v := m.GetVideoMode()
		w = int(u.dipFromGLFWMonitorPixel(float64(v.Width), m))
		h = int(u.dipFromGLFWMonitorPixel(float64(v.Height), m))
	})
	return w, h
}

// isFullscreen must be called from the main thread.
func (u *UserInterface) isFullscreen() bool {
	if !u.isRunning() {
		panic("glfw: isFullscreen can't be called before the main loop starts")
	}
	return u.window.GetMonitor() != nil || u.isNativeFullscreen()
}

func (u *UserInterface) IsFullscreen() bool {
	if !u.isRunning() {
		return u.isInitFullscreen()
	}
	b := false
	u.t.Call(func() {
		b = u.isFullscreen()
	})
	return b
}

func (u *UserInterface) SetFullscreen(fullscreen bool) {
	if !u.isRunning() {
		u.setInitFullscreen(fullscreen)
		return
	}

	var update bool
	u.t.Call(func() {
		update = u.isFullscreen() != fullscreen
	})
	if !update {
		return
	}

	u.t.Call(func() {
		w, h := u.windowWidthInDIP, u.windowHeightInDIP
		u.setWindowSizeInDIP(w, h, fullscreen)
	})
}

func (u *UserInterface) IsFocused() bool {
	if !u.isRunning() {
		return false
	}

	var focused bool
	u.t.Call(func() {
		focused = u.window.GetAttrib(glfw.Focused) == glfw.True
	})
	return focused
}

func (u *UserInterface) SetRunnableOnUnfocused(runnableOnUnfocused bool) {
	u.setRunnableOnUnfocused(runnableOnUnfocused)
}

func (u *UserInterface) IsRunnableOnUnfocused() bool {
	return u.isRunnableOnUnfocused()
}

func (u *UserInterface) SetFPSMode(mode driver.FPSMode) {
	if !u.isRunning() {
		u.m.Lock()
		u.fpsMode = mode
		u.m.Unlock()
		return
	}
	u.t.Call(func() {
		if !u.fpsModeInited {
			u.fpsMode = mode
			return
		}
		u.setFPSMode(mode)
		u.updateVsync()
	})
}

func (u *UserInterface) FPSMode() driver.FPSMode {
	if !u.isRunning() {
		u.m.Lock()
		m := u.fpsMode
		u.m.Unlock()
		return m
	}
	var v driver.FPSMode
	u.t.Call(func() {
		v = u.fpsMode
	})
	return v
}

func (u *UserInterface) ScheduleFrame() {
	if !u.isRunning() {
		return
	}
	// As the main thread can be blocked, do not check the current FPS mode.
	// PostEmptyEvent is concurrent safe.
	glfw.PostEmptyEvent()
}

func (u *UserInterface) CursorMode() driver.CursorMode {
	if !u.isRunning() {
		return u.getInitCursorMode()
	}

	var mode int
	u.t.Call(func() {
		mode = u.window.GetInputMode(glfw.CursorMode)
	})

	var v driver.CursorMode
	switch mode {
	case glfw.CursorNormal:
		v = driver.CursorModeVisible
	case glfw.CursorHidden:
		v = driver.CursorModeHidden
	case glfw.CursorDisabled:
		v = driver.CursorModeCaptured
	default:
		panic(fmt.Sprintf("glfw: invalid GLFW cursor mode: %d", mode))
	}
	return v
}

func (u *UserInterface) SetCursorMode(mode driver.CursorMode) {
	if !u.isRunning() {
		u.setInitCursorMode(mode)
		return
	}
	u.t.Call(func() {
		u.window.SetInputMode(glfw.CursorMode, driverCursorModeToGLFWCursorMode(mode))
	})
}

func (u *UserInterface) CursorShape() driver.CursorShape {
	return u.getCursorShape()
}

func (u *UserInterface) SetCursorShape(shape driver.CursorShape) {
	old := u.setCursorShape(shape)
	if old == shape {
		return
	}
	if !u.isRunning() {
		return
	}
	u.t.Call(func() {
		u.setNativeCursor(shape)
	})
}

func (u *UserInterface) DeviceScaleFactor() float64 {
	if !u.isRunning() {
		// TODO: Use the initWindowPosition. This requires to convert the units correctly (#1575).
		return u.deviceScaleFactor(u.currentMonitor())
	}

	f := 0.0
	u.t.Call(func() {
		f = u.deviceScaleFactor(u.currentMonitor())
	})
	return f
}

// deviceScaleFactor must be called from the main thread.
func (u *UserInterface) deviceScaleFactor(monitor *glfw.Monitor) float64 {
	// It is rare, but monitor can be nil when glfw.GetPrimaryMonitor returns nil.
	// In this case, return 1 as a tentative scale (#1878).
	if monitor == nil {
		return 1
	}

	mx, my := monitor.GetPos()
	return devicescale.GetAt(mx, my)
}

func init() {
	// Lock the main thread.
	runtime.LockOSThread()
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
	initializeWindowAfterCreation(window)
	u.window = window

	// Even just after a window creation, FramebufferSize callback might be invoked (#1847).
	// Ensure to consume this callback.
	u.waitForFramebufferSizeCallback(u.window, nil)

	if u.Graphics().IsGL() {
		u.window.MakeContextCurrent()
	}

	u.window.SetInputMode(glfw.CursorMode, driverCursorModeToGLFWCursorMode(u.getInitCursorMode()))
	u.window.SetCursor(glfwSystemCursors[u.getCursorShape()])
	u.window.SetTitle(u.title)
	// TODO: Set icons

	u.registerWindowSetSizeCallback()
	u.registerWindowCloseCallback()

	return nil
}

// registerWindowSetSizeCallback must be called from the main thread.
func (u *UserInterface) registerWindowSetSizeCallback() {
	if u.sizeCallback == 0 {
		u.sizeCallback = glfw.ToSizeCallback(func(_ *glfw.Window, width, height int) {
			if !u.setSizeCallbackEnabled {
				return
			}

			u.adjustViewSize()

			if u.window.GetAttrib(glfw.Resizable) == glfw.False {
				return
			}
			if u.isFullscreen() && !u.isNativeFullscreen() {
				return
			}

			if err := u.runOnAnotherThreadFromMainThread(func() error {
				// Disable Vsync temporarily. On macOS, getting a next frame can get stuck (#1740).
				u.Graphics().SetVsyncEnabled(false)

				var outsideWidth, outsideHeight float64

				u.t.Call(func() {
					if width != 0 || height != 0 {
						w := int(u.dipFromGLFWPixel(float64(width), u.currentMonitor()))
						h := int(u.dipFromGLFWPixel(float64(height), u.currentMonitor()))
						u.setWindowSizeInDIP(w, h, u.isFullscreen())
					}

					outsideWidth, outsideHeight = u.updateSize()
				})
				u.context.Layout(outsideWidth, outsideHeight)
				if err := u.context.ForceUpdateFrame(); err != nil {
					return err
				}
				if u.Graphics().IsGL() {
					u.t.Call(func() {
						u.swapBuffers()
					})
				}
				return nil
			}); err != nil {
				u.err = err
			}
		})
	}
	u.window.SetSizeCallback(u.sizeCallback)
}

// registerWindowCloseCallback must be called from the main thread.
func (u *UserInterface) registerWindowCloseCallback() {
	if u.closeCallback == 0 {
		u.closeCallback = glfw.ToCloseCallback(func(_ *glfw.Window) {
			u.m.Lock()
			u.windowBeingClosed = true
			u.m.Unlock()

			if !u.isWindowClosingHandled() {
				return
			}
			u.window.SetShouldClose(false)
		})
	}
	u.window.SetCloseCallback(u.closeCallback)
}

// waitForFramebufferSizeCallback waits for GLFW's FramebufferSize callback.
// f is a process executed after registering the callback.
// If the callback is not invoked for a while, waitForFramebufferSizeCallback times out and return.
//
// waitForFramebufferSizeCallback must be called from the main thread.
func (u *UserInterface) waitForFramebufferSizeCallback(window *glfw.Window, f func()) {
	u.framebufferSizeCallbackCh = make(chan struct{}, 1)

	if u.framebufferSizeCallback == 0 {
		u.framebufferSizeCallback = glfw.ToFramebufferSizeCallback(func(_ *glfw.Window, _, _ int) {
			// This callback can be invoked multiple times by one PollEvents in theory (#1618).
			// Allow the case when the channel is full.
			select {
			case u.framebufferSizeCallbackCh <- struct{}{}:
			default:
			}
		})
	}
	window.SetFramebufferSizeCallback(u.framebufferSizeCallback)

	if f != nil {
		f()
	}

	// Use the timeout as FramebufferSize event might not be fired (#1618).
	t := time.NewTimer(time.Second)
	defer t.Stop()

event:
	for {
		glfw.PollEvents()
		select {
		case <-u.framebufferSizeCallbackCh:
			break event
		case <-t.C:
			break event
		default:
			time.Sleep(time.Millisecond)
		}
	}
	window.SetFramebufferSizeCallback(glfw.ToFramebufferSizeCallback(nil))
	close(u.framebufferSizeCallbackCh)
	u.framebufferSizeCallbackCh = nil
}

func (u *UserInterface) init() error {
	if u.Graphics().IsGL() {
		glfw.WindowHint(glfw.ClientAPI, glfw.OpenGLAPI)
		glfw.WindowHint(glfw.ContextVersionMajor, 2)
		glfw.WindowHint(glfw.ContextVersionMinor, 1)
	} else {
		glfw.WindowHint(glfw.ClientAPI, glfw.NoAPI)
	}

	glfw.WindowHint(glfw.AutoIconify, glfw.False)

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

	focused := glfw.False
	if u.isInitFocused() {
		focused = glfw.True
	}
	glfw.WindowHint(glfw.FocusOnShow, focused)

	// Set the window visible explicitly or the application freezes on Wayland (#974).
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		glfw.WindowHint(glfw.Visible, glfw.True)
	}

	if err := u.createWindow(); err != nil {
		return err
	}

	u.setSizeCallbackEnabled = true

	setSize := func() {
		ww, wh := u.getInitWindowSizeInDIP()
		u.setWindowSizeInDIP(ww, wh, u.isFullscreen())
	}

	// Set the window size and the window position in this order on Linux or other UNIX using X (#1118),
	// but this should be inverted on Windows. This is very tricky, but there is no obvious way to solve
	// this. This doesn't matter on macOS.
	wx, wy := u.getInitWindowPositionInDIP()
	if runtime.GOOS == "windows" {
		u.setWindowPositionInDIP(wx, wy, u.initMonitor)
		setSize()
	} else {
		setSize()
		u.setWindowPositionInDIP(wx, wy, u.initMonitor)
	}

	u.updateWindowSizeLimits()

	// Maximizing a window requires a proper size and position. Call Maximize here (#1117).
	if u.isInitWindowMaximized() {
		u.window.Maximize()
	}

	u.window.SetTitle(u.title)
	u.window.Show()

	if g, ok := u.Graphics().(interface{ SetWindow(uintptr) }); ok {
		g.SetWindow(u.nativeWindow())
	}

	return nil
}

func (u *UserInterface) updateSize() (float64, float64) {
	ww, wh := u.windowWidthInDIP, u.windowHeightInDIP
	u.setWindowSizeInDIP(ww, wh, u.isFullscreen())

	var w, h float64
	if u.isFullscreen() && !u.isNativeFullscreen() {
		// On Linux, the window size is not reliable just after making the window
		// fullscreened. Use the monitor size.
		// On macOS's native fullscreen, the window's size returns a more precise size
		// reflecting the adjustment of the view size (#1745).
		m := u.currentMonitor()
		v := m.GetVideoMode()
		ww, wh := v.Width, v.Height
		w = u.dipFromGLFWMonitorPixel(float64(ww), m)
		h = u.dipFromGLFWMonitorPixel(float64(wh), m)
	} else {
		// Instead of u.windowWidthInDIP and u.windowHeightInDIP, use the actual window size
		// here. On Windows, the specified size at SetSize and the actual window size might
		// not match (#1163).
		ww, wh := u.window.GetSize()
		w = u.dipFromGLFWPixel(float64(ww), u.currentMonitor())
		h = u.dipFromGLFWPixel(float64(wh), u.currentMonitor())
	}

	return w, h
}

// setFPSMode must be called from the main thread.
func (u *UserInterface) setFPSMode(fpsMode driver.FPSMode) {
	needUpdate := u.fpsMode != fpsMode || !u.fpsModeInited
	u.fpsMode = fpsMode
	u.fpsModeInited = true

	if !needUpdate {
		return
	}

	sticky := glfw.True
	if fpsMode == driver.FPSModeVsyncOffMinimum {
		sticky = glfw.False
	}
	u.window.SetInputMode(glfw.StickyMouseButtonsMode, sticky)
	u.window.SetInputMode(glfw.StickyKeysMode, sticky)
}

// update must be called from the main thread.
func (u *UserInterface) update() (float64, float64, error) {
	if u.err != nil {
		return 0, 0, u.err
	}

	if u.window.ShouldClose() {
		return 0, 0, driver.RegularTermination
	}

	if u.isInitFullscreen() {
		w, h := u.window.GetSize()
		ww := int(u.dipFromGLFWPixel(float64(w), u.currentMonitor()))
		wh := int(u.dipFromGLFWPixel(float64(h), u.currentMonitor()))
		u.setWindowSizeInDIP(ww, wh, true)
		u.setInitFullscreen(false)
	}

	// Initialize vsync after SetMonitor is called. See the comment in updateVsync.
	// Calling this inside setWindowSize didn't work (#1363).
	if !u.fpsModeInited {
		u.setFPSMode(u.fpsMode)
	}

	// Call updateVsync even though fpsMode is not updated.
	// The vsync state might be changed in other places (e.g., the SetSizeCallback).
	// Also, when toggling to fullscreen, vsync state might be reset unexpectedly (#1787).
	u.updateVsync()

	outsideWidth, outsideHeight := u.updateSize()

	if u.fpsMode != driver.FPSModeVsyncOffMinimum {
		// TODO: Updating the input can be skipped when clock.Update returns 0 (#1367).
		glfw.PollEvents()
	} else {
		glfw.WaitEvents()
	}
	u.input.update(u.window, u.context)

	for !u.isRunnableOnUnfocused() && u.window.GetAttrib(glfw.Focused) == 0 && !u.window.ShouldClose() {
		if err := hooks.SuspendAudio(); err != nil {
			return 0, 0, err
		}
		// Wait for an arbitrary period to avoid busy loop.
		time.Sleep(time.Second / 60)
		glfw.PollEvents()
	}
	if err := hooks.ResumeAudio(); err != nil {
		return 0, 0, err
	}

	return outsideWidth, outsideHeight, nil
}

func (u *UserInterface) loop() error {
	defer u.t.Call(glfw.Terminate)

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

		var outsideWidth, outsideHeight float64
		var err error
		if u.t.Call(func() {
			outsideWidth, outsideHeight, err = u.update()
		}); err != nil {
			return err
		}
		u.context.Layout(outsideWidth, outsideHeight)

		if err := u.context.UpdateFrame(); err != nil {
			return err
		}

		// Create icon images in a different goroutine (#1478).
		// In the fullscreen mode, SetIcon fails (#1578).
		if imgs := u.getIconImages(); len(imgs) > 0 && !u.isFullscreen() {
			u.setIconImages(nil)

			// Convert the icons in the different goroutine, as (*ebiten.Image).At cannot be invoked
			// from this goroutine. At works only in between BeginFrame and EndFrame.
			go func() {
				newImgs := make([]image.Image, len(imgs))
				for i, img := range imgs {
					// TODO: If img is not *ebiten.Image, this converting is not necessary.
					// However, this package cannot refer *ebiten.Image due to the package
					// dependencies.

					b := img.Bounds()
					rgba := image.NewRGBA(b)
					for j := b.Min.Y; j < b.Max.Y; j++ {
						for i := b.Min.X; i < b.Max.X; i++ {
							rgba.Set(i, j, img.At(i, j))
						}
					}
					newImgs[i] = rgba
				}

				u.t.Call(func() {
					// In the fullscreen mode, reset the icon images and try again later.
					if u.isFullscreen() {
						u.setIconImages(imgs)
						return
					}
					u.window.SetIcon(newImgs)
				})
			}()
		}

		// swapBuffers also checks IsGL, so this condition is redundant.
		// However, (*thread).Call is not good for performance due to channels.
		// Let's avoid this whenever possible (#1367).
		if u.Graphics().IsGL() {
			u.t.Call(u.swapBuffers)
		}

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

// updateWindowSizeLimits must be called from the main thread.
func (u *UserInterface) updateWindowSizeLimits() {
	m := u.currentMonitor()
	minw, minh, maxw, maxh := u.getWindowSizeLimitsInDIP()
	if minw < 0 {
		minw = glfw.DontCare
	} else {
		minw = int(u.dipToGLFWPixel(float64(minw), m))
	}
	if minh < 0 {
		minh = glfw.DontCare
	} else {
		minh = int(u.dipToGLFWPixel(float64(minh), m))
	}
	if maxw < 0 {
		maxw = glfw.DontCare
	} else {
		maxw = int(u.dipToGLFWPixel(float64(maxw), m))
	}
	if maxh < 0 {
		maxh = glfw.DontCare
	} else {
		maxh = int(u.dipToGLFWPixel(float64(maxh), m))
	}
	u.window.SetSizeLimits(minw, minh, maxw, maxh)
}

// adjustWindowSizeBasedOnSizeLimitsInDIP adjust the size based on the window size limits.
// width and height are in device-independent pixels.
func (u *UserInterface) adjustWindowSizeBasedOnSizeLimitsInDIP(width, height int) (int, int) {
	minw, minh, maxw, maxh := u.getWindowSizeLimitsInDIP()
	if minw >= 0 && width < minw {
		width = minw
	}
	if minh >= 0 && height < minh {
		height = minh
	}
	if maxw >= 0 && width > maxw {
		width = maxw
	}
	if maxh >= 0 && height > maxh {
		height = maxh
	}
	return width, height
}

// setWindowSize must be called from the main thread.
func (u *UserInterface) setWindowSizeInDIP(width, height int, fullscreen bool) {
	width, height = u.adjustWindowSizeBasedOnSizeLimitsInDIP(width, height)

	u.Graphics().SetFullscreen(fullscreen)

	scale := u.deviceScaleFactor(u.currentMonitor())
	if u.windowWidthInDIP == width && u.windowHeightInDIP == height && u.isFullscreen() == fullscreen && u.lastDeviceScaleFactor == scale {
		return
	}

	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}

	u.lastDeviceScaleFactor = scale

	// To make sure the current existing framebuffers are rendered,
	// swap buffers here before SetSize is called.
	u.swapBuffers()

	// Disable the callback of SetSize. This callback can be invoked by SetMonitor or SetSize.
	// ForceUpdateFrame is called from the callback.
	// While setWindowSize can be called from UpdateFrame,
	// calling ForceUpdateFrame inside UpdateFrame is illegal (#1505).
	if u.setSizeCallbackEnabled {
		u.setSizeCallbackEnabled = false
		defer func() {
			u.setSizeCallbackEnabled = true
		}()
	}

	windowRecreated := u.setWindowSizeInDIPImpl(width, height, fullscreen)

	u.adjustViewSize()

	// As width might be updated, update windowWidth/Height here.
	u.windowWidthInDIP = width
	u.windowHeightInDIP = height

	if windowRecreated {
		if g, ok := u.Graphics().(interface{ SetWindow(uintptr) }); ok {
			g.SetWindow(u.nativeWindow())
		}
	}
}

func (u *UserInterface) setWindowSizeInDIPImpl(width, height int, fullscreen bool) bool {
	var windowRecreated bool

	if fullscreen {
		if x, y := u.origPos(); x == invalidPos || y == invalidPos {
			u.setOrigPos(u.window.GetPos())
		}

		if u.isNativeFullscreenAvailable() {
			u.setNativeFullscreen(fullscreen)
		} else {
			m := u.currentMonitor()
			v := m.GetVideoMode()
			u.window.SetMonitor(m, 0, 0, v.Width, v.Height, v.RefreshRate)

			// Swapping buffer is necesary to prevent the image lag (#1004).
			// TODO: This might not work when vsync is disabled.
			if u.Graphics().IsGL() {
				glfw.PollEvents()
				u.swapBuffers()
			}
		}
	} else {
		// On Windows, giving a too small width doesn't call a callback (#165).
		// To prevent hanging up, return asap if the width is too small.
		// 126 is an arbitrary number and I guess this is small enough.
		minWindowWidth := 126
		if u.window.GetAttrib(glfw.Decorated) == glfw.False {
			minWindowWidth = 1
		}
		if width < minWindowWidth {
			width = minWindowWidth
		}

		if u.isNativeFullscreenAvailable() && u.isNativeFullscreen() {
			u.setNativeFullscreen(false)
		} else if !u.isNativeFullscreenAvailable() && u.window.GetMonitor() != nil {
			if u.Graphics().IsGL() {
				// When OpenGL is used, swapping buffer is enough to solve the image-lag
				// issue (#1004). Rather, recreating window destroys GPU resources.
				// TODO: This might not work when vsync is disabled.
				ww := int(u.dipToGLFWPixel(float64(width), u.currentMonitor()))
				wh := int(u.dipToGLFWPixel(float64(height), u.currentMonitor()))
				u.window.SetMonitor(nil, 0, 0, ww, wh, 0)
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
				// Reset the size limits explicitly.
				u.updateWindowSizeLimits()
				u.window.Show()
				windowRecreated = true
			}
		}

		if x, y := u.origPos(); x != invalidPos && y != invalidPos {
			u.window.SetPos(x, y)
			// Dirty hack for macOS (#703). Rendering doesn't work correctly with one SetPos, but
			// work with two or more SetPos.
			if runtime.GOOS == "darwin" {
				u.window.SetPos(x+1, y)
				u.window.SetPos(x, y)
			}
			u.setOrigPos(invalidPos, invalidPos)
		}

		// Set the window size after the position. The order matters.
		// In the opposite order, the window size might not be correct when going back from fullscreen with multi monitors.
		oldW, oldH := u.window.GetSize()
		newW := int(u.dipToGLFWPixel(float64(width), u.currentMonitor()))
		newH := int(u.dipToGLFWPixel(float64(height), u.currentMonitor()))
		if oldW != newW || oldH != newH {
			// Just after SetSize, GetSize is not reliable especially on Linux/UNIX.
			// Let's wait for FramebufferSize callback in any cases.
			u.waitForFramebufferSizeCallback(u.window, func() {
				u.window.SetSize(newW, newH)
			})
		}

		// Window title might be lost on macOS after coming back from fullscreen.
		u.window.SetTitle(u.title)
	}

	return windowRecreated
}

// updateVsync must be called on the main thread.
func (u *UserInterface) updateVsync() {
	if u.Graphics().IsGL() {
		// SwapInterval is affected by the current monitor of the window.
		// This needs to be called at least after SetMonitor.
		// Without SwapInterval after SetMonitor, vsynch doesn't work (#375).
		//
		// TODO: (#405) If triple buffering is needed, SwapInterval(0) should be called,
		// but is this correct? If glfw.SwapInterval(0) and the driver doesn't support triple
		// buffering, what will happen?
		if u.fpsMode == driver.FPSModeVsyncOn {
			glfw.SwapInterval(1)
		} else {
			glfw.SwapInterval(0)
		}
	}
	u.Graphics().SetVsyncEnabled(u.fpsMode == driver.FPSModeVsyncOn)
}

// initialMonitor returns the initial monitor to show the window.
//
// The given window is just a hint and might not be used to determine the initial monitor.
//
// initialMonitor must be called on the main thread.
func initialMonitor(window *glfw.Window) *glfw.Monitor {
	if m := initialMonitorByOS(); m != nil {
		return m
	}
	return currentMonitorImpl(window)
}

// currentMonitor returns the current active monitor.
//
// currentMonitor must be called on the main thread.
func (u *UserInterface) currentMonitor() *glfw.Monitor {
	if !u.isRunning() {
		return u.initMonitor
	}
	return currentMonitorImpl(u.window)
}

// currentMonitorImpl returns the current active monitor.
//
// The given window might or might not be used to detect the monitor.
//
// currentMonitorImpl must be called on the main thread.
func currentMonitorImpl(window *glfw.Window) *glfw.Monitor {
	// GetMonitor is available only in fullscreen.
	if m := window.GetMonitor(); m != nil {
		return m
	}

	// Getting a monitor from a window position is not reliable in general (e.g., when a window is put across
	// multiple monitors, or, before SetWindowPosition is called.).
	// Get the monitor which the current window belongs to. This requires OS API.
	if m := currentMonitorByOS(window); m != nil {
		return m
	}

	// As the fallback, detect the monitor from the window.
	if m := getMonitorFromPosition(window.GetPos()); m != nil {
		return m.m
	}
	return glfw.GetPrimaryMonitor()
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
	u.t.Call(func() {
		val = u.window.GetAttrib(glfw.TransparentFramebuffer) == glfw.True
	})
	return val
}

func (u *UserInterface) ResetForFrame() {
	// The offscreens must be updated every frame (#490).
	var w, h float64
	u.t.Call(func() {
		w, h = u.updateSize()
	})
	u.context.Layout(w, h)
	u.input.resetForFrame()

	u.m.Lock()
	u.windowBeingClosed = false
	u.m.Unlock()
}

func (u *UserInterface) SetInitFocused(focused bool) {
	if u.isRunning() {
		panic("ui: SetInitFocused must be called before the main loop")
	}
	u.setInitFocused(focused)
}

func (u *UserInterface) Input() driver.Input {
	return &u.input
}

func (u *UserInterface) Window() driver.Window {
	return &u.iwindow
}

// GLFW's functions to manipulate a window can invoke the SetSize callback (#1576, #1585, #1606).
// As the callback must not be called in the frame (between BeginFrame and EndFrame),
// disable the callback temporarily.

// maximizeWindow must be called from the main thread.
func (u *UserInterface) maximizeWindow() {
	if u.isNativeFullscreen() {
		return
	}

	if u.setSizeCallbackEnabled {
		u.setSizeCallbackEnabled = false
		defer func() {
			u.setSizeCallbackEnabled = true
		}()
	}
	u.window.Maximize()

	if !u.isFullscreen() {
		// On Linux/UNIX, maximizing might not finish even though Maximize returns. Just wait for its finish.
		// Do not check this in the fullscreen since apparently the condition can never be true.
		for u.window.GetAttrib(glfw.Maximized) != glfw.True {
			glfw.PollEvents()
		}

		// Call setWindowSize explicitly in order to update the rendering since the callback is disabled now.
		// Do not call setWindowSize in the fullscreen mode since setWindowSize requires the window size
		// before the fullscreen, while window.GetSize() returns the desktop screen size in the fullscreen mode.
		w, h := u.window.GetSize()
		ww := int(u.dipFromGLFWPixel(float64(w), u.currentMonitor()))
		wh := int(u.dipFromGLFWPixel(float64(h), u.currentMonitor()))
		u.setWindowSizeInDIP(ww, wh, u.isFullscreen())
	}
}

// iconifyWindow must be called from the main thread.
func (u *UserInterface) iconifyWindow() {
	if u.isNativeFullscreen() {
		return
	}

	if u.setSizeCallbackEnabled {
		u.setSizeCallbackEnabled = false
		defer func() {
			u.setSizeCallbackEnabled = true
		}()
	}
	u.window.Iconify()

	// On Linux/UNIX, iconifying might not finish even though Iconify returns. Just wait for its finish.
	for u.window.GetAttrib(glfw.Iconified) != glfw.True {
		glfw.PollEvents()
	}

	// After iconifiying, the window is invisible and setWindowSize doesn't have to be called.
	// Rather, the window size might be (0, 0) and it might be impossible to call setWindowSize (#1585).
}

// restoreWindow must be called from the main thread.
func (u *UserInterface) restoreWindow() {
	if u.setSizeCallbackEnabled {
		u.setSizeCallbackEnabled = false
		defer func() {
			u.setSizeCallbackEnabled = true
		}()
	}

	u.window.Restore()

	// On Linux/UNIX, restoring might not finish even though Restore returns (#1608). Just wait for its finish.
	// On macOS, the restoring state might be the same as the maximized state. Skip this.
	if runtime.GOOS != "darwin" {
		for u.window.GetAttrib(glfw.Maximized) == glfw.True || u.window.GetAttrib(glfw.Iconified) == glfw.True {
			glfw.PollEvents()
		}
	}

	// Call setWindowSize explicitly in order to update the rendering since the callback is disabled now.
	// Do not call setWindowSize in the fullscreen mode since setWindowSize requires the window size
	// before the fullscreen, while window.GetSize() returns the desktop screen size in the fullscreen mode.
	if !u.isFullscreen() {
		w, h := u.window.GetSize()
		ww := int(u.dipFromGLFWPixel(float64(w), u.currentMonitor()))
		wh := int(u.dipFromGLFWPixel(float64(h), u.currentMonitor()))
		u.setWindowSizeInDIP(ww, wh, u.isFullscreen())
	}
}

// setWindowDecorated must be called from the main thread.
func (u *UserInterface) setWindowDecorated(decorated bool) {
	if u.setSizeCallbackEnabled {
		u.setSizeCallbackEnabled = false
		defer func() {
			u.setSizeCallbackEnabled = true
		}()
	}
	v := glfw.False
	if decorated {
		v = glfw.True
	}
	u.window.SetAttrib(glfw.Decorated, v)

	// The title can be lost when the decoration is gone. Recover this.
	if decorated {
		u.window.SetTitle(u.title)
	}
}

// setWindowFloating must be called from the main thread.
func (u *UserInterface) setWindowFloating(floating bool) {
	if u.setSizeCallbackEnabled {
		u.setSizeCallbackEnabled = false
		defer func() {
			u.setSizeCallbackEnabled = true
		}()
	}
	v := glfw.False
	if floating {
		v = glfw.True
	}
	u.window.SetAttrib(glfw.Floating, v)
}

// setWindowResizable must be called from the main thread.
func (u *UserInterface) setWindowResizable(resizable bool) {
	if u.setSizeCallbackEnabled {
		u.setSizeCallbackEnabled = false
		defer func() {
			u.setSizeCallbackEnabled = true
		}()
	}

	v := glfw.False
	if resizable {
		v = glfw.True
	}
	u.window.SetAttrib(glfw.Resizable, v)
}

// setWindowPositionInDIP sets the window position.
//
// x and y are the position in device-independent pixels.
//
// setWindowPositionInDIP must be called from the main thread.
func (u *UserInterface) setWindowPositionInDIP(x, y int, monitor *glfw.Monitor) {
	if u.setSizeCallbackEnabled {
		u.setSizeCallbackEnabled = false
		defer func() {
			u.setSizeCallbackEnabled = true
		}()
	}

	mx, my := monitor.GetPos()
	xf := u.dipToGLFWPixel(float64(x), monitor)
	yf := u.dipToGLFWPixel(float64(y), monitor)
	if x, y := u.adjustWindowPosition(mx+int(xf), my+int(yf)); u.isFullscreen() {
		u.setOrigPos(x, y)
	} else {
		u.window.SetPos(x, y)
	}

	// Call setWindowSize explicitly in order to update the rendering since the callback is disabled now.
	//
	// There are cases when setWindowSize should be called (#1606) and should not be called (#1609).
	// For the former, macOS seems enough so far.
	//
	// Do not call setWindowSize in the fullscreen mode since setWindowSize requires the window size
	// before the fullscreen, while window.GetSize() returns the desktop screen size in the fullscreen mode.
	if !u.isFullscreen() && runtime.GOOS == "darwin" {
		w, h := u.window.GetSize()
		ww := int(u.dipFromGLFWPixel(float64(w), u.currentMonitor()))
		wh := int(u.dipFromGLFWPixel(float64(h), u.currentMonitor()))
		u.setWindowSizeInDIP(ww, wh, u.isFullscreen())
	}
}

// setWindowTitle must be called from the main thread.
func (u *UserInterface) setWindowTitle(title string) {
	if u.setSizeCallbackEnabled {
		u.setSizeCallbackEnabled = false
		defer func() {
			u.setSizeCallbackEnabled = true
		}()
	}

	u.window.SetTitle(title)
}

func (u *UserInterface) origPos() (int, int) {
	// On macOS, the window can be fullscreened without calling an Ebiten function.
	// Then, an original position might not be available by u.window.GetPos().
	// Do not rely on the window position.
	if u.isNativeFullscreenAvailable() {
		return invalidPos, invalidPos
	}
	return u.origPosX, u.origPosY
}

func (u *UserInterface) setOrigPos(x, y int) {
	// TODO: The original position should be updated at a 'PosCallback'.

	// On macOS, the window can be fullscreened without calling an Ebiten function.
	// Then, an original position might not be available by u.window.GetPos().
	// Do not rely on the window position.
	if u.isNativeFullscreenAvailable() {
		return
	}
	u.origPosX = x
	u.origPosY = y
}

func (u *UserInterface) Vibrate(duration time.Duration, magnitude float64) {
	// Do nothing.
}
