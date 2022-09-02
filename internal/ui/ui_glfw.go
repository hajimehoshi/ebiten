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

//go:build !android && !ios && !js && !nintendosdk
// +build !android,!ios,!js,!nintendosdk

package ui

import (
	"errors"
	"fmt"
	"image"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hajimehoshi/ebiten/v2/internal/devicescale"
	"github.com/hajimehoshi/ebiten/v2/internal/gamepad"
	"github.com/hajimehoshi/ebiten/v2/internal/glfw"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/hooks"
	"github.com/hajimehoshi/ebiten/v2/internal/microsoftgdk"
	"github.com/hajimehoshi/ebiten/v2/internal/thread"
)

func driverCursorModeToGLFWCursorMode(mode CursorMode) int {
	switch mode {
	case CursorModeVisible:
		return glfw.CursorNormal
	case CursorModeHidden:
		return glfw.CursorHidden
	case CursorModeCaptured:
		return glfw.CursorDisabled
	default:
		panic(fmt.Sprintf("ui: invalid CursorMode: %d", mode))
	}
}

type userInterfaceImpl struct {
	graphicsDriver graphicsdriver.Graphics

	context *context
	title   string
	window  *glfw.Window

	minWindowWidthInDIP  int
	minWindowHeightInDIP int
	maxWindowWidthInDIP  int
	maxWindowHeightInDIP int

	running              uint32
	runnableOnUnfocused  bool
	fpsMode              FPSModeType
	iconImages           []image.Image
	cursorShape          CursorShape
	windowClosingHandled bool
	windowBeingClosed    bool
	windowResizingMode   WindowResizingMode
	justAfterResized     bool

	// setSizeCallbackEnabled must be accessed from the main thread.
	setSizeCallbackEnabled bool

	// err must be accessed from the main thread.
	err error

	lastDeviceScaleFactor float64

	// These values are not changed after initialized.
	// TODO: the fullscreen size should be updated when the initial window position is changed?
	initMonitor               *glfw.Monitor
	initDeviceScaleFactor     float64
	initFullscreenWidthInDIP  int
	initFullscreenHeightInDIP int

	initFullscreen           bool
	initCursorMode           CursorMode
	initWindowDecorated      bool
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
	iwindow glfwWindow

	sizeCallback                   glfw.SizeCallback
	closeCallback                  glfw.CloseCallback
	framebufferSizeCallback        glfw.FramebufferSizeCallback
	defaultFramebufferSizeCallback glfw.FramebufferSizeCallback
	framebufferSizeCallbackCh      chan struct{}

	// t is the main thread == the rendering thread.
	t thread.Thread
	m sync.RWMutex

	native userInterfaceImplNative
}

const (
	maxInt     = int(^uint(0) >> 1)
	minInt     = -maxInt - 1
	invalidPos = minInt
)

func init() {
	theUI.userInterfaceImpl = userInterfaceImpl{
		runnableOnUnfocused:      true,
		minWindowWidthInDIP:      glfw.DontCare,
		minWindowHeightInDIP:     glfw.DontCare,
		maxWindowWidthInDIP:      glfw.DontCare,
		maxWindowHeightInDIP:     glfw.DontCare,
		initCursorMode:           CursorModeVisible,
		initWindowDecorated:      true,
		initWindowPositionXInDIP: invalidPos,
		initWindowPositionYInDIP: invalidPos,
		initWindowWidthInDIP:     640,
		initWindowHeightInDIP:    480,
		initFocused:              true,
		fpsMode:                  FPSModeVsyncOn,
	}
	theUI.native.initialize()
	theUI.input.ui = &theUI.userInterfaceImpl
	theUI.iwindow.ui = &theUI.userInterfaceImpl
}

func init() {
	hideConsoleWindowOnWindows()
	if err := initialize(); err != nil {
		panic(err)
	}
	glfw.SetMonitorCallback(glfw.ToMonitorCallback(func(monitor *glfw.Monitor, event glfw.PeripheralEvent) {
		updateMonitors()
	}))
	updateMonitors()
}

var glfwSystemCursors = map[CursorShape]*glfw.Cursor{}

