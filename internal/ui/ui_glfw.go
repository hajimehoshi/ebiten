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

package ui

import (
	"errors"
	"fmt"
	"image"
	"math"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hajimehoshi/ebiten/v2/internal/devicescale"
	"github.com/hajimehoshi/ebiten/v2/internal/file"
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
	terminated           uint32
	runnableOnUnfocused  bool
	fpsMode              FPSModeType
	iconImages           []image.Image
	cursorShape          CursorShape
	windowClosingHandled bool
	windowResizingMode   WindowResizingMode
	inFrame              uint32

	// err must be accessed from the main thread.
	err error

	lastDeviceScaleFactor float64

	// These values are not changed after initialized.
	// TODO: the fullscreen size should be updated when the initial window position is changed?
	initMonitor               *Monitor
	initDeviceScaleFactor     float64
	initFullscreenWidthInDIP  int
	initFullscreenHeightInDIP int

	initFullscreen             bool
	initCursorMode             CursorMode
	initWindowDecorated        bool
	initWindowPositionXInDIP   int
	initWindowPositionYInDIP   int
	initWindowWidthInDIP       int
	initWindowHeightInDIP      int
	initWindowFloating         bool
	initWindowMaximized        bool
	initWindowMousePassthrough bool

	// bufferOnceSwapped must be accessed from the main thread.
	bufferOnceSwapped bool

	origWindowPosX        int
	origWindowPosY        int
	origWindowWidthInDIP  int
	origWindowHeightInDIP int

	fpsModeInited bool

	inputState   InputState
	iwindow      glfwWindow
	savedCursorX float64
	savedCursorY float64

	sizeCallback                   glfw.SizeCallback
	closeCallback                  glfw.CloseCallback
	framebufferSizeCallback        glfw.FramebufferSizeCallback
	defaultFramebufferSizeCallback glfw.FramebufferSizeCallback
	dropCallback                   glfw.DropCallback
	framebufferSizeCallbackCh      chan struct{}

	darwinInitOnce        sync.Once
	bufferOnceSwappedOnce sync.Once

	mainThread   thread.Thread
	renderThread thread.Thread
	m            sync.RWMutex
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
		fpsMode:                  FPSModeVsyncOn,
		origWindowPosX:           invalidPos,
		origWindowPosY:           invalidPos,
		savedCursorX:             math.NaN(),
		savedCursorY:             math.NaN(),
	}
	theUI.iwindow.ui = &theUI.userInterfaceImpl
}

func init() {
	hideConsoleWindowOnWindows()

	if err := initialize(); err != nil {
		panic(err)
	}
	glfw.SetMonitorCallback(glfw.ToMonitorCallback(func(monitor *glfw.Monitor, event glfw.PeripheralEvent) {
		theMonitors.update()
	}))
}

var glfwSystemCursors = map[CursorShape]*glfw.Cursor{}

func initialize() error {
	if err := glfw.Init(); err != nil {
		return err
	}

	glfw.WindowHint(glfw.Visible, glfw.False)
	glfw.WindowHint(glfw.ClientAPI, glfw.NoAPI)

	// Update the monitor first. The monitor state is depended on various functions like initialMonitorByOS.
	theMonitors.update()

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

	theUI.setInitMonitor(theMonitors.monitorFromGLFWMonitor(m))

	// Create system cursors. These cursors are destroyed at glfw.Terminate().
	glfwSystemCursors[CursorShapeDefault] = nil
	glfwSystemCursors[CursorShapeText] = glfw.CreateStandardCursor(glfw.IBeamCursor)
	glfwSystemCursors[CursorShapeCrosshair] = glfw.CreateStandardCursor(glfw.CrosshairCursor)
	glfwSystemCursors[CursorShapePointer] = glfw.CreateStandardCursor(glfw.HandCursor)
	glfwSystemCursors[CursorShapeEWResize] = glfw.CreateStandardCursor(glfw.HResizeCursor)
	glfwSystemCursors[CursorShapeNSResize] = glfw.CreateStandardCursor(glfw.VResizeCursor)
	glfwSystemCursors[CursorShapeNESWResize] = glfw.CreateStandardCursor(glfw.ResizeNESWCursor)
	glfwSystemCursors[CursorShapeNWSEResize] = glfw.CreateStandardCursor(glfw.ResizeNWSECursor)
	glfwSystemCursors[CursorShapeMove] = glfw.CreateStandardCursor(glfw.ResizeAllCursor)
	glfwSystemCursors[CursorShapeNotAllowed] = glfw.CreateStandardCursor(glfw.NotAllowedCursor)

	return nil
}

