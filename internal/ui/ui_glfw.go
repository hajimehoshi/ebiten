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

//go:build !android && !ios && !js && !nintendosdk && !playstation5

package ui

import (
	"errors"
	"fmt"
	"image"
	"math"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2/internal/file"
	"github.com/hajimehoshi/ebiten/v2/internal/gamepad"
	"github.com/hajimehoshi/ebiten/v2/internal/glfw"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicscommand"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/hook"
	"github.com/hajimehoshi/ebiten/v2/internal/microsoftgdk"
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

	runnableOnUnfocused  bool
	fpsMode              FPSModeType
	iconImages           []image.Image
	cursorShape          CursorShape
	windowClosingHandled bool
	windowResizingMode   WindowResizingMode

	lastDeviceScaleFactor float64

	initMonitor                *Monitor
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

	initUnfocused bool

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

	closeCallback                  glfw.CloseCallback
	framebufferSizeCallback        glfw.FramebufferSizeCallback
	defaultFramebufferSizeCallback glfw.FramebufferSizeCallback
	dropCallback                   glfw.DropCallback
	framebufferSizeCallbackCh      chan struct{}

	darwinInitOnce        sync.Once
	showWindowOnce        sync.Once
	bufferOnceSwappedOnce sync.Once

	// immContext is used only in Windows.
	immContext uintptr

	m sync.RWMutex
}

const (
	maxInt     = int(^uint(0) >> 1)
	minInt     = -maxInt - 1
	invalidPos = minInt
)

func init() {
	// Lock the main thread.
	runtime.LockOSThread()
}

func (u *UserInterface) init() error {
	u.userInterfaceImpl = userInterfaceImpl{
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
		origWindowPosX:           invalidPos,
		origWindowPosY:           invalidPos,
		savedCursorX:             math.NaN(),
		savedCursorY:             math.NaN(),
	}
	u.iwindow.ui = u

	if err := u.initializePlatform(); err != nil {
		return err
	}
	if err := u.initializeGLFW(); err != nil {
		return err
	}
	if _, err := glfw.SetMonitorCallback(func(monitor *glfw.Monitor, event glfw.PeripheralEvent) {
		if err := theMonitors.update(); err != nil {
			u.setError(err)
		}
	}); err != nil {
		return err
	}

	return nil
}

var glfwSystemCursors = map[CursorShape]*glfw.Cursor{}

func (u *UserInterface) initializeGLFW() error {
	if err := glfw.Init(); err != nil {
		return err
	}

	// Update the monitor first. The monitor state is depended on various functions like initialMonitorByOS.
	if err := theMonitors.update(); err != nil {
		return err
	}

	m, err := initialMonitorByOS()
	if err != nil {
		return err
	}
	if m == nil {
		m = theMonitors.primaryMonitor()
	}

	// GetMonitors might return nil in theory (#1878, #1887).
	if m == nil {
		return errors.New("ui: no monitor was found at initializeGLFW")
	}

	u.setInitMonitor(m)

	// Create system cursors. These cursors are destroyed at glfw.Terminate().
	glfwSystemCursors[CursorShapeDefault] = nil

	c, err := glfw.CreateStandardCursor(glfw.IBeamCursor)
	if err != nil {
		return err
	}
	glfwSystemCursors[CursorShapeText] = c

	c, err = glfw.CreateStandardCursor(glfw.CrosshairCursor)
	if err != nil {
		return err
	}
	glfwSystemCursors[CursorShapeCrosshair] = c

	c, err = glfw.CreateStandardCursor(glfw.HandCursor)
	if err != nil {
		return err
	}
	glfwSystemCursors[CursorShapePointer] = c

	c, err = glfw.CreateStandardCursor(glfw.HResizeCursor)
	if err != nil {
		return err
	}
	glfwSystemCursors[CursorShapeEWResize] = c

	c, err = glfw.CreateStandardCursor(glfw.VResizeCursor)
	if err != nil {
		return err
	}
	glfwSystemCursors[CursorShapeNSResize] = c

	c, err = glfw.CreateStandardCursor(glfw.ResizeNESWCursor)
	if err != nil {
		return err
	}
	glfwSystemCursors[CursorShapeNESWResize] = c

	c, err = glfw.CreateStandardCursor(glfw.ResizeNWSECursor)
	if err != nil {
		return err
	}
	glfwSystemCursors[CursorShapeNWSEResize] = c

	c, err = glfw.CreateStandardCursor(glfw.ResizeAllCursor)
	if err != nil {
		return err
	}
	glfwSystemCursors[CursorShapeMove] = c

	c, err = glfw.CreateStandardCursor(glfw.NotAllowedCursor)
	if err != nil {
		return err
	}
	glfwSystemCursors[CursorShapeNotAllowed] = c

	return nil
}

func (u *UserInterface) setInitMonitor(m *Monitor) {
	u.m.Lock()
	defer u.m.Unlock()
	u.initMonitor = m
}

func (u *UserInterface) getInitMonitor() *Monitor {
	u.m.RLock()
	defer u.m.RUnlock()
	return u.initMonitor
}

// AppendMonitors appends the current monitors to the passed in mons slice and returns it.
func (u *UserInterface) AppendMonitors(monitors []*Monitor) []*Monitor {
	return theMonitors.append(monitors)
}

// Monitor returns the window's current monitor. Returns nil if there is no current monitor yet.
func (u *UserInterface) Monitor() *Monitor {
	if !u.isRunning() {
		return u.getInitMonitor()
	}
	var monitor *Monitor
	u.mainThread.Call(func() {
		if u.isTerminated() {
			return
		}
		m, err := u.currentMonitor()
		if err != nil {
			u.setError(err)
			return
		}
		monitor = m
	})
	return monitor
}