func initialize() error {
	if err := glfw.Init(); err != nil {
		return err
	}

	glfw.WindowHint(glfw.Visible, glfw.False)
	glfw.WindowHint(glfw.ClientAPI, glfw.NoAPI)

	m, err := initialMonitorByOS()
	if err != nil {
		return err
	}
	if m == nil {
		m = glfw.GetPrimaryMonitor()
	}

	// GetPrimaryMonitor might return nil in theory (#1887).
	if m == nil {
		return errors.New("ui: no monitor was found at initialize")
	}

	theUI.initMonitor = m
	theUI.initDeviceScaleFactor = theUI.deviceScaleFactor(m)
	// GetVideoMode must be called from the main thread, then call this here and record
	// initFullscreen{Width,Height}InDIP.
	v := m.GetVideoMode()
	theUI.initFullscreenWidthInDIP = int(theUI.dipFromGLFWMonitorPixel(float64(v.Width), m))
	theUI.initFullscreenHeightInDIP = int(theUI.dipFromGLFWMonitorPixel(float64(v.Height), m))

	// Create system cursors. These cursors are destroyed at glfw.Terminate().
	glfwSystemCursors[CursorShapeDefault] = nil
	glfwSystemCursors[CursorShapeText] = glfw.CreateStandardCursor(glfw.IBeamCursor)
	glfwSystemCursors[CursorShapeCrosshair] = glfw.CreateStandardCursor(glfw.CrosshairCursor)
	glfwSystemCursors[CursorShapePointer] = glfw.CreateStandardCursor(glfw.HandCursor)
	glfwSystemCursors[CursorShapeEWResize] = glfw.CreateStandardCursor(glfw.HResizeCursor)
	glfwSystemCursors[CursorShapeNSResize] = glfw.CreateStandardCursor(glfw.VResizeCursor)

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

func (u *userInterfaceImpl) isRunning() bool {
	return atomic.LoadUint32(&u.running) != 0
}

func (u *userInterfaceImpl) setRunning(running bool) {
	if running {
		atomic.StoreUint32(&u.running, 1)
	} else {
		atomic.StoreUint32(&u.running, 0)
	}
}

func (u *userInterfaceImpl) getWindowSizeLimitsInDIP() (minw, minh, maxw, maxh int) {
	if microsoftgdk.IsXbox() {
		return glfw.DontCare, glfw.DontCare, glfw.DontCare, glfw.DontCare
	}

	u.m.RLock()
	defer u.m.RUnlock()
	return u.minWindowWidthInDIP, u.minWindowHeightInDIP, u.maxWindowWidthInDIP, u.maxWindowHeightInDIP
}

func (u *userInterfaceImpl) setWindowSizeLimitsInDIP(minw, minh, maxw, maxh int) bool {
	if microsoftgdk.IsXbox() {
		// Do nothing. The size is always fixed.
		return false
	}

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

func (u *userInterfaceImpl) areWindowSizeLimitsSpecified() bool {
	minw, minh, maxw, maxh := u.getWindowSizeLimitsInDIP()
	return minw != glfw.DontCare || minh != glfw.DontCare || maxw != glfw.DontCare || maxh != glfw.DontCare
}

func (u *userInterfaceImpl) isInitFullscreen() bool {
	u.m.RLock()
	v := u.initFullscreen
	u.m.RUnlock()
	return v
}

func (u *userInterfaceImpl) setInitFullscreen(initFullscreen bool) {
	u.m.Lock()
	u.initFullscreen = initFullscreen
	u.m.Unlock()
}

func (u *userInterfaceImpl) getInitCursorMode() CursorMode {
	u.m.RLock()
	v := u.initCursorMode
	u.m.RUnlock()
	return v
}

func (u *userInterfaceImpl) setInitCursorMode(mode CursorMode) {
	u.m.Lock()
	u.initCursorMode = mode
	u.m.Unlock()
}

func (u *userInterfaceImpl) getCursorShape() CursorShape {
	u.m.RLock()
	v := u.cursorShape
	u.m.RUnlock()
	return v
}

func (u *userInterfaceImpl) setCursorShape(shape CursorShape) CursorShape {
	u.m.Lock()
	old := u.cursorShape
	u.cursorShape = shape
	u.m.Unlock()
	return old
}

func (u *userInterfaceImpl) isInitWindowDecorated() bool {
	u.m.RLock()
	v := u.initWindowDecorated
	u.m.RUnlock()
	return v
}

func (u *userInterfaceImpl) setInitWindowDecorated(decorated bool) {
	u.m.Lock()
	u.initWindowDecorated = decorated
	u.m.Unlock()
}

func (u *userInterfaceImpl) isRunnableOnUnfocused() bool {
	u.m.RLock()
	v := u.runnableOnUnfocused
	u.m.RUnlock()
	return v
}

func (u *userInterfaceImpl) setRunnableOnUnfocused(runnableOnUnfocused bool) {
	u.m.Lock()
	u.runnableOnUnfocused = runnableOnUnfocused
	u.m.Unlock()
}

func (u *userInterfaceImpl) isInitScreenTransparent() bool {
	u.m.RLock()
	v := u.initScreenTransparent
	u.m.RUnlock()
	return v
}

func (u *userInterfaceImpl) setInitScreenTransparent(transparent bool) {
	u.m.Lock()
	u.initScreenTransparent = transparent
	u.m.Unlock()
}

func (u *userInterfaceImpl) getIconImages() []image.Image {
	u.m.RLock()
	i := u.iconImages
	u.m.RUnlock()
	return i
}

func (u *userInterfaceImpl) setIconImages(iconImages []image.Image) {
	u.m.Lock()
	u.iconImages = iconImages
	u.m.Unlock()
}

func (u *userInterfaceImpl) getInitWindowPositionInDIP() (int, int) {
	if microsoftgdk.IsXbox() {
		return 0, 0
	}

	u.m.RLock()
	defer u.m.RUnlock()
	if u.initWindowPositionXInDIP != invalidPos && u.initWindowPositionYInDIP != invalidPos {
		return u.initWindowPositionXInDIP, u.initWindowPositionYInDIP
	}
	return invalidPos, invalidPos
}

func (u *userInterfaceImpl) setInitWindowPositionInDIP(x, y int) {
	if microsoftgdk.IsXbox() {
		return
	}

	u.m.Lock()
	defer u.m.Unlock()

	// TODO: Update initMonitor if necessary (#1575).
	u.initWindowPositionXInDIP = x
	u.initWindowPositionYInDIP = y
}

func (u *userInterfaceImpl) getInitWindowSizeInDIP() (int, int) {
	if microsoftgdk.IsXbox() {
		return microsoftgdk.MonitorResolution()
	}

	u.m.Lock()
	w, h := u.initWindowWidthInDIP, u.initWindowHeightInDIP
	u.m.Unlock()
	return w, h
}

func (u *userInterfaceImpl) setInitWindowSizeInDIP(width, height int) {
	if microsoftgdk.IsXbox() {
		return
	}

	u.m.Lock()
	u.initWindowWidthInDIP, u.initWindowHeightInDIP = width, height
	u.m.Unlock()
}

func (u *userInterfaceImpl) isInitWindowFloating() bool {
	if microsoftgdk.IsXbox() {
		return false
	}

	u.m.RLock()
	f := u.initWindowFloating
	u.m.RUnlock()
	return f
}

func (u *userInterfaceImpl) setInitWindowFloating(floating bool) {
	if microsoftgdk.IsXbox() {
		return
	}

	u.m.Lock()
	u.initWindowFloating = floating
	u.m.Unlock()
}

func (u *userInterfaceImpl) isInitWindowMaximized() bool {
	// TODO: Is this always true on Xbox?
	u.m.RLock()
	m := u.initWindowMaximized
	u.m.RUnlock()
	return m
}

func (u *userInterfaceImpl) setInitWindowMaximized(maximized bool) {
	u.m.Lock()
	u.initWindowMaximized = maximized
	u.m.Unlock()
}

func (u *userInterfaceImpl) isWindowClosingHandled() bool {
	u.m.RLock()
	v := u.windowClosingHandled
	u.m.RUnlock()
	return v
}

func (u *userInterfaceImpl) setWindowClosingHandled(handled bool) {
	u.m.Lock()
	u.windowClosingHandled = handled
	u.m.Unlock()
}

func (u *userInterfaceImpl) isWindowBeingClosed() bool {
	u.m.RLock()
	v := u.windowBeingClosed
	u.m.RUnlock()
	return v
}

func (u *userInterfaceImpl) isInitFocused() bool {
	if microsoftgdk.IsXbox() {
		return true
	}

	u.m.RLock()
	v := u.initFocused
	u.m.RUnlock()
	return v
}

func (u *userInterfaceImpl) setInitFocused(focused bool) {
	if microsoftgdk.IsXbox() {
		return
	}

	u.m.Lock()
	u.initFocused = focused
	u.m.Unlock()
}

func (u *userInterfaceImpl) ScreenSizeInFullscreen() (int, int) {
	if !u.isRunning() {
		return u.initFullscreenWidthInDIP, u.initFullscreenHeightInDIP
	}

	var w, h int
	u.t.Call(func() {
		m := u.currentMonitor()
		if m == nil {
			return
		}
		v := m.GetVideoMode()
		w = int(u.dipFromGLFWMonitorPixel(float64(v.Width), m))
		h = int(u.dipFromGLFWMonitorPixel(float64(v.Height), m))
	})
	return w, h
}

// isFullscreen must be called from the main thread.
func (u *userInterfaceImpl) isFullscreen() bool {
	if !u.isRunning() {
		panic("ui: isFullscreen can't be called before the main loop starts")
	}
	return u.window.GetMonitor() != nil || u.isNativeFullscreen()
}

func (u *userInterfaceImpl) IsFullscreen() bool {
	if !u.isRunning() {
		return u.isInitFullscreen()
	}
	b := false
	u.t.Call(func() {
		b = u.isFullscreen()
	})
	return b
}

func (u *userInterfaceImpl) SetFullscreen(fullscreen bool) {
	if !u.isRunning() {
		u.setInitFullscreen(fullscreen)
		return
	}

	u.t.Call(func() {
		if u.isFullscreen() == fullscreen {
			return
		}
		w, h := u.origWindowSizeInDIP()
		u.setWindowSizeInDIP(w, h, fullscreen)
	})
}

func (u *userInterfaceImpl) IsFocused() bool {
	if !u.isRunning() {
		return false
	}

	var focused bool
	u.t.Call(func() {
		focused = u.window.GetAttrib(glfw.Focused) == glfw.True
	})
	return focused
}

func (u *userInterfaceImpl) SetRunnableOnUnfocused(runnableOnUnfocused bool) {
	u.setRunnableOnUnfocused(runnableOnUnfocused)
}

func (u *userInterfaceImpl) IsRunnableOnUnfocused() bool {
	return u.isRunnableOnUnfocused()
}

func (u *userInterfaceImpl) SetFPSMode(mode FPSModeType) {
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

func (u *userInterfaceImpl) ScheduleFrame() {
	if !u.isRunning() {
		return
	}
	// As the main thread can be blocked, do not check the current FPS mode.
	// PostEmptyEvent is concurrent safe.
	glfw.PostEmptyEvent()
}

func (u *userInterfaceImpl) CursorMode() CursorMode {
	if !u.isRunning() {
		return u.getInitCursorMode()
	}

	var mode int
	u.t.Call(func() {
		mode = u.window.GetInputMode(glfw.CursorMode)
	})

	var v CursorMode
	switch mode {
	case glfw.CursorNormal:
		v = CursorModeVisible
	case glfw.CursorHidden:
		v = CursorModeHidden
	case glfw.CursorDisabled:
		v = CursorModeCaptured
	default:
		panic(fmt.Sprintf("ui: invalid GLFW cursor mode: %d", mode))
	}
	return v
}

func (u *userInterfaceImpl) SetCursorMode(mode CursorMode) {
	if !u.isRunning() {
		u.setInitCursorMode(mode)
		return
	}
	u.t.Call(func() {
		u.window.SetInputMode(glfw.CursorMode, driverCursorModeToGLFWCursorMode(mode))
	})
}

func (u *userInterfaceImpl) CursorShape() CursorShape {
	return u.getCursorShape()
}

func (u *userInterfaceImpl) SetCursorShape(shape CursorShape) {
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

func (u *userInterfaceImpl) DeviceScaleFactor() float64 {
	if !u.isRunning() {
		return u.initDeviceScaleFactor
	}

	f := 0.0
	u.t.Call(func() {
		f = u.deviceScaleFactor(u.currentMonitor())
	})
	return f
}

// deviceScaleFactor must be called from the main thread.
func (u *userInterfaceImpl) deviceScaleFactor(monitor *glfw.Monitor) float64 {
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

// createWindow creates a GLFW window.
//
// width and height are in GLFW pixels (not device-independent pixels).
//
// createWindow must be called from the main thread.
//
// createWindow does not set the position or size so far.
func (u *userInterfaceImpl) createWindow(width, height int) error {
	if u.window != nil {
		panic("ui: u.window must not exist at createWindow")
	}

	// As a start, create a window with temporary size to create OpenGL context thread.
	window, err := glfw.CreateWindow(width, height, "", nil, nil)
	if err != nil {
		return err
	}
	initializeWindowAfterCreation(window)
	u.window = window

	// Even just after a window creation, FramebufferSize callback might be invoked (#1847).
	// Ensure to consume this callback.
	u.waitForFramebufferSizeCallback(u.window, nil)

	if u.graphicsDriver.IsGL() {
		u.window.MakeContextCurrent()
	}

	u.window.SetInputMode(glfw.CursorMode, driverCursorModeToGLFWCursorMode(u.getInitCursorMode()))
	u.window.SetCursor(glfwSystemCursors[u.getCursorShape()])
	u.window.SetTitle(u.title)
	// Icons are set after every frame. They don't have to be cared here.

	u.updateWindowSizeLimits()

	return nil
}

// registerWindowSetSizeCallback must be called from the main thread.
func (u *userInterfaceImpl) registerWindowSetSizeCallback() {
	if u.sizeCallback == nil {
		u.sizeCallback = glfw.ToSizeCallback(func(_ *glfw.Window, width, height int) {
			if !u.setSizeCallbackEnabled {
				return
			}

			u.adjustViewSize()

			if u.window.GetAttrib(glfw.Resizable) == glfw.False {
				return
			}
			if u.isFullscreen() {
				return
			}

			if width != 0 || height != 0 {
				w := int(u.dipFromGLFWPixel(float64(width), u.currentMonitor()))
				h := int(u.dipFromGLFWPixel(float64(height), u.currentMonitor()))
				u.setWindowSizeInDIP(w, h, u.isFullscreen())
			}

			u.updateSize()
			outsideWidth, outsideHeight := u.outsideSize()
			deviceScaleFactor := u.deviceScaleFactor(u.currentMonitor())

			// In the game's update, u.t.Call might be called.
			// In order to call it safely, use runOnAnotherThreadFromMainThread.
			var err error
			u.runOnAnotherThreadFromMainThread(func() {
				err = u.context.forceUpdateFrame(u.graphicsDriver, outsideWidth, outsideHeight, deviceScaleFactor)
			})
			if err != nil {
				u.err = err
			}

			if u.graphicsDriver.IsGL() {
				u.swapBuffers()
			}

			u.forceToRefreshIfNeeded()
			u.justAfterResized = true
		})
	}
	u.window.SetSizeCallback(u.sizeCallback)
}

// registerWindowCloseCallback must be called from the main thread.
func (u *userInterfaceImpl) registerWindowCloseCallback() {
	if u.closeCallback == nil {
		u.closeCallback = glfw.ToCloseCallback(func(_ *glfw.Window) {
			u.m.Lock()
			u.windowBeingClosed = true
			u.m.Unlock()

			if !u.isWindowClosingHandled() {
				return
			}
			u.window.Focus()
			u.window.SetShouldClose(false)
		})
	}
	u.window.SetCloseCallback(u.closeCallback)
}

// registerWindowFramebufferSizeCallback must be called from the main thread.
func (u *userInterfaceImpl) registerWindowFramebufferSizeCallback() {
	if u.defaultFramebufferSizeCallback == nil && runtime.GOOS != "darwin" {
		// When the window gets resized (either by manual window resize or a window
		// manager), glfw sends a framebuffer size callback which we need to handle (#1960).
		// This event is the only way to handle the size change at least on i3 window manager.
		//
		// When a decorating state changes, the callback of arguments might be an unexpected value on macOS (#2257)
		// Then, do not register this callback on macOS.
		u.defaultFramebufferSizeCallback = glfw.ToFramebufferSizeCallback(func(_ *glfw.Window, w, h int) {
			if u.isFullscreen() {
				return
			}
			if u.window.GetAttrib(glfw.Iconified) == glfw.True {
				return
			}

			// The framebuffer size is always scaled by the device scale factor (#1975).
			// See also the implementation in uiContext.updateOffscreen.
			s := u.deviceScaleFactor(u.currentMonitor())
			ww := int(float64(w) / s)
			wh := int(float64(h) / s)
			u.setWindowSizeInDIP(ww, wh, u.isFullscreen())
		})
	}
	u.window.SetFramebufferSizeCallback(u.defaultFramebufferSizeCallback)
}

// waitForFramebufferSizeCallback waits for GLFW's FramebufferSize callback.
// f is a process executed after registering the callback.
// If the callback is not invoked for a while, waitForFramebufferSizeCallback times out and return.
//
// waitForFramebufferSizeCallback must be called from the main thread.
func (u *userInterfaceImpl) waitForFramebufferSizeCallback(window *glfw.Window, f func()) {
	u.framebufferSizeCallbackCh = make(chan struct{}, 1)

	if u.framebufferSizeCallback == nil {
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
	t := time.NewTimer(100 * time.Millisecond)
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
	window.SetFramebufferSizeCallback(u.defaultFramebufferSizeCallback)

	close(u.framebufferSizeCallbackCh)
	u.framebufferSizeCallbackCh = nil
}

func (u *userInterfaceImpl) init() error {
	glfw.WindowHint(glfw.AutoIconify, glfw.False)

	decorated := glfw.False
	if u.isInitWindowDecorated() {
		decorated = glfw.True
	}
	glfw.WindowHint(glfw.Decorated, decorated)

	transparent := u.isInitScreenTransparent()
	glfwTransparent := glfw.False
	if transparent {
		glfwTransparent = glfw.True
	}
	glfw.WindowHint(glfw.TransparentFramebuffer, glfwTransparent)

	g, err := newGraphicsDriver(&graphicsDriverCreatorImpl{
		transparent: transparent,
	})
	if err != nil {
		return err
	}
	u.graphicsDriver = g
	u.graphicsDriver.SetTransparent(u.isInitScreenTransparent())

	if u.graphicsDriver.IsGL() {
		glfw.WindowHint(glfw.ClientAPI, glfw.OpenGLAPI)
		glfw.WindowHint(glfw.ContextVersionMajor, 2)
		glfw.WindowHint(glfw.ContextVersionMinor, 1)
	} else {
		glfw.WindowHint(glfw.ClientAPI, glfw.NoAPI)
	}

	// Before creating a window, set it unresizable no matter what u.isInitWindowResizable() is (#1987).
	// Making the window resizable here doesn't work correctly when switching to enable resizing.
	resizable := glfw.False
	if u.windowResizingMode == WindowResizingModeEnabled {
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

	ww, wh := u.getInitWindowSizeInDIP()
	initW := int(u.dipToGLFWPixel(float64(ww), u.initMonitor))
	initH := int(u.dipToGLFWPixel(float64(wh), u.initMonitor))
	if err := u.createWindow(initW, initH); err != nil {
		return err
	}

	u.setSizeCallbackEnabled = true

	// The position must be set before the size is set (#1982).
	// setWindowSize refers the current monitor's device scale.
	// TODO: currentMonitor is very hard to use correctly. Refactor this.
	wx, wy := u.getInitWindowPositionInDIP()
	// Force to put the window in the initial monitor (#1575).
	if wx < 0 {
		wx = 0
	}
	if wy < 0 {
		wy = 0
	}
	if max := u.initFullscreenWidthInDIP - ww; wx >= max {
		wx = max
	}
	if max := u.initFullscreenHeightInDIP - wh; wy >= max {
		wy = max
	}
	u.setWindowPositionInDIP(wx, wy, u.initMonitor)
	u.setWindowSizeInDIP(ww, wh, u.isFullscreen())

	// Maximizing a window requires a proper size and position. Call Maximize here (#1117).
	if u.isInitWindowMaximized() {
		u.window.Maximize()
	}

	u.setWindowResizingModeForOS(u.windowResizingMode)

	u.window.Show()

	if g, ok := u.graphicsDriver.(interface{ SetWindow(uintptr) }); ok {
		g.SetWindow(u.nativeWindow())
	}

	gamepad.SetNativeWindow(u.nativeWindow())

	// Register callbacks after the window initialization done.
	// The callback might cause swapping frames, that assumes the window is already set (#2137).
	u.registerWindowSetSizeCallback()
	u.registerWindowCloseCallback()
	u.registerWindowFramebufferSizeCallback()

	return nil
}

func (u *userInterfaceImpl) updateSize() {
	ww, wh := u.origWindowSizeInDIP()
	u.setWindowSizeInDIP(ww, wh, u.isFullscreen())
}

func (u *userInterfaceImpl) outsideSize() (float64, float64) {
	if u.isFullscreen() && !u.isNativeFullscreen() {
		// On Linux, the window size is not reliable just after making the window
		// fullscreened. Use the monitor size.
		// On macOS's native fullscreen, the window's size returns a more precise size
		// reflecting the adjustment of the view size (#1745).
		var w, h float64
		if m := u.currentMonitor(); m != nil {
			v := m.GetVideoMode()
			ww, wh := v.Width, v.Height
			w = u.dipFromGLFWMonitorPixel(float64(ww), m)
			h = u.dipFromGLFWMonitorPixel(float64(wh), m)
		}
		return w, h
	}

	if u.window.GetAttrib(glfw.Iconified) == glfw.True {
		w, h := u.origWindowSizeInDIP()
		return float64(w), float64(h)
	}

	// Instead of u.origWindowSizeInDIP(), use the actual window size here.
	// On Windows, the specified size at SetSize and the actual window size might
	// not match (#1163).
	ww, wh := u.window.GetSize()
	w := u.dipFromGLFWPixel(float64(ww), u.currentMonitor())
	h := u.dipFromGLFWPixel(float64(wh), u.currentMonitor())
	return w, h
}

// setFPSMode must be called from the main thread.
func (u *userInterfaceImpl) setFPSMode(fpsMode FPSModeType) {
	needUpdate := u.fpsMode != fpsMode || !u.fpsModeInited
	u.fpsMode = fpsMode
	u.fpsModeInited = true

	if !needUpdate {
		return
	}

	sticky := glfw.True
	if fpsMode == FPSModeVsyncOffMinimum {
		sticky = glfw.False
	}
	u.window.SetInputMode(glfw.StickyMouseButtonsMode, sticky)
	u.window.SetInputMode(glfw.StickyKeysMode, sticky)
}

// update must be called from the main thread.
func (u *userInterfaceImpl) update() (float64, float64, error) {
	if u.err != nil {
		return 0, 0, u.err
	}

	if u.window.ShouldClose() {
		return 0, 0, RegularTermination
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

	if u.justAfterResized {
		u.forceToRefreshIfNeeded()
	}
	u.justAfterResized = false

	// Call updateVsync even though fpsMode is not updated.
	// The vsync state might be changed in other places (e.g., the SetSizeCallback).
	// Also, when toggling to fullscreen, vsync state might be reset unexpectedly (#1787).
	u.updateVsync()
	u.updateSize()

	if u.fpsMode != FPSModeVsyncOffMinimum {
		// TODO: Updating the input can be skipped when clock.Update returns 0 (#1367).
		glfw.PollEvents()
	} else {
		glfw.WaitEvents()
	}
	if err := u.input.update(u.window, u.context); err != nil {
		return 0, 0, err
	}

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

	outsideWidth, outsideHeight := u.outsideSize()
	return outsideWidth, outsideHeight, nil
}

func (u *userInterfaceImpl) loop() error {
	defer u.t.Call(func() {
		u.window.Destroy()
		glfw.Terminate()
	})

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
		var deviceScaleFactor float64
		var err error
		if u.t.Call(func() {
			outsideWidth, outsideHeight, err = u.update()
			deviceScaleFactor = u.deviceScaleFactor(u.currentMonitor())
		}); err != nil {
			return err
		}

		if err := u.context.updateFrame(u.graphicsDriver, outsideWidth, outsideHeight, deviceScaleFactor); err != nil {
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
		if u.graphicsDriver.IsGL() {
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
func (u *userInterfaceImpl) swapBuffers() {
	if u.graphicsDriver.IsGL() {
		u.window.SwapBuffers()
	}
}

// updateWindowSizeLimits must be called from the main thread.
func (u *userInterfaceImpl) updateWindowSizeLimits() {
	m := u.currentMonitor()
	minw, minh, maxw, maxh := u.getWindowSizeLimitsInDIP()

	if minw < 0 {
		// Always set the minimum window width.
		minw = int(u.dipToGLFWPixel(float64(u.minimumWindowWidth()), m))
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
func (u *userInterfaceImpl) adjustWindowSizeBasedOnSizeLimitsInDIP(width, height int) (int, int) {
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
func (u *userInterfaceImpl) setWindowSizeInDIP(width, height int, fullscreen bool) {
	if microsoftgdk.IsXbox() {
		// Do nothing. The size is always fixed.
		return
	}

	width, height = u.adjustWindowSizeBasedOnSizeLimitsInDIP(width, height)

	u.graphicsDriver.SetFullscreen(fullscreen)

	scale := u.deviceScaleFactor(u.currentMonitor())
	if ow, oh := u.origWindowSizeInDIP(); ow == width && oh == height && u.isFullscreen() == fullscreen && u.lastDeviceScaleFactor == scale {
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

	u.setWindowSizeInDIPImpl(width, height, fullscreen)

	u.updateWindowSizeLimits()

	u.adjustViewSize()

	// As width might be updated, update windowWidth/Height here.
	u.setOrigWindowSizeInDIP(width, height)
}

func (u *userInterfaceImpl) minimumWindowWidth() int {
	if u.window.GetAttrib(glfw.Decorated) == glfw.False {
		return 1
	}

	// On Windows, giving a too small width doesn't call a callback (#165).
	// To prevent hanging up, return asap if the width is too small.
	// 126 is an arbitrary number and I guess this is small enough .
	if runtime.GOOS == "windows" {
		return 126
	}

	// On macOS, resizing the window by cursor sometimes ignores the minimum size.
	// To avoid the flaky behavior, do not add a limitation.
	return 1
}

func (u *userInterfaceImpl) setWindowSizeInDIPImpl(width, height int, fullscreen bool) {
	if fullscreen {
		if x, y := u.origWindowPos(); x == invalidPos || y == invalidPos {
			u.setOrigWindowPos(u.window.GetPos())
		}

		if u.isNativeFullscreenAvailable() {
			u.setNativeFullscreen(fullscreen)
		} else {
			m := u.currentMonitor()
			if m == nil {
				return
			}

			v := m.GetVideoMode()
			u.window.SetMonitor(m, 0, 0, v.Width, v.Height, v.RefreshRate)

			// Swapping buffer is necessary to prevent the image lag (#1004).
			// TODO: This might not work when vsync is disabled.
			if u.graphicsDriver.IsGL() {
				glfw.PollEvents()
				u.swapBuffers()
			}
		}
		return
	}

	if mw := u.minimumWindowWidth(); width < mw {
		width = mw
	}
	if u.isNativeFullscreenAvailable() && u.isNativeFullscreen() {
		u.setNativeFullscreen(false)
	} else if !u.isNativeFullscreenAvailable() && u.window.GetMonitor() != nil {
		ww := int(u.dipToGLFWPixel(float64(width), u.currentMonitor()))
		wh := int(u.dipToGLFWPixel(float64(height), u.currentMonitor()))
		u.window.SetMonitor(nil, 0, 0, ww, wh, 0)
		glfw.PollEvents()
		u.swapBuffers()
	}

	// TODO: origWindowPos should always return invalidPos, then this logic should not be needed.
	if x, y := u.origWindowPos(); x != invalidPos && y != invalidPos {
		u.window.SetPos(x, y)
		// Dirty hack for macOS (#703). Rendering doesn't work correctly with one SetPos, but
		// work with two or more SetPos.
		if runtime.GOOS == "darwin" {
			u.window.SetPos(x+1, y)
			u.window.SetPos(x, y)
		}
		u.setOrigWindowPos(invalidPos, invalidPos)
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
}

// updateVsync must be called on the main thread.
func (u *userInterfaceImpl) updateVsync() {
	if u.graphicsDriver.IsGL() {
		// SwapInterval is affected by the current monitor of the window.
		// This needs to be called at least after SetMonitor.
		// Without SwapInterval after SetMonitor, vsynch doesn't work (#375).
		//
		// TODO: (#405) If triple buffering is needed, SwapInterval(0) should be called,
		// but is this correct? If glfw.SwapInterval(0) and the driver doesn't support triple
		// buffering, what will happen?
		if u.fpsMode == FPSModeVsyncOn {
			glfw.SwapInterval(1)
		} else {
			glfw.SwapInterval(0)
		}
	}
	u.graphicsDriver.SetVsyncEnabled(u.fpsMode == FPSModeVsyncOn)
}

// currentMonitor returns the current active monitor.
//
// currentMonitor must be called on the main thread.
func (u *userInterfaceImpl) currentMonitor() *glfw.Monitor {
	if u.window == nil {
		return u.initMonitor
	}
	if m := monitorFromWindow(u.window); m != nil {
		return m
	}
	return glfw.GetPrimaryMonitor()
}

// monitorFromWindow returns the monitor from the given window.
//
// monitorFromWindow must be called on the main thread.
func monitorFromWindow(window *glfw.Window) *glfw.Monitor {
	// GetMonitor is available only in fullscreen.
	if m := window.GetMonitor(); m != nil {
		return m
	}

	// Getting a monitor from a window position is not reliable in general (e.g., when a window is put across
	// multiple monitors, or, before SetWindowPosition is called.).
	// Get the monitor which the current window belongs to. This requires OS API.
	if m := monitorFromWindowByOS(window); m != nil {
		return m
	}

	// As the fallback, detect the monitor from the window.
	if m := getMonitorFromPosition(window.GetPos()); m != nil {
		return m.m
	}

	return nil
}

func (u *userInterfaceImpl) SetScreenTransparent(transparent bool) {
	if !u.isRunning() {
		u.setInitScreenTransparent(transparent)
		return
	}
	panic("ui: SetScreenTransparent can't be called after the main loop starts")
}

func (u *userInterfaceImpl) IsScreenTransparent() bool {
	if !u.isRunning() {
		return u.isInitScreenTransparent()
	}
	val := false
	u.t.Call(func() {
		val = u.window.GetAttrib(glfw.TransparentFramebuffer) == glfw.True
	})
	return val
}

func (u *userInterfaceImpl) resetForTick() {
	u.input.resetForTick()

	u.m.Lock()
	u.windowBeingClosed = false
	u.m.Unlock()
}

func (u *userInterfaceImpl) SetInitFocused(focused bool) {
	if u.isRunning() {
		panic("ui: SetInitFocused must be called before the main loop")
	}
	u.setInitFocused(focused)
}

func (u *userInterfaceImpl) Input() *Input {
	return &u.input
}

func (u *userInterfaceImpl) Window() Window {
	if microsoftgdk.IsXbox() {
		return &nullWindow{}
	}
	return &u.iwindow
}

// GLFW's functions to manipulate a window can invoke the SetSize callback (#1576, #1585, #1606).
// As the callback must not be called in the frame (between BeginFrame and EndFrame),
// disable the callback temporarily.

// maximizeWindow must be called from the main thread.
func (u *userInterfaceImpl) maximizeWindow() {
	// TODO: Can we remove this condition?
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

	if u.isFullscreen() {
		return
	}

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

// iconifyWindow must be called from the main thread.
func (u *userInterfaceImpl) iconifyWindow() {
	// Iconifying a native fullscreen window on macOS is forbidden.
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
func (u *userInterfaceImpl) restoreWindow() {
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
			time.Sleep(time.Second / 60)
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
func (u *userInterfaceImpl) setWindowDecorated(decorated bool) {
	if microsoftgdk.IsXbox() {
		return
	}

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
func (u *userInterfaceImpl) setWindowFloating(floating bool) {
	if microsoftgdk.IsXbox() {
		return
	}

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

// setWindowResizingMode must be called from the main thread.
func (u *userInterfaceImpl) setWindowResizingMode(mode WindowResizingMode) {
	if microsoftgdk.IsXbox() {
		return
	}

	if u.windowResizingMode == mode {
		return
	}

	if u.setSizeCallbackEnabled {
		u.setSizeCallbackEnabled = false
		defer func() {
			u.setSizeCallbackEnabled = true
		}()
	}

	u.windowResizingMode = mode

	v := glfw.False
	if mode == WindowResizingModeEnabled {
		v = glfw.True
	}
	u.window.SetAttrib(glfw.Resizable, v)
	u.setWindowResizingModeForOS(mode)
}

// setWindowPositionInDIP sets the window position.
//
// x and y are the position in device-independent pixels.
//
// setWindowPositionInDIP must be called from the main thread.
func (u *userInterfaceImpl) setWindowPositionInDIP(x, y int, monitor *glfw.Monitor) {
	if microsoftgdk.IsXbox() {
		// Do nothing. The position is always fixed.
		return
	}

	if u.setSizeCallbackEnabled {
		u.setSizeCallbackEnabled = false
		defer func() {
			u.setSizeCallbackEnabled = true
		}()
	}

	mx, my := monitor.GetPos()
	xf := u.dipToGLFWPixel(float64(x), monitor)
	yf := u.dipToGLFWPixel(float64(y), monitor)
	if x, y := u.adjustWindowPosition(mx+int(xf), my+int(yf), monitor); u.isFullscreen() {
		u.setOrigWindowPos(x, y)
	} else {
		u.window.SetPos(x, y)
	}
}

// setWindowTitle must be called from the main thread.
func (u *userInterfaceImpl) setWindowTitle(title string) {
	if u.setSizeCallbackEnabled {
		u.setSizeCallbackEnabled = false
		defer func() {
			u.setSizeCallbackEnabled = true
		}()
	}

	u.window.SetTitle(title)
}

// forceToRefreshIfNeeded forces to refresh the framebuffer by resizing the window quickly.
// This is a very dirty but necessary hack for DirectX (#2050).
// With DirectX, the framebuffer is not rendered correctly when the window is resized by dragging
// or just after the resizing finishes by dragging.
// forceToRefreshIfNeeded must be called from the main thread.
func (u *userInterfaceImpl) forceToRefreshIfNeeded() {
	if !u.graphicsDriver.IsDirectX() {
		return
	}

	x, y := u.window.GetPos()
	u.window.SetPos(x+1, y+1)
	glfw.PollEvents()
	time.Sleep(time.Millisecond)
	u.window.SetPos(x, y)
	glfw.PollEvents()
	time.Sleep(time.Millisecond)
}

// isWindowMaximized must be called from the main thread.
func (u *userInterfaceImpl) isWindowMaximized() bool {
	return u.window.GetAttrib(glfw.Maximized) == glfw.True && !u.isNativeFullscreen()
}