func (u *userInterfaceImpl) setInitMonitor(m *Monitor) {
	u.m.Lock()
	defer u.m.Unlock()

	u.initMonitor = m

	// TODO: Remove these members. These can be calculated anytime from initMonitor.
	u.initDeviceScaleFactor = u.deviceScaleFactor(m.m)
	v := m.vm
	u.initFullscreenWidthInDIP = int(u.dipFromGLFWMonitorPixel(float64(v.Width), m.m))
	u.initFullscreenHeightInDIP = int(u.dipFromGLFWMonitorPixel(float64(v.Height), m.m))
}

// AppendMonitors appends the current monitors to the passed in mons slice and returns it.
func (u *userInterfaceImpl) AppendMonitors(monitors []*Monitor) []*Monitor {
	return theMonitors.append(monitors)
}

// Monitor returns the window's current monitor. Returns nil if there is no current monitor yet.
func (u *userInterfaceImpl) Monitor() *Monitor {
	if !u.isRunning() {
		return nil
	}
	var monitor *Monitor
	u.mainThread.Call(func() {
		if u.isTerminated() {
			return
		}
		glfwMonitor := u.currentMonitor()
		if glfwMonitor == nil {
			return
		}
		monitor = theMonitors.monitorFromGLFWMonitor(glfwMonitor)
	})
	return monitor
}

// getMonitorFromPosition returns a monitor for the given window x/y,
// or returns nil if monitor is not found.
//
// getMonitorFromPosition must be called on the main thread.
func getMonitorFromPosition(wx, wy int) *Monitor {
	for _, m := range theMonitors.append(nil) {
		// TODO: Fix incorrectness in the cases of https://github.com/glfw/glfw/issues/1961.
		// See also internal/devicescale/impl_desktop.go for a maybe better way of doing this.
		if m.x <= wx && wx < m.x+m.vm.Width && m.y <= wy && wy < m.y+m.vm.Height {
			return m
		}
	}
	return nil
}

func (u *userInterfaceImpl) isRunning() bool {
	return atomic.LoadUint32(&u.running) != 0 && !u.isTerminated()
}

func (u *userInterfaceImpl) isTerminated() bool {
	return atomic.LoadUint32(&u.terminated) != 0
}

func (u *userInterfaceImpl) setRunning(running bool) {
	if running {
		atomic.StoreUint32(&u.running, 1)
	} else {
		atomic.StoreUint32(&u.running, 0)
	}
}

func (u *userInterfaceImpl) setTerminated() {
	atomic.StoreUint32(&u.terminated, 1)
}