// setWindowMonitor must be called on the main thread.
func (u *UserInterface) setWindowMonitor(monitor *Monitor) error {
	if microsoftgdk.IsXbox() {
		return nil
	}

	// Ignore if it is the same monitor.
	m, err := u.currentMonitor()
	if err != nil {
		return err
	}
	if monitor == m {
		return nil
	}

	ww := u.origWindowWidthInDIP
	wh := u.origWindowHeightInDIP

	fullscreen, err := u.isFullscreen()
	if err != nil {
		return err
	}
	// This is copied from setFullscreen. They should probably use a shared function.
	if fullscreen {
		if err := u.setFullscreen(false); err != nil {
			return err
		}
		// Just after exiting fullscreen, the window state seems very unstable (#2758).
		// Wait for a while with polling events.
		if runtime.GOOS == "darwin" {
			for i := 0; i < 60; i++ {
				if err := glfw.PollEvents(); err != nil {
					return err
				}
				time.Sleep(time.Second / 60)
			}
		}
	}

	s := monitor.DeviceScaleFactor()
	w := dipToGLFWPixel(float64(ww), s)
	h := dipToGLFWPixel(float64(wh), s)
	mx := monitor.boundsInGLFWPixels.Min.X
	my := monitor.boundsInGLFWPixels.Min.Y
	mw, mh := monitor.sizeInDIP()
	mw = dipToGLFWPixel(mw, s)
	mh = dipToGLFWPixel(mh, s)
	px, py := InitialWindowPosition(int(mw), int(mh), int(w), int(h))
	if err := u.window.SetPos(mx+px, my+py); err != nil {
		return err
	}

	if fullscreen {
		// Calling setFullscreen immediately might not work well, especially on Linux (#2778).
		// Just wait a little bit. 1/30[s] seems enough in most cases.
		time.Sleep(time.Second / 30)
		if err := u.setFullscreen(true); err != nil {
			return err
		}
	}

	return nil
}

func (u *UserInterface) getWindowSizeLimitsInDIP() (minw, minh, maxw, maxh int) {
	if microsoftgdk.IsXbox() {
		return glfw.DontCare, glfw.DontCare, glfw.DontCare, glfw.DontCare
	}

	u.m.RLock()
	defer u.m.RUnlock()
	return u.minWindowWidthInDIP, u.minWindowHeightInDIP, u.maxWindowWidthInDIP, u.maxWindowHeightInDIP
}

func (u *UserInterface) setWindowSizeLimitsInDIP(minw, minh, maxw, maxh int) bool {
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

func (u *UserInterface) isWindowMaximizable() bool {
	_, _, maxw, maxh := u.getWindowSizeLimitsInDIP()
	return maxw == glfw.DontCare && maxh == glfw.DontCare
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

func (u *UserInterface) getInitCursorMode() CursorMode {
	u.m.RLock()
	v := u.initCursorMode
	u.m.RUnlock()
	return v
}

func (u *UserInterface) setInitCursorMode(mode CursorMode) {
	u.m.Lock()
	u.initCursorMode = mode
	u.m.Unlock()
}

func (u *UserInterface) getCursorShape() CursorShape {
	u.m.RLock()
	v := u.cursorShape
	u.m.RUnlock()
	return v
}

func (u *UserInterface) setCursorShape(shape CursorShape) CursorShape {
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

func (u *UserInterface) getAndResetIconImages() []image.Image {
	u.m.RLock()
	defer u.m.RUnlock()
	s := u.iconImages
	u.iconImages = nil
	return s
}

func (u *UserInterface) setIconImages(iconImages []image.Image) {
	u.m.Lock()
	defer u.m.Unlock()

	// Even if iconImages is nil, always create a slice.
	// A 0-size slice and nil are distinguished.
	// See the comment in updateIconIfNeeded.
	u.iconImages = make([]image.Image, len(iconImages))
	copy(u.iconImages, iconImages)
}

func (u *UserInterface) getInitWindowPositionInDIP() (int, int) {
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

func (u *UserInterface) setInitWindowPositionInDIP(x, y int) {
	if microsoftgdk.IsXbox() {
		return
	}

	u.m.Lock()
	defer u.m.Unlock()

	// TODO: Update initMonitor if necessary (#1575).
	u.initWindowPositionXInDIP = x
	u.initWindowPositionYInDIP = y
}

func (u *UserInterface) getInitWindowSizeInDIP() (int, int) {
	if microsoftgdk.IsXbox() {
		return microsoftgdk.MonitorResolution()
	}

	u.m.RLock()
	defer u.m.RUnlock()
	return u.initWindowWidthInDIP, u.initWindowHeightInDIP
}

func (u *UserInterface) setInitWindowSizeInDIP(width, height int) {
	if microsoftgdk.IsXbox() {
		return
	}

	u.m.Lock()
	u.initWindowWidthInDIP, u.initWindowHeightInDIP = width, height
	u.m.Unlock()
}

func (u *UserInterface) isInitWindowFloating() bool {
	if microsoftgdk.IsXbox() {
		return false
	}

	u.m.RLock()
	f := u.initWindowFloating
	u.m.RUnlock()
	return f
}

func (u *UserInterface) setInitWindowFloating(floating bool) {
	if microsoftgdk.IsXbox() {
		return
	}

	u.m.Lock()
	u.initWindowFloating = floating
	u.m.Unlock()
}

func (u *UserInterface) isInitWindowMaximized() bool {
	// TODO: Is this always true on Xbox?
	u.m.RLock()
	m := u.initWindowMaximized
	u.m.RUnlock()
	return m
}

func (u *UserInterface) setInitWindowMaximized(maximized bool) {
	u.m.Lock()
	u.initWindowMaximized = maximized
	u.m.Unlock()
}

func (u *UserInterface) isInitWindowMousePassthrough() bool {
	u.m.RLock()
	defer u.m.RUnlock()
	return u.initWindowMousePassthrough
}

func (u *UserInterface) setInitWindowMousePassthrough(enabled bool) {
	u.m.Lock()
	defer u.m.Unlock()
	u.initWindowMousePassthrough = enabled
}

func (u *UserInterface) isWindowClosingHandled() bool {
	u.m.RLock()
	v := u.windowClosingHandled
	u.m.RUnlock()
	return v
}

func (u *UserInterface) setWindowClosingHandled(handled bool) {
	u.m.Lock()
	u.windowClosingHandled = handled
	u.m.Unlock()

	if !u.isRunning() {
		return
	}
	if u.isTerminated() {
		return
	}
	u.mainThread.Call(func() {
		if u.isTerminated() {
			return
		}
		if err := u.setDocumentEdited(handled); err != nil {
			u.setError(err)
			return
		}
	})
}

// isFullscreen must be called from the main thread.
func (u *UserInterface) isFullscreen() (bool, error) {
	if !u.isRunning() {
		panic("ui: isFullscreen can't be called before the main loop starts")
	}
	m, err := u.window.GetMonitor()
	if err != nil {
		return false, err
	}
	n, err := u.isNativeFullscreen()
	if err != nil {
		return false, err
	}
	return m != nil || n, nil
}

func (u *UserInterface) IsFullscreen() bool {
	if microsoftgdk.IsXbox() {
		return false
	}

	if u.isTerminated() {
		return false
	}
	if !u.isRunning() {
		return u.isInitFullscreen()
	}
	var fullscreen bool
	u.mainThread.Call(func() {
		if u.isTerminated() {
			return
		}
		b, err := u.isFullscreen()
		if err != nil {
			u.setError(err)
			return
		}
		fullscreen = b
	})
	return fullscreen
}

func (u *UserInterface) SetFullscreen(fullscreen bool) {
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
		f, err := u.isFullscreen()
		if err != nil {
			u.setError(err)
			return
		}
		if f == fullscreen {
			return
		}
		if err := u.setFullscreen(fullscreen); err != nil {
			u.setError(err)
			return
		}
	})
}

func (u *UserInterface) IsFocused() bool {
	if !u.isRunning() {
		return false
	}

	var focused bool
	u.mainThread.Call(func() {
		if u.isTerminated() {
			return
		}
		a, err := u.window.GetAttrib(glfw.Focused)
		if err != nil {
			u.setError(err)
			return
		}
		focused = a == glfw.True
	})
	return focused
}

func (u *UserInterface) SetRunnableOnUnfocused(runnableOnUnfocused bool) {
	u.setRunnableOnUnfocused(runnableOnUnfocused)
}

func (u *UserInterface) IsRunnableOnUnfocused() bool {
	return u.isRunnableOnUnfocused()
}

func (u *UserInterface) FPSMode() FPSModeType {
	u.m.Lock()
	defer u.m.Unlock()
	return u.fpsMode
}

func (u *UserInterface) SetFPSMode(mode FPSModeType) {
	if u.isTerminated() {
		return
	}
	if !u.isRunning() {
		u.m.Lock()
		defer u.m.Unlock()
		u.fpsMode = mode
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
		if err := u.setFPSMode(mode); err != nil {
			u.setError(err)
			return
		}
	})
}

func (u *UserInterface) ScheduleFrame() {
	if !u.isRunning() {
		return
	}
	// As the main thread can be blocked, do not check the current FPS mode.
	// PostEmptyEvent is concurrent safe.
	if err := glfw.PostEmptyEvent(); err != nil {
		u.setError(err)
		return
	}
}

func (u *UserInterface) CursorMode() CursorMode {
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
		m, err := u.window.GetInputMode(glfw.CursorMode)
		if err != nil {
			u.setError(err)
			return
		}
		mode = m
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

func (u *UserInterface) SetCursorMode(mode CursorMode) {
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
		if err := u.window.SetInputMode(glfw.CursorMode, driverCursorModeToGLFWCursorMode(mode)); err != nil {
			u.setError(err)
			return
		}
		if mode == CursorModeVisible {
			if err := u.window.SetCursor(glfwSystemCursors[u.getCursorShape()]); err != nil {
				u.setError(err)
				return
			}
		}
	})
}