// setWindowMonitor must be called on the main thread.
func (u *userInterfaceImpl) setWindowMonitor(monitor *Monitor) {
	if microsoftgdk.IsXbox() {
		return
	}

	// Ignore if it is the same monitor.
	if monitor.m == u.currentMonitor() {
		return
	}

	ww := u.origWindowWidthInDIP
	wh := u.origWindowHeightInDIP

	fullscreen := u.isFullscreen()
	// This is copied from setFullscreen. They should probably use a shared function.
	if fullscreen {
		u.setFullscreen(false)
		// Just after exiting fullscreen, the window state seems very unstable (#2758).
		// Wait for a while with polling events.
		if runtime.GOOS == "darwin" {
			for i := 0; i < 60; i++ {
				glfw.PollEvents()
				time.Sleep(time.Second / 60)
			}
		}
	}

	w := u.dipToGLFWPixel(float64(ww), monitor.m)
	h := u.dipToGLFWPixel(float64(wh), monitor.m)
	x, y := monitor.x, monitor.y
	mw := u.dipFromGLFWMonitorPixel(float64(monitor.width), monitor.m)
	mh := u.dipFromGLFWMonitorPixel(float64(monitor.height), monitor.m)
	mw = u.dipToGLFWPixel(mw, monitor.m)
	mh = u.dipToGLFWPixel(mh, monitor.m)
	px, py := InitialWindowPosition(int(mw), int(mh), int(w), int(h))
	u.window.SetPos(x+px, y+py)

	if fullscreen {
		// Calling setFullscreen immediately might not work well, especially on Linux (#2778).
		// Just wait a little bit. 1/30[s] seems enough in most cases.
		time.Sleep(time.Second / 30)
		u.setFullscreen(true)
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

func (u *userInterfaceImpl) isWindowMaximizable() bool {
	_, _, maxw, maxh := u.getWindowSizeLimitsInDIP()
	return maxw == glfw.DontCare && maxh == glfw.DontCare
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

func (u *userInterfaceImpl) getAndResetIconImages() []image.Image {
	u.m.RLock()
	defer u.m.RUnlock()
	i := u.iconImages
	u.iconImages = nil
	return i
}

func (u *userInterfaceImpl) setIconImages(iconImages []image.Image) {
	u.m.Lock()
	defer u.m.Unlock()

	// Even if iconImages is nil, always create a slice.
	// A 0-size slice and nil are distinguished.
	// See the comment in updateIconIfNeeded.
	u.iconImages = make([]image.Image, len(iconImages))
	copy(u.iconImages, iconImages)
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

func (u *userInterfaceImpl) isInitWindowMousePassthrough() bool {
	u.m.RLock()
	defer u.m.RUnlock()
	return u.initWindowMousePassthrough
}

func (u *userInterfaceImpl) setInitWindowMousePassthrough(enabled bool) {
	u.m.Lock()
	defer u.m.Unlock()
	u.initWindowMousePassthrough = enabled
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

func (u *userInterfaceImpl) ScreenSizeInFullscreen() (int, int) {
	if u.isTerminated() {
		return 0, 0
	}
	if !u.isRunning() {
		return u.initFullscreenWidthInDIP, u.initFullscreenHeightInDIP
	}

	var w, h int
	u.mainThread.Call(func() {
		if u.isTerminated() {
			return
		}
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
	if microsoftgdk.IsXbox() {
		return false
	}

	if u.isTerminated() {
		return false
	}
	if !u.isRunning() {
		return u.isInitFullscreen()
	}
	var b bool
	u.mainThread.Call(func() {
		if u.isTerminated() {
			return
		}
		b = u.isFullscreen()
	})
	return b
}

func (u *userInterfaceImpl) SetFullscreen(fullscreen bool) {
	if microsoftgdk.IsXbox() {
		return
	}

	if u.isTerminated() {
		return
	}
	if !u.isRunning() {
		u.setInitFullscreen(fullscreen)
		return
	}

	u.mainThread.Call(func() {
		if u.isTerminated() {
			return
		}
		if u.isFullscreen() == fullscreen {
			return
		}
		u.setFullscreen(fullscreen)
	})
}

func (u *userInterfaceImpl) IsFocused() bool {
	if !u.isRunning() {
		return false
	}

	var focused bool
	u.mainThread.Call(func() {
		if u.isTerminated() {
			return
		}
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
	if u.isTerminated() {
		return
	}
	if !u.isRunning() {
		u.m.Lock()
		u.fpsMode = mode
		u.m.Unlock()
		return
	}
	u.mainThread.Call(func() {
		if u.isTerminated() {
			return
		}
		if !u.fpsModeInited {
			u.fpsMode = mode
			return
		}
		u.setFPSMode(mode)
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
	if u.isTerminated() {
		return 0
	}
	if !u.isRunning() {
		return u.getInitCursorMode()
	}

	var mode int
	u.mainThread.Call(func() {
		if u.isTerminated() {
			return
		}
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
	if u.isTerminated() {
		return
	}
	if !u.isRunning() {
		u.setInitCursorMode(mode)
		return
	}
	u.mainThread.Call(func() {
		if u.isTerminated() {
			return
		}
		u.window.SetInputMode(glfw.CursorMode, driverCursorModeToGLFWCursorMode(mode))
		if mode == CursorModeVisible {
			u.window.SetCursor(glfwSystemCursors[u.getCursorShape()])
		}
	})
}

func (u *userInterfaceImpl) CursorShape() CursorShape {
	return u.getCursorShape()
}

func (u *userInterfaceImpl) SetCursorShape(shape CursorShape) {
	if u.isTerminated() {
		return
	}

	old := u.setCursorShape(shape)
	if old == shape {
		return
	}
	if !u.isRunning() {
		return
	}
	u.mainThread.Call(func() {
		if u.isTerminated() {
			return
		}
		u.window.SetCursor(glfwSystemCursors[shape])
	})
}

func (u *userInterfaceImpl) DeviceScaleFactor() float64 {
	if u.isTerminated() {
		return 0
	}
	if !u.isRunning() {
		return u.initDeviceScaleFactor
	}

	var f float64
	u.mainThread.Call(func() {
		if u.isTerminated() {
			return
		}
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
func (u *userInterfaceImpl) createWindow(width, height int, monitor *glfw.Monitor) error {
	if u.window != nil {
		panic("ui: u.window must not exist at createWindow")
	}

	// As a start, create a window with temporary size to create OpenGL context thread.
	window, err := glfw.CreateWindow(width, height, "", nil, nil)
	if err != nil {
		return err
	}

	// Set our target monitor if provided. This is required to prevent an initial window flash on the default monitor.
	x, y := monitor.GetPos()
	vm := monitor.GetVideoMode()
	mw := u.dipFromGLFWMonitorPixel(float64(vm.Width), monitor)
	mh := u.dipFromGLFWMonitorPixel(float64(vm.Height), monitor)
	mw = u.dipToGLFWPixel(mw, monitor)
	mh = u.dipToGLFWPixel(mh, monitor)
	px, py := InitialWindowPosition(int(mw), int(mh), width, height)
	window.SetPos(x+px, y+py)

	initializeWindowAfterCreation(window)

	u.window = window

	// Even just after a window creation, FramebufferSize callback might be invoked (#1847).
	// Ensure to consume this callback.
	u.waitForFramebufferSizeCallback(u.window, nil)

	u.window.SetInputMode(glfw.CursorMode, driverCursorModeToGLFWCursorMode(u.getInitCursorMode()))
	u.window.SetCursor(glfwSystemCursors[u.getCursorShape()])
	u.window.SetTitle(u.title)
	// Icons are set after every frame. They don't have to be cared here.

	u.updateWindowSizeLimits()

	return nil
}

func (u *userInterfaceImpl) beginFrame() {
	atomic.StoreUint32(&u.inFrame, 1)
}

func (u *userInterfaceImpl) endFrame() {
	atomic.StoreUint32(&u.inFrame, 0)
}

// registerWindowCloseCallback must be called from the main thread.
func (u *userInterfaceImpl) registerWindowCloseCallback() {
	if u.closeCallback == nil {
		u.closeCallback = glfw.ToCloseCallback(func(_ *glfw.Window) {
			u.m.Lock()
			u.inputState.WindowBeingClosed = true
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
			u.setWindowSizeInDIP(ww, wh, false)
		})
	}
	u.window.SetFramebufferSizeCallback(u.defaultFramebufferSizeCallback)
}

func (u *userInterfaceImpl) registerDropCallback() {
	if u.dropCallback == nil {
		u.dropCallback = glfw.ToDropCallback(func(_ *glfw.Window, names []string) {
			u.m.Lock()
			defer u.m.Unlock()
			u.inputState.DroppedFiles = file.NewVirtualFS(names)
		})
	}
	u.window.SetDropCallback(u.dropCallback)
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

func (u *userInterfaceImpl) initOnMainThread(options *RunOptions) error {
	glfw.WindowHint(glfw.AutoIconify, glfw.False)

	// On macOS, window decoration should be initialized once after buffers are swapped (#2600).
	if runtime.GOOS != "darwin" {
		decorated := glfw.False
		if u.isInitWindowDecorated() {
			decorated = glfw.True
		}
		glfw.WindowHint(glfw.Decorated, decorated)
	}

	glfwTransparent := glfw.False
	if options.ScreenTransparent {
		glfwTransparent = glfw.True
	}
	glfw.WindowHint(glfw.TransparentFramebuffer, glfwTransparent)

	g, err := newGraphicsDriver(&graphicsDriverCreatorImpl{
		transparent: options.ScreenTransparent,
	}, options.GraphicsLibrary)
	if err != nil {
		return err
	}
	u.graphicsDriver = g
	u.graphicsDriver.SetTransparent(options.ScreenTransparent)

	if u.graphicsDriver.IsGL() {
		u.graphicsDriver.(interface{ SetGLFWClientAPI() }).SetGLFWClientAPI()
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

	focused := glfw.True
	if options.InitUnfocused {
		focused = glfw.False
	}
	glfw.WindowHint(glfw.FocusOnShow, focused)

	mousePassthrough := glfw.False
	if u.isInitWindowMousePassthrough() {
		mousePassthrough = glfw.True
	}
	glfw.WindowHint(glfw.MousePassthrough, mousePassthrough)

	// Set the window visible explicitly or the application freezes on Wayland (#974).
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		glfw.WindowHint(glfw.Visible, glfw.True)
	}

	ww, wh := u.getInitWindowSizeInDIP()
	initW := int(u.dipToGLFWPixel(float64(ww), u.initMonitor.m))
	initH := int(u.dipToGLFWPixel(float64(wh), u.initMonitor.m))
	if err := u.createWindow(initW, initH, u.initMonitor.m); err != nil {
		return err
	}

	// The position must be set before the size is set (#1982).
	// setWindowSize refers the current monitor's device scale.
	// TODO: The window position is already set at createWindow. Unify the logic.
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
	u.setWindowPositionInDIP(wx, wy, u.initMonitor.m)
	u.setWindowSizeInDIP(ww, wh, true)

	// Maximizing a window requires a proper size and position. Call Maximize here (#1117).
	if u.isInitWindowMaximized() {
		u.window.Maximize()
	}

	u.setWindowResizingModeForOS(u.windowResizingMode)

	if options.SkipTaskbar {
		// Ignore the error.
		_ = u.skipTaskbar()
	}

	// On macOS, the window is shown once after buffers are swapped at update.
	if runtime.GOOS != "darwin" {
		u.window.Show()
	}

	if g, ok := u.graphicsDriver.(interface{ SetWindow(uintptr) }); ok {
		g.SetWindow(u.nativeWindow())
	}

	gamepad.SetNativeWindow(u.nativeWindow())

	// Register callbacks after the window initialization done.
	// The callback might cause swapping frames, that assumes the window is already set (#2137).
	u.registerWindowCloseCallback()
	u.registerWindowFramebufferSizeCallback()
	u.registerInputCallbacks()
	u.registerDropCallback()

	return nil
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
		return float64(u.origWindowWidthInDIP), float64(u.origWindowHeightInDIP)
	}

	// Instead of u.origWindow{Width,Height}InDIP, use the actual window size here.
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

	// On macOS, one swapping buffers seems required before entering fullscreen (#2599).
	if u.isInitFullscreen() && (u.bufferOnceSwapped || runtime.GOOS != "darwin") {
		u.setFullscreen(true)
		u.setInitFullscreen(false)
	}

	if runtime.GOOS == "darwin" && u.bufferOnceSwapped {
		u.darwinInitOnce.Do(func() {
			// On macOS, window decoration should be initialized once after buffers are swapped (#2600).
			decorated := glfw.False
			if u.isInitWindowDecorated() {
				decorated = glfw.True
			}
			u.window.SetAttrib(glfw.Decorated, decorated)

			// The window is not shown at the initialization on macOS. Show the window here.
			u.window.Show()
		})
	}

	// Initialize vsync after SetMonitor is called. See the comment in updateVsync.
	// Calling this inside setWindowSize didn't work (#1363).
	if !u.fpsModeInited {
		u.setFPSMode(u.fpsMode)
	}

	if u.fpsMode != FPSModeVsyncOffMinimum {
		// TODO: Updating the input can be skipped when clock.Update returns 0 (#1367).
		glfw.PollEvents()
	} else {
		glfw.WaitEvents()
	}

	// In the initial state on macOS, the window is not shown (#2620).
	for u.window.GetAttrib(glfw.Visible) != 0 && !u.isRunnableOnUnfocused() && u.window.GetAttrib(glfw.Focused) == 0 && !u.window.ShouldClose() {
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

func (u *userInterfaceImpl) loopGame() error {
	defer func() {
		// Post a task to the render thread to ensure all the queued functions are executed.
		// glfw.Terminate will remove the context and any graphics calls after that will be invalidated.
		u.renderThread.Call(func() {})
		u.mainThread.Call(func() {
			glfw.Terminate()
			u.setTerminated()
		})
	}()

	u.renderThread.Call(func() {
		if u.graphicsDriver.IsGL() {
			u.window.MakeContextCurrent()
		}
	})

	for {
		if err := u.updateGame(); err != nil {
			return err
		}
	}
}

func (u *userInterfaceImpl) updateGame() error {
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
	if u.mainThread.Call(func() {
		outsideWidth, outsideHeight, err = u.update()
		deviceScaleFactor = u.deviceScaleFactor(u.currentMonitor())
	}); err != nil {
		return err
	}

	if err := u.context.updateFrame(u.graphicsDriver, outsideWidth, outsideHeight, deviceScaleFactor, u, func() {
		// Call updateVsync even though fpsMode is not updated.
		// When toggling to fullscreen, vsync state might be reset unexpectedly (#1787).
		u.updateVsyncOnRenderThread()

		// This works only for OpenGL.
		u.swapBuffersOnRenderThread()
	}); err != nil {
		return err
	}

	u.bufferOnceSwappedOnce.Do(func() {
		u.mainThread.Call(func() {
			u.bufferOnceSwapped = true
		})
	})

	if unfocused {
		t2 = time.Now()
	}

	// When a window is not focused or in another space, SwapBuffers might return immediately and CPU might be busy.
	// Mitigate this by sleeping (#982, #2521).
	if unfocused {
		d := t2.Sub(t1)
		const wait = time.Second / 60
		if d < wait {
			time.Sleep(wait - d)
		}
	}

	return nil
}

func (u *userInterfaceImpl) updateIconIfNeeded() error {
	// In the fullscreen mode, SetIcon fails (#1578).
	if u.isFullscreen() {
		return nil
	}

	imgs := u.getAndResetIconImages()
	// A 0-size slice and nil are distinguished here.
	// A 0-size slice means a user indicates to reset the icon.
	// On the other hand, nil means a user didn't update the icon state.
	if imgs == nil {
		return nil
	}

	var newImgs []image.Image
	if len(imgs) > 0 {
		newImgs = make([]image.Image, len(imgs))
	}
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

	// Catch a possible error at 'At' (#2647).
	if err := theGlobalState.error(); err != nil {
		return err
	}

	u.mainThread.Call(func() {
		u.window.SetIcon(newImgs)
	})

	return nil
}

func (u *userInterfaceImpl) swapBuffersOnRenderThread() {
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

	// The window size limit affects the resizing mode, especially on macOS (#).
	u.setWindowResizingModeForOS(u.windowResizingMode)
}

// disableWindowSizeLimits disables a window size limitation temporarily, especially for fullscreen
// In order to enable the size limitation, call updateWindowSizeLimits.
//
// disableWindowSizeLimits must be called from the main thread.
func (u *userInterfaceImpl) disableWindowSizeLimits() {
	u.window.SetSizeLimits(glfw.DontCare, glfw.DontCare, glfw.DontCare, glfw.DontCare)
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
func (u *userInterfaceImpl) setWindowSizeInDIP(width, height int, callSetSize bool) {
	if microsoftgdk.IsXbox() {
		// Do nothing. The size is always fixed.
		return
	}

	width, height = u.adjustWindowSizeBasedOnSizeLimitsInDIP(width, height)
	if m := u.minimumWindowWidth(); width < m {
		width = m
	}
	if height < 1 {
		height = 1
	}

	scale := u.deviceScaleFactor(u.currentMonitor())
	if u.origWindowWidthInDIP == width && u.origWindowHeightInDIP == height && u.lastDeviceScaleFactor == scale {
		return
	}
	u.lastDeviceScaleFactor = scale

	u.origWindowWidthInDIP = width
	u.origWindowHeightInDIP = height

	if !u.isFullscreen() && callSetSize {
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

	u.updateWindowSizeLimits()
}

// setOrigWindowPosWithCurrentPos must be called from the main thread.
func (u *userInterfaceImpl) setOrigWindowPosWithCurrentPos() {
	if x, y := u.origWindowPos(); x == invalidPos || y == invalidPos {
		u.setOrigWindowPos(u.window.GetPos())
	}
}

// setFullscreen must be called from the main thread.
func (u *userInterfaceImpl) setFullscreen(fullscreen bool) {
	if u.isFullscreen() == fullscreen {
		return
	}

	if u.window.GetInputMode(glfw.CursorMode) == glfw.CursorDisabled {
		u.saveCursorPosition()
	}

	// Enter the fullscreen.
	if fullscreen {
		u.disableWindowSizeLimits()

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
		}
		u.adjustViewSizeAfterFullscreen()
		return
	}

	// Exit the fullscreen.
	u.updateWindowSizeLimits()

	// Get the original window position and size before changing the state of fullscreen.
	// TODO: Why?
	origX, origY := u.origWindowPos()

	ww := int(u.dipToGLFWPixel(float64(u.origWindowWidthInDIP), u.currentMonitor()))
	wh := int(u.dipToGLFWPixel(float64(u.origWindowHeightInDIP), u.currentMonitor()))
	if u.isNativeFullscreenAvailable() {
		u.setNativeFullscreen(false)
		// Adjust the window size later (after adjusting the position).
	} else if !u.isNativeFullscreenAvailable() && u.window.GetMonitor() != nil {
		u.window.SetMonitor(nil, 0, 0, ww, wh, 0)
	}

	// glfw.PollEvents is necessary for macOS to enable (*glfw.Window).SetPos and SetSize (#2296).
	// This polling causes issues on Linux and Windows when rapidly toggling fullscreen, so we only run it under macOS.
	if runtime.GOOS == "darwin" {
		glfw.PollEvents()
	}

	if origX != invalidPos && origY != invalidPos {
		u.window.SetPos(origX, origY)
		// Dirty hack for macOS (#703). Rendering doesn't work correctly with one SetPos, but
		// work with two or more SetPos.
		if runtime.GOOS == "darwin" {
			u.window.SetPos(origX+1, origY)
			u.window.SetPos(origX, origY)
		}
		u.setOrigWindowPos(invalidPos, invalidPos)
	}

	if u.isNativeFullscreenAvailable() {
		// Set the window size after the position. The order matters.
		// In the opposite order, the window size might not be correct when going back from fullscreen with multi monitors.
		u.window.SetSize(ww, wh)
	}
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

func (u *userInterfaceImpl) updateVsyncOnRenderThread() {
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
		return u.initMonitor.m
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

func (u *userInterfaceImpl) readInputState(inputState *InputState) {
	u.m.Lock()
	defer u.m.Unlock()
	u.inputState.copyAndReset(inputState)
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
	if u.isNativeFullscreen() {
		return
	}

	if u.isFullscreen() {
		return
	}

	u.window.Maximize()

	// On Linux/UNIX, maximizing might not finish even though Maximize returns. Just wait for its finish.
	// Do not check this in the fullscreen since apparently the condition can never be true.
	for u.window.GetAttrib(glfw.Maximized) != glfw.True {
		glfw.PollEvents()
	}
}

// iconifyWindow must be called from the main thread.
func (u *userInterfaceImpl) iconifyWindow() {
	// Iconifying a native fullscreen window on macOS is forbidden.
	if u.isNativeFullscreen() {
		return
	}

	u.window.Iconify()

	// On Linux/UNIX, iconifying might not finish even though Iconify returns. Just wait for its finish.
	for u.window.GetAttrib(glfw.Iconified) != glfw.True {
		glfw.PollEvents()
	}
}

// restoreWindow must be called from the main thread.
func (u *userInterfaceImpl) restoreWindow() {
	u.window.Restore()

	// On Linux/UNIX, restoring might not finish even though Restore returns (#1608). Just wait for its finish.
	// On macOS, the restoring state might be the same as the maximized state. Skip this.
	if runtime.GOOS != "darwin" {
		for u.window.GetAttrib(glfw.Maximized) == glfw.True || u.window.GetAttrib(glfw.Iconified) == glfw.True {
			glfw.PollEvents()
			time.Sleep(time.Second / 60)
		}
	}
}

// setWindowDecorated must be called from the main thread.
func (u *userInterfaceImpl) setWindowDecorated(decorated bool) {
	if microsoftgdk.IsXbox() {
		return
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

func (u *userInterfaceImpl) origWindowPos() (int, int) {
	return u.origWindowPosX, u.origWindowPosY
}

func (u *userInterfaceImpl) setOrigWindowPos(x, y int) {
	u.origWindowPosX = x
	u.origWindowPosY = y
}

// setWindowMousePassthrough must be called from the main thread.
func (u *userInterfaceImpl) setWindowMousePassthrough(enabled bool) {
	if microsoftgdk.IsXbox() {
		return
	}

	v := glfw.False
	if enabled {
		v = glfw.True
	}
	u.window.SetAttrib(glfw.MousePassthrough, v)
}

func IsScreenTransparentAvailable() bool {
	return true
}

func RunOnMainThread(f func()) {
	theUI.mainThread.Call(f)
}