func (u *UserInterface) CursorShape() CursorShape {
	return u.getCursorShape()
}

func (u *UserInterface) SetCursorShape(shape CursorShape) {
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
		if err := u.window.SetCursor(glfwSystemCursors[shape]); err != nil {
			u.setError(err)
			return
		}
	})
}

// createWindow creates a GLFW window.
//
// createWindow must be called from the main thread.
func (u *UserInterface) createWindow() error {
	if u.window != nil {
		panic("ui: u.window must not exist at createWindow")
	}

	monitor := u.getInitMonitor()
	ww, wh := u.getInitWindowSizeInDIP()
	s := monitor.DeviceScaleFactor()
	width := int(dipToGLFWPixel(float64(ww), s))
	height := int(dipToGLFWPixel(float64(wh), s))
	window, err := glfw.CreateWindow(width, height, "", nil, nil)
	if err != nil {
		return err
	}
	u.window = window
	// Set the running state true just a window is set (#2742).
	u.setRunning(true)

	// The position must be set before the size is set (#1982).
	// setWindowSizeInDIP refers the current monitor's device scale.
	wx, wy := u.getInitWindowPositionInDIP()
	mw, mh := monitor.sizeInDIP()
	if max := int(mw) - ww; wx >= max {
		wx = max
	}
	if max := int(mh) - wh; wy >= max {
		wy = max
	}
	if wx < 0 {
		wx = 0
	}
	if wy < 0 {
		wy = 0
	}
	if err := u.setWindowPositionInDIP(wx, wy, monitor); err != nil {
		return err
	}

	// Though the size is already specified, call setWindowSizeInDIP explicitly to adjust member variables.
	if err := u.setWindowSizeInDIP(ww, wh, true); err != nil {
		return err
	}

	if err := initializeWindowAfterCreation(window); err != nil {
		return err
	}

	// Even just after a window creation, FramebufferSize callback might be invoked (#1847).
	// Ensure to consume this callback.
	if err := u.waitForFramebufferSizeCallback(u.window, nil); err != nil {
		return err
	}

	if err := u.window.SetInputMode(glfw.CursorMode, driverCursorModeToGLFWCursorMode(u.getInitCursorMode())); err != nil {
		return err
	}
	if err := u.window.SetCursor(glfwSystemCursors[u.getCursorShape()]); err != nil {
		return err
	}
	if err := u.window.SetTitle(u.title); err != nil {
		return err
	}
	// Icons are set after every frame. They don't have to be cared here.

	if err := u.updateWindowSizeLimits(); err != nil {
		return err
	}

	u.m.Lock()
	closingHandled := u.windowClosingHandled
	u.m.Unlock()
	if err := u.setDocumentEdited(closingHandled); err != nil {
		return err
	}

	if err := u.afterWindowCreation(); err != nil {
		return err
	}

	return nil
}

// registerWindowCloseCallback must be called from the main thread.
func (u *UserInterface) registerWindowCloseCallback() error {
	if u.closeCallback == nil {
		u.closeCallback = func(_ *glfw.Window) {
			u.m.Lock()
			u.inputState.WindowBeingClosed = true
			u.m.Unlock()

			if !u.isWindowClosingHandled() {
				return
			}
			if err := u.window.Focus(); err != nil {
				u.setError(err)
				return
			}
			if err := u.window.SetShouldClose(false); err != nil {
				u.setError(err)
				return
			}
		}
	}
	if _, err := u.window.SetCloseCallback(u.closeCallback); err != nil {
		return err
	}
	return nil
}

// registerWindowFramebufferSizeCallback must be called from the main thread.
func (u *UserInterface) registerWindowFramebufferSizeCallback() error {
	if u.defaultFramebufferSizeCallback == nil {
		// When the window gets resized (either by manual window resize or a window
		// manager), glfw sends a framebuffer size callback which we need to handle (#1960).
		// This event is the only way to handle the size change at least on i3 window manager.
		u.defaultFramebufferSizeCallback = func(_ *glfw.Window, w, h int) {
			f, err := u.isFullscreen()
			if err != nil {
				u.setError(err)
				return
			}
			if f {
				return
			}
			a, err := u.window.GetAttrib(glfw.Iconified)
			if err != nil {
				u.setError(err)
				return
			}
			if a == glfw.True {
				return
			}

			// The framebuffer size is always scaled by the device scale factor (#1975).
			// See also the implementation in uiContext.updateOffscreen.
			m, err := u.currentMonitor()
			if err != nil {
				u.setError(err)
				return
			}
			s := m.DeviceScaleFactor()
			ww := int(float64(w) / s)
			wh := int(float64(h) / s)
			if err := u.setWindowSizeInDIP(ww, wh, false); err != nil {
				u.setError(err)
				return
			}
		}
	}
	if _, err := u.window.SetFramebufferSizeCallback(u.defaultFramebufferSizeCallback); err != nil {
		return err
	}
	return nil
}

func (u *UserInterface) registerDropCallback() error {
	if u.dropCallback == nil {
		u.dropCallback = func(_ *glfw.Window, names []string) {
			u.m.Lock()
			defer u.m.Unlock()
			u.inputState.DroppedFiles = file.NewVirtualFS(names)
		}
	}
	if _, err := u.window.SetDropCallback(u.dropCallback); err != nil {
		return err
	}
	return nil
}

// waitForFramebufferSizeCallback waits for GLFW's FramebufferSize callback.
// f is a process executed after registering the callback.
// If the callback is not invoked for a while, waitForFramebufferSizeCallback times out and return.
//
// waitForFramebufferSizeCallback must be called from the main thread.
func (u *UserInterface) waitForFramebufferSizeCallback(window *glfw.Window, f func() error) error {
	u.framebufferSizeCallbackCh = make(chan struct{}, 1)

	if u.framebufferSizeCallback == nil {
		u.framebufferSizeCallback = func(_ *glfw.Window, _, _ int) {
			// This callback can be invoked multiple times by one PollEvents in theory (#1618).
			// Allow the case when the channel is full.
			select {
			case u.framebufferSizeCallbackCh <- struct{}{}:
			default:
			}
		}
	}
	if _, err := window.SetFramebufferSizeCallback(u.framebufferSizeCallback); err != nil {
		return err
	}

	if f != nil {
		if err := f(); err != nil {
			return err
		}
	}

	// Use the timeout as FramebufferSize event might not be fired (#1618).
	t := time.NewTimer(100 * time.Millisecond)
	defer t.Stop()

event:
	for {
		if err := glfw.PollEvents(); err != nil {
			return err
		}
		select {
		case <-u.framebufferSizeCallbackCh:
			break event
		case <-t.C:
			break event
		default:
			time.Sleep(time.Millisecond)
		}
	}
	if _, err := window.SetFramebufferSizeCallback(u.defaultFramebufferSizeCallback); err != nil {
		return err
	}

	close(u.framebufferSizeCallbackCh)
	u.framebufferSizeCallbackCh = nil

	return nil
}

func (u *UserInterface) initOnMainThread(options *RunOptions) error {
	if err := glfw.WindowHint(glfw.AutoIconify, glfw.False); err != nil {
		return err
	}

	// Window is shown after the first buffer swap (#2725).
	if err := glfw.WindowHint(glfw.Visible, glfw.False); err != nil {
		return err
	}

	if err := glfw.WindowHintString(glfw.X11ClassName, options.X11ClassName); err != nil {
		return err
	}

	if err := glfw.WindowHintString(glfw.X11InstanceName, options.X11InstanceName); err != nil {
		return err
	}

	// On macOS, window decoration should be initialized once after buffers are swapped (#2600).
	if runtime.GOOS != "darwin" {
		decorated := glfw.False
		if u.isInitWindowDecorated() {
			decorated = glfw.True
		}
		if err := glfw.WindowHint(glfw.Decorated, decorated); err != nil {
			return err
		}
	}

	glfwTransparent := glfw.False
	if options.ScreenTransparent {
		glfwTransparent = glfw.True
	}
	if err := glfw.WindowHint(glfw.TransparentFramebuffer, glfwTransparent); err != nil {
		return err
	}

	g, lib, err := newGraphicsDriver(&graphicsDriverCreatorImpl{
		transparent: options.ScreenTransparent,
		colorSpace:  options.ColorSpace,
	}, options.GraphicsLibrary)
	if err != nil {
		return err
	}
	u.graphicsDriver = g
	u.setGraphicsLibrary(lib)
	u.graphicsDriver.SetTransparent(options.ScreenTransparent)

	// internal/glfw is customized and the default client API is NoAPI, not OpenGLAPI.
	// Then, glfw.WindowHint(glfw.ClientAPI, glfw.NoAPI) doesn't have to be called.

	// Before creating a window, set it unresizable no matter what u.isInitWindowResizable() is (#1987).
	// Making the window resizable here doesn't work correctly when switching to enable resizing.
	resizable := glfw.False
	if u.windowResizingMode == WindowResizingModeEnabled {
		resizable = glfw.True
	}
	if err := glfw.WindowHint(glfw.Resizable, resizable); err != nil {
		return err
	}

	floating := glfw.False
	if u.isInitWindowFloating() {
		floating = glfw.True
	}
	if err := glfw.WindowHint(glfw.Floating, floating); err != nil {
		return err
	}

	u.initUnfocused = options.InitUnfocused
	focused := glfw.True
	if options.InitUnfocused {
		focused = glfw.False
	}
	if err := glfw.WindowHint(glfw.FocusOnShow, focused); err != nil {
		return err
	}

	mousePassthrough := glfw.False
	if u.isInitWindowMousePassthrough() {
		mousePassthrough = glfw.True
	}
	if err := glfw.WindowHint(glfw.MousePassthrough, mousePassthrough); err != nil {
		return err
	}

	// Set the window visible explicitly or the application freezes on Wayland (#974).
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		if err := glfw.WindowHint(glfw.Visible, glfw.True); err != nil {
			return err
		}
	}

	if err := u.createWindow(); err != nil {
		return err
	}

	// Maximizing a window requires a proper size and position. Call Maximize here (#1117).
	if u.isInitWindowMaximized() {
		if err := u.window.Maximize(); err != nil {
			return err
		}
	}

	if err := u.setWindowResizingModeForOS(u.windowResizingMode); err != nil {
		return err
	}

	if options.SkipTaskbar {
		// Ignore the error.
		_ = u.skipTaskbar()
	}

	switch g := u.graphicsDriver.(type) {
	case interface{ SetGLFWWindow(window *glfw.Window) }:
		g.SetGLFWWindow(u.window)
	case interface{ SetWindow(uintptr) }:
		w, err := u.nativeWindow()
		if err != nil {
			return err
		}
		g.SetWindow(w)
	}

	w, err := u.nativeWindow()
	if err != nil {
		return err
	}
	gamepad.SetNativeWindow(w)

	// Register callbacks after the window initialization done.
	// The callback might cause swapping frames, that assumes the window is already set (#2137).
	if err := u.registerWindowCloseCallback(); err != nil {
		return err
	}
	if err := u.registerWindowFramebufferSizeCallback(); err != nil {
		return err
	}
	if err := u.registerInputCallbacks(); err != nil {
		return err
	}
	if err := u.registerDropCallback(); err != nil {
		return err
	}

	return nil
}

func (u *UserInterface) outsideSize() (float64, float64, error) {
	f, err := u.isFullscreen()
	if err != nil {
		return 0, 0, err
	}
	n, err := u.isNativeFullscreen()
	if err != nil {
		return 0, 0, err
	}
	if f && !n {
		// On Linux, the window size is not reliable just after making the window
		// fullscreened. Use the monitor size.
		// On macOS's native fullscreen, the window's size returns a more precise size
		// reflecting the adjustment of the view size (#1745).
		var w, h float64
		m, err := u.currentMonitor()
		if err != nil {
			return 0, 0, err
		}
		if m != nil {
			w, h = m.sizeInDIP()
		}
		return w, h, nil
	}

	a, err := u.window.GetAttrib(glfw.Iconified)
	if err != nil {
		return 0, 0, err
	}
	if a == glfw.True {
		return float64(u.origWindowWidthInDIP), float64(u.origWindowHeightInDIP), nil
	}

	// Instead of u.origWindow{Width,Height}InDIP, use the actual window size here.
	// On Windows, the specified size at SetSize and the actual window size might
	// not match (#1163).
	ww, wh, err := u.window.GetSize()
	if err != nil {
		return 0, 0, err
	}
	m, err := u.currentMonitor()
	if err != nil {
		return 0, 0, err
	}
	s := m.DeviceScaleFactor()
	w := dipFromGLFWPixel(float64(ww), s)
	h := dipFromGLFWPixel(float64(wh), s)
	return w, h, nil
}

// setFPSMode must be called from the main thread.
func (u *UserInterface) setFPSMode(fpsMode FPSModeType) error {
	needUpdate := u.fpsMode != fpsMode || !u.fpsModeInited
	u.fpsMode = fpsMode
	u.fpsModeInited = true

	if !needUpdate {
		return nil
	}

	sticky := glfw.True
	if fpsMode == FPSModeVsyncOffMinimum {
		sticky = glfw.False
	}
	if err := u.window.SetInputMode(glfw.StickyMouseButtonsMode, sticky); err != nil {
		return err
	}
	if err := u.window.SetInputMode(glfw.StickyKeysMode, sticky); err != nil {
		return err
	}

	vsyncEnabled := u.fpsMode == FPSModeVsyncOn
	graphicscommand.SetVsyncEnabled(vsyncEnabled, u.graphicsDriver)

	return nil
}

// update must be called from the main thread.
func (u *UserInterface) update() (float64, float64, error) {
	if err := u.error(); err != nil {
		return 0, 0, err
	}

	sc, err := u.window.ShouldClose()
	if err != nil {
		return 0, 0, err
	}
	if sc {
		return 0, 0, RegularTermination
	}

	// On macOS, one swapping buffers seems required before entering fullscreen (#2599).
	if u.isInitFullscreen() && (u.bufferOnceSwapped || runtime.GOOS != "darwin") {
		if err := u.setFullscreen(true); err != nil {
			return 0, 0, err
		}
		u.setInitFullscreen(false)
	}

	if runtime.GOOS == "darwin" && u.bufferOnceSwapped {
		var err error
		u.darwinInitOnce.Do(func() {
			// On macOS, window decoration should be initialized once after buffers are swapped (#2600).
			decorated := glfw.False
			if u.isInitWindowDecorated() {
				decorated = glfw.True
			}
			if err = u.window.SetAttrib(glfw.Decorated, decorated); err != nil {
				return
			}
		})
		if err != nil {
			return 0, 0, err
		}
	}

	if u.bufferOnceSwapped {
		var err error
		u.showWindowOnce.Do(func() {
			// Show the window after first buffer swap to avoid flash of white especially on Windows.
			if err = u.window.Show(); err != nil {
				return
			}
			if !u.initUnfocused {
				if err = u.window.Focus(); err != nil {
					return
				}
			}

			if runtime.GOOS == "darwin" || runtime.GOOS == "windows" {
				return
			}

			// On Linux or UNIX, there is a problematic desktop environment like i3wm
			// where an invisible window size cannot be initialized correctly (#2951).
			// Call SetSize explicitly after the window becomes visible.

			fullscreen, e := u.isFullscreen()
			if e != nil {
				err = e
				return
			}
			if fullscreen {
				return
			}

			m, e := u.currentMonitor()
			if e != nil {
				err = e
				return
			}
			s := m.DeviceScaleFactor()
			newW := int(dipToGLFWPixel(float64(u.origWindowWidthInDIP), s))
			newH := int(dipToGLFWPixel(float64(u.origWindowHeightInDIP), s))

			// Even though a framebuffer callback is not called, waitForFramebufferSizeCallback returns by timeout,
			// so it is safe to use this.
			if err = u.waitForFramebufferSizeCallback(u.window, func() error {
				return u.window.SetSize(newW, newH)
			}); err != nil {
				return
			}
		})
		if err != nil {
			return 0, 0, err
		}
	}

	// Initialize vsync after SetMonitor is called.
	// Calling this inside setWindowSize didn't work (#1363).
	// Also, setFPSMode has to be called after graphicscommand.SetRenderThread is called (#2714).
	if !u.fpsModeInited {
		if err := u.setFPSMode(u.fpsMode); err != nil {
			return 0, 0, err
		}
	}

	if u.fpsMode != FPSModeVsyncOffMinimum {
		// TODO: Updating the input can be skipped when clock.Update returns 0 (#1367).
		if err := glfw.PollEvents(); err != nil {
			return 0, 0, err
		}
	} else {
		if err := glfw.WaitEvents(); err != nil {
			return 0, 0, err
		}
	}

	// If isRunnableOnUnfocused is false and the window is not focused, wait here.
	// For the first update, skip this check as the window might not be seen yet in some environments like ChromeOS (#3091).
	for !u.isRunnableOnUnfocused() && u.bufferOnceSwapped {
		// In the initial state on macOS, the window is not shown (#2620).
		visible, err := u.window.GetAttrib(glfw.Visible)
		if err != nil {
			return 0, 0, err
		}
		if visible == glfw.False {
			break
		}

		focused, err := u.window.GetAttrib(glfw.Focused)
		if err != nil {
			return 0, 0, err
		}
		if focused != glfw.False {
			break
		}

		shouldClose, err := u.window.ShouldClose()
		if err != nil {
			return 0, 0, err
		}
		if shouldClose {
			break
		}

		if err := hook.SuspendAudio(); err != nil {
			return 0, 0, err
		}
		// Wait for an arbitrary period to avoid busy loop.
		time.Sleep(time.Second / 60)
		if err := glfw.PollEvents(); err != nil {
			return 0, 0, err
		}
	}

	if err := hook.ResumeAudio(); err != nil {
		return 0, 0, err
	}

	return u.outsideSize()
}

func (u *UserInterface) loopGame() (ferr error) {
	defer func() {
		graphicscommand.Terminate()
		u.mainThread.Call(func() {
			if err := glfw.Terminate(); err != nil {
				ferr = err
			}
			u.setTerminated()
		})
	}()

	for {
		if err := u.updateGame(); err != nil {
			return err
		}
	}
}

func (u *UserInterface) updateGame() error {
	var unfocused bool

	// On Windows, the focusing state might be always false (#987).
	// On Windows, even if a window is in another workspace, vsync seems to work.
	// Then let's assume the window is always 'focused' as a workaround.
	if runtime.GOOS != "windows" {
		a, err := u.window.GetAttrib(glfw.Focused)
		if err != nil {
			return err
		}
		unfocused = a == glfw.False
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
		if err != nil {
			return
		}
		m, err := u.currentMonitor()
		if err != nil {
			return
		}
		deviceScaleFactor = m.DeviceScaleFactor()
	}); err != nil {
		return err
	}

	if err := u.context.updateFrame(u.graphicsDriver, outsideWidth, outsideHeight, deviceScaleFactor, u); err != nil {
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

func (u *UserInterface) updateIconIfNeeded() error {
	// In the fullscreen mode, SetIcon fails (#1578).
	f, err := u.isFullscreen()
	if err != nil {
		return err
	}
	if f {
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
	if err := u.error(); err != nil {
		return err
	}

	u.mainThread.Call(func() {
		err = u.window.SetIcon(newImgs)
	})
	if err != nil {
		return err
	}

	return nil
}

// updateWindowSizeLimits must be called from the main thread.
func (u *UserInterface) updateWindowSizeLimits() error {
	m, err := u.currentMonitor()
	if err != nil {
		return err
	}
	minw, minh, maxw, maxh := u.getWindowSizeLimitsInDIP()

	s := m.DeviceScaleFactor()
	if minw < 0 {
		// Always set the minimum window width.
		mw, err := u.minimumWindowWidth()
		if err != nil {
			return err
		}
		minw = int(dipToGLFWPixel(float64(mw), s))
	} else {
		minw = int(dipToGLFWPixel(float64(minw), s))
	}
	if minh < 0 {
		minh = glfw.DontCare
	} else {
		minh = int(dipToGLFWPixel(float64(minh), s))
	}
	if maxw < 0 {
		maxw = glfw.DontCare
	} else {
		maxw = int(dipToGLFWPixel(float64(maxw), s))
	}
	if maxh < 0 {
		maxh = glfw.DontCare
	} else {
		maxh = int(dipToGLFWPixel(float64(maxh), s))
	}
	if err := u.window.SetSizeLimits(minw, minh, maxw, maxh); err != nil {
		return err
	}

	// The window size limit affects the resizing mode, especially on macOS (#2260).
	if err := u.setWindowResizingModeForOS(u.windowResizingMode); err != nil {
		return err
	}

	return nil
}

// disableWindowSizeLimits disables a window size limitation temporarily, especially for fullscreen
// In order to enable the size limitation, call updateWindowSizeLimits.
//
// disableWindowSizeLimits must be called from the main thread.
func (u *UserInterface) disableWindowSizeLimits() error {
	return u.window.SetSizeLimits(glfw.DontCare, glfw.DontCare, glfw.DontCare, glfw.DontCare)
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
func (u *UserInterface) setWindowSizeInDIP(width, height int, callSetSize bool) error {
	if microsoftgdk.IsXbox() {
		// Do nothing. The size is always fixed.
		return nil
	}

	width, height = u.adjustWindowSizeBasedOnSizeLimitsInDIP(width, height)
	m, err := u.minimumWindowWidth()
	if err != nil {
		return err
	}
	if width < m {
		width = m
	}
	if height < 1 {
		height = 1
	}

	mon, err := u.currentMonitor()
	if err != nil {
		return err
	}
	scale := mon.DeviceScaleFactor()
	if u.origWindowWidthInDIP == width && u.origWindowHeightInDIP == height && u.lastDeviceScaleFactor == scale {
		return nil
	}
	u.lastDeviceScaleFactor = scale

	u.origWindowWidthInDIP = width
	u.origWindowHeightInDIP = height

	f, err := u.isFullscreen()
	if err != nil {
		return err
	}
	if !f && callSetSize {
		// Set the window size after the position. The order matters.
		// In the opposite order, the window size might not be correct when going back from fullscreen with multi monitors.
		oldW, oldH, err := u.window.GetSize()
		if err != nil {
			return err
		}
		m, err := u.currentMonitor()
		if err != nil {
			return err
		}
		s := m.DeviceScaleFactor()
		newW := int(dipToGLFWPixel(float64(width), s))
		newH := int(dipToGLFWPixel(float64(height), s))
		if oldW != newW || oldH != newH {
			// Just after SetSize, GetSize is not reliable especially on Linux/UNIX.
			// Let's wait for FramebufferSize callback in any cases.
			if err := u.waitForFramebufferSizeCallback(u.window, func() error {
				return u.window.SetSize(newW, newH)
			}); err != nil {
				return err
			}
		}
	}

	if err := u.updateWindowSizeLimits(); err != nil {
		return err
	}
	return nil
}

// setOrigWindowPosWithCurrentPos must be called from the main thread.
func (u *UserInterface) setOrigWindowPosWithCurrentPos() error {
	if x, y := u.origWindowPos(); x == invalidPos || y == invalidPos {
		x, y, err := u.window.GetPos()
		if err != nil {
			return err
		}
		u.setOrigWindowPos(x, y)
	}
	return nil
}

// setFullscreen must be called from the main thread.
func (u *UserInterface) setFullscreen(fullscreen bool) error {
	f, err := u.isFullscreen()
	if err != nil {
		return err
	}
	if f == fullscreen {
		return nil
	}

	im, err := u.window.GetInputMode(glfw.CursorMode)
	if err != nil {
		return err
	}
	if im == glfw.CursorDisabled {
		u.saveCursorPosition()
	}

	// Enter the fullscreen.
	if fullscreen {
		if err := u.disableWindowSizeLimits(); err != nil {
			return err
		}

		if x, y := u.origWindowPos(); x == invalidPos || y == invalidPos {
			x, y, err := u.window.GetPos()
			if err != nil {
				return err
			}
			u.setOrigWindowPos(x, y)
		}

		if u.isNativeFullscreenAvailable() {
			if err := u.setNativeFullscreen(fullscreen); err != nil {
				return err
			}
		} else {
			m, err := u.currentMonitor()
			if err != nil {
				return err
			}
			if m == nil {
				return nil
			}

			vm := m.videoMode
			if err := u.window.SetMonitor(m.m, 0, 0, vm.Width, vm.Height, vm.RefreshRate); err != nil {
				return err
			}
		}
		if err := u.adjustViewSizeAfterFullscreen(); err != nil {
			return err
		}
		return nil
	}

	// Exit the fullscreen.
	if err := u.updateWindowSizeLimits(); err != nil {
		return err
	}

	// Get the original window position and size before changing the state of fullscreen.
	// TODO: Why?
	origX, origY := u.origWindowPos()

	m, err := u.currentMonitor()
	if err != nil {
		return err
	}
	s := m.DeviceScaleFactor()
	ww := int(dipToGLFWPixel(float64(u.origWindowWidthInDIP), s))
	wh := int(dipToGLFWPixel(float64(u.origWindowHeightInDIP), s))
	if u.isNativeFullscreenAvailable() {
		if err := u.setNativeFullscreen(false); err != nil {
			return err
		}
		// Adjust the window size later (after adjusting the position).
	} else {
		m, err := u.window.GetMonitor()
		if err != nil {
			return err
		}
		if !u.isNativeFullscreenAvailable() && m != nil {
			if err := u.window.SetMonitor(nil, 0, 0, ww, wh, 0); err != nil {
				return err
			}
		}
	}

	// glfw.PollEvents is necessary for macOS to enable (*glfw.Window).SetPos and SetSize (#2296).
	// This polling causes issues on Linux and Windows when rapidly toggling fullscreen, so we only run it under macOS.
	if runtime.GOOS == "darwin" {
		if err := glfw.PollEvents(); err != nil {
			return err
		}
	}

	if origX != invalidPos && origY != invalidPos {
		if err := u.window.SetPos(origX, origY); err != nil {
			return err
		}
		// Dirty hack for macOS (#703). Rendering doesn't work correctly with one SetPos, but
		// work with two or more SetPos.
		if runtime.GOOS == "darwin" {
			if err := u.window.SetPos(origX+1, origY); err != nil {
				return err
			}
			if err := u.window.SetPos(origX, origY); err != nil {
				return err
			}
		}
		u.setOrigWindowPos(invalidPos, invalidPos)
	}

	if u.isNativeFullscreenAvailable() {
		// Set the window size after the position. The order matters.
		// In the opposite order, the window size might not be correct when going back from fullscreen with multi monitors.
		if err := u.window.SetSize(ww, wh); err != nil {
			return err
		}
	}

	return nil
}

func (u *UserInterface) minimumWindowWidth() (int, error) {
	a, err := u.window.GetAttrib(glfw.Decorated)
	if err != nil {
		return 0, err
	}
	if a == glfw.False {
		return 1, nil
	}

	// On Windows, giving a too small width doesn't call a callback (#165).
	// To prevent hanging up, return asap if the width is too small.
	// 126 is an arbitrary number and I guess this is small enough .
	if runtime.GOOS == "windows" {
		return 126, nil
	}

	// On macOS, resizing the window by cursor sometimes ignores the minimum size.
	// To avoid the flaky behavior, do not add a limitation.
	return 1, nil
}

// currentMonitor returns the current active monitor.
//
// currentMonitor must be called on the main thread.
func (u *UserInterface) currentMonitor() (*Monitor, error) {
	if u.window == nil {
		return u.getInitMonitor(), nil
	}

	// Getting a monitor from a window position is not reliable in general (e.g., when a window is put across
	// multiple monitors, or, before SetWindowPosition is called.).
	// Get the monitor which the current window belongs to. This requires OS API.
	m, err := monitorFromWindowByOS(u.window)
	if err != nil {
		return nil, err
	}
	if m != nil {
		return m, nil
	}

	// As the fallback, detect the monitor from the window.
	x, y, err := u.window.GetPos()
	if err != nil {
		return nil, err
	}
	// On fullscreen, shift the position slightly. Otherwise, a wrong monitor could be detected, as the position is on the edge (#2794).
	f, err := u.isFullscreen()
	if err != nil {
		return nil, err
	}
	if f {
		x++
		y++
	}
	if m := theMonitors.monitorFromPosition(x, y); m != nil {
		return m, nil
	}

	return theMonitors.primaryMonitor(), nil
}

func (u *UserInterface) readInputState(inputState *InputState) {
	u.m.Lock()
	defer u.m.Unlock()
	u.inputState.copyAndReset(inputState)
}

func (u *UserInterface) Window() Window {
	if microsoftgdk.IsXbox() {
		return &nullWindow{}
	}
	return &u.iwindow
}

// GLFW's functions to manipulate a window can invoke the SetSize callback (#1576, #1585, #1606).
// As the callback must not be called in the frame (between BeginFrame and EndFrame),
// disable the callback temporarily.

// maximizeWindow must be called from the main thread.
func (u *UserInterface) maximizeWindow() error {
	n, err := u.isNativeFullscreen()
	if err != nil {
		return err
	}
	if n {
		return nil
	}

	f, err := u.isFullscreen()
	if err != nil {
		return err
	}
	if f {
		return nil
	}

	if err := u.window.Maximize(); err != nil {
		return err
	}

	// On Linux/UNIX, maximizing might not finish even though Maximize returns. Just wait for its finish.
	// Do not check this in the fullscreen since apparently the condition can never be true.
	for {
		a, err := u.window.GetAttrib(glfw.Maximized)
		if err != nil {
			return err
		}
		if a == glfw.True {
			break
		}
		if err := glfw.PollEvents(); err != nil {
			return err
		}
	}

	return nil
}

// iconifyWindow must be called from the main thread.
func (u *UserInterface) iconifyWindow() error {
	// Iconifying a native fullscreen window on macOS is forbidden.
	n, err := u.isNativeFullscreen()
	if err != nil {
		return err
	}
	if n {
		return nil
	}

	if err := u.window.Iconify(); err != nil {
		return err
	}

	// On Linux/UNIX, iconifying might not finish even though Iconify returns. Just wait for its finish.
	for {
		a, err := u.window.GetAttrib(glfw.Iconified)
		if err != nil {
			return err
		}
		if a == glfw.True {
			break
		}
		if err := glfw.PollEvents(); err != nil {
			return err
		}
	}

	return nil
}

// restoreWindow must be called from the main thread.
func (u *UserInterface) restoreWindow() error {
	if err := u.window.Restore(); err != nil {
		return err
	}

	// On Linux/UNIX, restoring might not finish even though Restore returns (#1608). Just wait for its finish.
	// On macOS, the restoring state might be the same as the maximized state. Skip this.
	if runtime.GOOS != "darwin" {
		for {
			maximized, err := u.window.GetAttrib(glfw.Maximized)
			if err != nil {
				return err
			}
			iconified, err := u.window.GetAttrib(glfw.Iconified)
			if err != nil {
				return err
			}
			if maximized == glfw.False && iconified == glfw.False {
				break
			}
			if err := glfw.PollEvents(); err != nil {
				return err
			}
			time.Sleep(time.Second / 60)
		}
	}

	return nil
}

// setWindowDecorated must be called from the main thread.
func (u *UserInterface) setWindowDecorated(decorated bool) error {
	if microsoftgdk.IsXbox() {
		return nil
	}

	v := glfw.False
	if decorated {
		v = glfw.True
	}
	if err := u.window.SetAttrib(glfw.Decorated, v); err != nil {
		return err
	}

	// The title can be lost when the decoration is gone. Recover this.
	if decorated {
		if err := u.window.SetTitle(u.title); err != nil {
			return err
		}
	}

	return nil
}

// setWindowFloating must be called from the main thread.
func (u *UserInterface) setWindowFloating(floating bool) error {
	if microsoftgdk.IsXbox() {
		return nil
	}

	v := glfw.False
	if floating {
		v = glfw.True
	}
	if err := u.window.SetAttrib(glfw.Floating, v); err != nil {
		return err
	}

	return nil
}

// setWindowResizingMode must be called from the main thread.
func (u *UserInterface) setWindowResizingMode(mode WindowResizingMode) error {
	if microsoftgdk.IsXbox() {
		return nil
	}

	if u.windowResizingMode == mode {
		return nil
	}

	u.windowResizingMode = mode

	v := glfw.False
	if mode == WindowResizingModeEnabled {
		v = glfw.True
	}
	if err := u.window.SetAttrib(glfw.Resizable, v); err != nil {
		return err
	}
	if err := u.setWindowResizingModeForOS(mode); err != nil {
		return err
	}

	return nil
}

// setWindowPositionInDIP sets the window position.
//
// x and y are the position in device-independent pixels.
//
// setWindowPositionInDIP must be called from the main thread.
func (u *UserInterface) setWindowPositionInDIP(x, y int, monitor *Monitor) error {
	if microsoftgdk.IsXbox() {
		// Do nothing. The position is always fixed.
		return nil
	}

	f, err := u.isFullscreen()
	if err != nil {
		return err
	}

	mx := monitor.boundsInGLFWPixels.Min.X
	my := monitor.boundsInGLFWPixels.Min.Y
	s := monitor.DeviceScaleFactor()
	xf := dipToGLFWPixel(float64(x), s)
	yf := dipToGLFWPixel(float64(y), s)
	if x, y := u.adjustWindowPosition(mx+int(xf), my+int(yf), monitor); f {
		u.setOrigWindowPos(x, y)
	} else {
		if err := u.window.SetPos(x, y); err != nil {
			return err
		}
	}

	return nil
}

// setWindowTitle must be called from the main thread.
func (u *UserInterface) setWindowTitle(title string) error {
	return u.window.SetTitle(title)
}

// isWindowMaximized must be called from the main thread.
func (u *UserInterface) isWindowMaximized() (bool, error) {
	a, err := u.window.GetAttrib(glfw.Maximized)
	if err != nil {
		return false, err
	}
	n, err := u.isNativeFullscreen()
	if err != nil {
		return false, err
	}
	return a == glfw.True && !n, nil
}

func (u *UserInterface) origWindowPos() (int, int) {
	return u.origWindowPosX, u.origWindowPosY
}

func (u *UserInterface) setOrigWindowPos(x, y int) {
	u.origWindowPosX = x
	u.origWindowPosY = y
}

// setWindowMousePassthrough must be called from the main thread.
func (u *UserInterface) setWindowMousePassthrough(enabled bool) error {
	if microsoftgdk.IsXbox() {
		return nil
	}

	v := glfw.False
	if enabled {
		v = glfw.True
	}
	if err := u.window.SetAttrib(glfw.MousePassthrough, v); err != nil {
		return err
	}
	return nil
}

func IsScreenTransparentAvailable() bool {
	return true
}

func (u *UserInterface) RunOnMainThread(f func()) {
	u.mainThread.Call(f)
}

func dipToNativePixels(x float64, scale float64) float64 {
	return dipToGLFWPixel(x, scale)
}
