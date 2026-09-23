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
	stdcontext "context"
	"errors"
	"fmt"
	"image"
	"math"
	"runtime"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/hajimehoshi/ebiten/v2/internal/clock"
	"github.com/hajimehoshi/ebiten/v2/internal/file"
	"github.com/hajimehoshi/ebiten/v2/internal/gamepad"
	"github.com/hajimehoshi/ebiten/v2/internal/glfw"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicscommand"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/opengl"
	"github.com/hajimehoshi/ebiten/v2/internal/hook"
	"github.com/hajimehoshi/ebiten/v2/internal/microsoftgdk"
	"github.com/hajimehoshi/ebiten/v2/internal/thread"
	"github.com/hajimehoshi/ebiten/v2/internal/windowsystem"
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

type glfwBackend struct {
	*UserInterface

	window *glfw.Window

	lastDeviceScaleFactor float64

	// These caches are accessed only on the main thread.
	layoutSizeCache            windowPropertyCache[windowSizes]
	nativeFullscreenCache      windowPropertyCache[bool]
	nativeFullscreenTransition bool
	occlusionCache             windowPropertyCache[bool]

	initUnfocused bool

	// bufferOnceSwapped must be accessed from the main thread.
	bufferOnceSwapped bool

	// lastFrameMonitor is the monitor the game was presented on at the previous frame.
	// lastFrameMonitor must be accessed from the main thread.
	lastFrameMonitor *Monitor

	// pollingEvents reports whether the main thread is polling events for the game loop.
	// pollingEvents must be accessed from the main thread.
	pollingEvents bool

	// forcingFrame reports whether a frame is being rendered from an event callback.
	// forcingFrame must be accessed from the main thread.
	forcingFrame bool

	// windowToRestore is the window's position and size in GLFW pixels, captured on entering
	// fullscreen and restored on leaving it. Its members are invalidPos and invalidSize when
	// nothing is captured.
	windowToRestore struct {
		pos  image.Point
		size image.Point

		// monitor is the monitor the window was on when size was captured.
		monitor *Monitor
	}

	// windowWidthInDIP and windowHeightInDIP are the window size in device-independent pixels,
	// as it was requested. Converting a pixel size back does not return the requested size at a
	// fractional scale factor (#2978), and while the window is iconified there is no pixel size
	// to convert at all: the window reports no client area on Windows, and windowToRestore is
	// captured only while fullscreen.
	// While the window is fullscreen, a resize of the window itself does not update them.
	// An explicit request does, and it also updates windowToRestore.
	windowWidthInDIP  int
	windowHeightInDIP int

	// windowXInDIP and windowYInDIP are the window position relative to its monitor, in
	// device-independent pixels, as it was requested. Converting a pixel position back does not
	// return the requested position at a fractional scale factor (#2978).
	// While the window is fullscreen, a move of the window itself does not update them.
	// An explicit request does, and it also updates windowToRestore.
	windowXInDIP int
	windowYInDIP int

	fpsModeInited bool

	input         glfwInput
	backendWindow glfwWindow

	unfocusedNextWake time.Time

	closeCallback                  glfw.CloseCallback
	posCallback                    glfw.PosCallback
	sizeCallback                   glfw.SizeCallback
	contentScaleCallback           glfw.ContentScaleCallback
	framebufferSizeCallback        glfw.FramebufferSizeCallback
	defaultFramebufferSizeCallback glfw.FramebufferSizeCallback
	dropCallback                   glfw.DropCallback
	framebufferSizeCallbackCh      chan struct{}

	cachedCurrentMonitor     *Monitor
	cachedCurrentMonitorTime int64

	// Window states are updated by callbacks and native operations, with periodic queries
	// to recover from missed notifications (#3318). Access is confined to the main thread.
	focusedCache   windowPropertyCache[bool]
	visibleCache   windowPropertyCache[bool]
	iconifiedCache windowPropertyCache[bool]

	focusCallback   glfw.FocusCallback
	iconifyCallback glfw.IconifyCallback

	showWindowOnce        sync.Once
	bufferOnceSwappedOnce sync.Once

	// immContext is used only on Windows.
	immContext uintptr
}

const (
	invalidPos  = math.MinInt
	invalidSize = math.MinInt
)

func init() {
	// Lock the main thread.
	runtime.LockOSThread()
}

// maybeNewGLFWBackend returns a glfw backend, or nil where there is no window
// system for it to use.
func maybeNewGLFWBackend(u *UserInterface) *glfwBackend {
	if !windowsystem.Available() {
		return nil
	}
	return newGLFWBackend(u)
}

func newGLFWBackend(u *UserInterface) *glfwBackend {
	b := &glfwBackend{
		UserInterface: u,
	}
	b.windowToRestore.pos = image.Pt(invalidPos, invalidPos)
	b.windowToRestore.size = image.Pt(invalidSize, invalidSize)
	b.input.clearSavedCursorPos()
	b.backendWindow.ui = b
	return b
}

var glfwSystemCursors = map[CursorShape]*glfw.Cursor{}

func (u *UserInterface) initializeGLFW() error {
	if err := glfw.Init(); err != nil {
		return err
	}

	// Update the monitor first. The monitor state is depended on by various functions like initialMonitorByOS.
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

	u.requestedMonitor.init(m)

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

// ensureGLFWInit lazily initializes GLFW and related state on the first call.
// This is safe to call multiple times; initialization happens only once.
func (u *UserInterface) ensureGLFWInit() error {
	u.glfwInitOnce.Do(func() {
		if err := u.initializePlatform(); err != nil {
			u.setError(err)
			return
		}
		if err := u.initializeGLFW(); err != nil {
			u.setError(err)
			return
		}
		if _, err := glfw.SetMonitorCallback(func(monitor *glfw.Monitor, event glfw.PeripheralEvent) {
			if b, ok := u.runningBackend().(*glfwBackend); ok {
				b.layoutSizeCache.invalidate()
			}
			if err := theMonitors.update(); err != nil {
				u.setError(err)
			}
		}); err != nil {
			u.setError(err)
			return
		}
	})
	return u.error()
}

// Monitor returns the window's current monitor.
func (u *glfwBackend) Monitor() *Monitor {
	if p := u.requestedMonitor.pending(); p != nil {
		return *p
	}
	return thread.CallWithArgAndResult(u.mainThread, func(u *glfwBackend) *Monitor {
		if u.isTerminated() {
			return nil
		}
		m, err := u.currentMonitor()
		if err != nil {
			u.setError(err)
			return nil
		}
		return m
	}, u)
}

// setWindowMonitor must be called on the main thread.
func (u *glfwBackend) setWindowMonitor(monitor *Monitor) error {
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

	ww := u.windowWidthInDIP
	wh := u.windowHeightInDIP

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
			for range 60 {
				if err := glfw.PollEvents(); err != nil {
					return err
				}
				time.Sleep(time.Second / 60)
			}
		}
	}

	s := monitor.DeviceScaleFactor()
	w, h := windowSizeInGLFWPixels(ww, wh, s)
	mx := monitor.boundsInGLFWPixels.Min.X
	my := monitor.boundsInGLFWPixels.Min.Y
	mw, mh := monitor.sizeInDIP()
	mwInGLFWPixels := int(math.Round(dipToGLFWPixel(mw, s)))
	mhInGLFWPixels := int(math.Round(dipToGLFWPixel(mh, s)))
	px, py := InitialWindowPosition(mwInGLFWPixels, mhInGLFWPixels, w, h)
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

// isWindowedFullscreen reports whether the window is in GLFW's fullscreen, which covers a monitor
// without using the platform's native fullscreen.
//
// isWindowedFullscreen must be called from the main thread.
func (u *glfwBackend) isWindowedFullscreen() (bool, error) {
	m, err := u.window.GetMonitor()
	if err != nil {
		return false, err
	}
	return m != nil, nil
}

// isFullscreen reports whether the window is fullscreen, either in windowed fullscreen or in native
// fullscreen.
//
// isFullscreen must be called from the main thread.
func (u *glfwBackend) isFullscreen() (bool, error) {
	if !u.isRunning() {
		panic("ui: isFullscreen can't be called before the main loop starts")
	}
	wf, err := u.isWindowedFullscreen()
	if err != nil {
		return false, err
	}
	nf, err := u.isNativeFullscreen()
	if err != nil {
		return false, err
	}
	return wf || nf, nil
}

func (u *glfwBackend) IsFullscreen() bool {
	return thread.CallWithArgAndResult(u.mainThread, func(u *glfwBackend) bool {
		if u.isTerminated() {
			return false
		}
		b, err := u.isFullscreen()
		if err != nil {
			u.setError(err)
			return false
		}
		return b
	}, u)
}

func (u *glfwBackend) SetFullscreen(fullscreen bool) {
	type args struct {
		u          *glfwBackend
		fullscreen bool
	}
	thread.CallWithArg(u.mainThread, func(a args) {
		u, fullscreen := a.u, a.fullscreen
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
	}, args{u: u, fullscreen: fullscreen})
}

func (u *glfwBackend) IsFocused() bool {
	return thread.CallWithArgAndResult(u.mainThread, func(u *glfwBackend) bool {
		if u.isTerminated() {
			return false
		}
		f, err := u.isWindowFocused()
		if err != nil {
			u.setError(err)
			return false
		}
		return f
	}, u)
}

func (u *glfwBackend) applyFPSMode() {
	thread.CallWithArg(u.mainThread, func(u *glfwBackend) {
		if u.isTerminated() {
			return
		}
		if err := u.setFPSMode(FPSModeType(u.fpsMode.Load())); err != nil {
			u.setError(err)
			return
		}
	}, u)
}

func (u *glfwBackend) ScheduleFrame() {
	// This check can slip past a termination running on the main thread, and then
	// PostEmptyEvent touches GLFW's state after glfw.Terminate. PostEmptyEvent must stay
	// harmless in that case.
	if u.isTerminated() {
		return
	}

	// As the main thread can be blocked, do not check the current FPS mode.
	// PostEmptyEvent is concurrent safe.
	if err := glfw.PostEmptyEvent(); err != nil {
		u.setError(err)
		return
	}
}

func (u *glfwBackend) CursorMode() CursorMode {
	return thread.CallWithArgAndResult(u.mainThread, func(u *glfwBackend) CursorMode {
		if u.isTerminated() {
			return 0
		}
		mode, err := u.window.GetInputMode(glfw.CursorMode)
		if err != nil {
			u.setError(err)
			return 0
		}
		switch mode {
		case glfw.CursorNormal:
			return CursorModeVisible
		case glfw.CursorHidden:
			return CursorModeHidden
		case glfw.CursorDisabled:
			return CursorModeCaptured
		default:
			panic(fmt.Sprintf("ui: invalid GLFW cursor mode: %d", mode))
		}
	}, u)
}

func (u *glfwBackend) SetCursorMode(mode CursorMode) {
	type args struct {
		u    *glfwBackend
		mode CursorMode
	}
	thread.CallWithArg(u.mainThread, func(a args) {
		u, mode := a.u, a.mode
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
	}, args{u: u, mode: mode})
}

func (u *glfwBackend) applyCursorShape() {
	thread.CallWithArg(u.mainThread, func(u *glfwBackend) {
		if u.isTerminated() {
			return
		}
		if err := u.window.SetCursor(glfwSystemCursors[u.getCursorShape()]); err != nil {
			u.setError(err)
			return
		}
	}, u)
}

// createWindow creates a GLFW window.
//
// createWindow must be called from the main thread.
func (u *glfwBackend) createWindow() error {
	if u.window != nil {
		panic("ui: u.window must not exist at createWindow")
	}

	monitor := u.getRequestedMonitor()
	ww, wh := u.desktopWindow.getWindowSizeInDIP()
	s := monitor.DeviceScaleFactor()
	width, height := windowSizeInGLFWPixels(ww, wh, s)
	window, err := glfw.CreateWindow(width, height, "", nil, nil)
	if err != nil {
		return err
	}
	u.window = window
	// Publish the backend and set the running state true just as a window is set (#2742).
	u.setRunningBackend(u)

	monitorRequest := u.requestedMonitor.value.Load()
	monitor = *monitorRequest
	sizeRequest := u.desktopWindow.windowSizeInDIP.value.Load()
	ww, wh = sizeRequest.X, sizeRequest.Y

	// The position must be set before the size is set (#1982).
	// setWindowSizeInDIP refers to the current monitor's device scale.
	positionRequest := u.desktopWindow.windowPositionInDIP.value.Load()
	wx, wy := invalidPos, invalidPos
	if positionRequest != nil {
		wx, wy = positionRequest.X, positionRequest.Y
	}
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
	if err := u.setWindowPositionInDIP(wx, wy, monitor, true); err != nil {
		return err
	}

	// Though the size is already specified, call setWindowSizeInDIP explicitly to adjust member variables.
	if err := u.setWindowSizeInDIP(ww, wh, true); err != nil {
		return err
	}
	u.requestedMonitor.markApplied(monitorRequest)
	u.desktopWindow.windowPositionInDIP.markApplied(positionRequest)
	u.desktopWindow.windowSizeInDIP.markApplied(sizeRequest)

	if err := u.initializeWindowAfterCreation(window); err != nil {
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
	if err := u.window.SetTitle(u.desktopWindow.title.Load()); err != nil {
		return err
	}
	// Icons are set after every frame. They don't have to be cared for here.

	if err := u.updateWindowSizeLimits(); err != nil {
		return err
	}

	if err := u.setDocumentEdited(u.desktopWindow.windowClosingHandled.Load()); err != nil {
		return err
	}

	if err := u.afterWindowCreation(); err != nil {
		return err
	}

	return nil
}

// registerWindowCloseCallback must be called from the main thread.
func (u *glfwBackend) registerWindowCloseCallback() error {
	if u.closeCallback == nil {
		u.closeCallback = func(_ *glfw.Window) {
			u.input.setWindowBeingClosed()

			if !u.desktopWindow.isWindowClosingHandled() {
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

// registerWindowPosCallback must be called from the main thread.
func (u *glfwBackend) registerWindowPosCallback() error {
	if u.posCallback == nil {
		u.posCallback = func(_ *glfw.Window, x, y int) {
			u.layoutSizeCache.invalidate()
			f, err := u.isFullscreen()
			if err != nil {
				u.setError(err)
				return
			}
			if f {
				return
			}
			// The iconification must not be cached here: on Windows, WM_MOVE can arrive before
			// the WM_SIZE which reports the iconification, so the cached value can be stale in
			// both directions. Then the iconified window position would be stored as the window
			// position, or the position reported when the window is restored would be discarded.
			// This callback is not called every tick, so the query is not harmful (#3318).
			iconified, err := u.isWindowIconifiedUncached()
			if err != nil {
				u.setError(err)
				return
			}
			if iconified {
				return
			}

			m, err := u.currentMonitor()
			if err != nil {
				u.setError(err)
				return
			}
			nx, ny := windowPositionInDIP(x, y, u.windowXInDIP, u.windowYInDIP, m)
			if err := u.setWindowPositionInDIP(nx, ny, m, false); err != nil {
				u.setError(err)
				return
			}
		}
	}
	if _, err := u.window.SetPosCallback(u.posCallback); err != nil {
		return err
	}
	return nil
}

// registerWindowFocusCallback must be called from the main thread.
func (u *glfwBackend) registerWindowFocusCallback() error {
	if u.focusCallback == nil {
		u.focusCallback = func(_ *glfw.Window, focused bool) {
			u.setCachedFocus(focused)
		}
	}
	if _, err := u.window.SetFocusCallback(u.focusCallback); err != nil {
		return err
	}
	return nil
}

// registerWindowIconifyCallback must be called from the main thread.
func (u *glfwBackend) registerWindowIconifyCallback() error {
	if u.iconifyCallback == nil {
		u.iconifyCallback = func(_ *glfw.Window, iconified bool) {
			u.layoutSizeCache.invalidate()
			u.setCachedIconified(iconified)
		}
	}
	if _, err := u.window.SetIconifyCallback(u.iconifyCallback); err != nil {
		return err
	}
	return nil
}

// registerWindowSizeCallback must be called from the main thread.
func (u *glfwBackend) registerWindowSizeCallback() error {
	if u.sizeCallback == nil {
		u.sizeCallback = func(_ *glfw.Window, _, _ int) {
			u.layoutSizeCache.invalidate()
		}
	}
	_, err := u.window.SetSizeCallback(u.sizeCallback)
	return err
}

// registerWindowContentScaleCallback must be called from the main thread.
func (u *glfwBackend) registerWindowContentScaleCallback() error {
	if u.contentScaleCallback == nil {
		u.contentScaleCallback = func(_ *glfw.Window, _, _ float32) {
			u.layoutSizeCache.invalidate()
		}
	}
	_, err := u.window.SetContentScaleCallback(u.contentScaleCallback)
	return err
}

// registerWindowFramebufferSizeCallback must be called from the main thread.
func (u *glfwBackend) registerWindowFramebufferSizeCallback() error {
	if u.defaultFramebufferSizeCallback == nil {
		// When the window gets resized (either by manual window resize or a window
		// manager), glfw sends a framebuffer size callback which we need to handle (#1960).
		// This event is the only way to handle the size change at least on i3 window manager.
		u.defaultFramebufferSizeCallback = func(_ *glfw.Window, w, h int) {
			u.layoutSizeCache.invalidate()
			f, err := u.isFullscreen()
			if err != nil {
				u.setError(err)
				return
			}
			if f {
				return
			}
			// The iconification must not be cached here, for the same reason as the window
			// position callback above (#3318).
			iconified, err := u.isWindowIconifiedUncached()
			if err != nil {
				u.setError(err)
				return
			}
			if iconified {
				return
			}

			// w and h are the framebuffer size, not the window size.
			gw, gh, err := u.window.GetSize()
			if err != nil {
				u.setError(err)
				return
			}
			m, err := u.currentMonitor()
			if err != nil {
				u.setError(err)
				return
			}
			s := m.DeviceScaleFactor()
			ww := int(math.Round(dipFromGLFWPixel(float64(gw), s)))
			wh := int(math.Round(dipFromGLFWPixel(float64(gh), s)))
			if err := u.setWindowSizeInDIP(ww, wh, false); err != nil {
				u.setError(err)
				return
			}

			// While the window is being resized on macOS or Windows, the OS traps the main
			// thread in an event-handling loop and the game loop cannot proceed. Render a frame
			// here so that the rendering result follows the window size (#2615).
			u.forceUpdateFrameDuringPollEvents(float64(ww), float64(wh), w, h, s)
		}
	}
	if _, err := u.window.SetFramebufferSizeCallback(u.defaultFramebufferSizeCallback); err != nil {
		return err
	}
	return nil
}

// forceUpdateFrameDuringPollEvents runs one frame when the game loop is blocked by event polling
// on the main thread, like the modal loop during window resizing on macOS and Windows.
// Otherwise, forceUpdateFrameDuringPollEvents does nothing.
//
// forceUpdateFrameDuringPollEvents must be called from the main thread.
func (u *glfwBackend) forceUpdateFrameDuringPollEvents(outsideWidth, outsideHeight float64, screenWidth, screenHeight int, deviceScaleFactor float64) {
	// On macOS and Windows, resizing a window runs a modal loop inside event polling and traps
	// the main thread until the mouse button is released, so a frame must be rendered here.
	// On X11 and Wayland, there is no such modal loop: resize events are delivered through the
	// normal event queue, event polling returns immediately, and the game loop keeps running
	// and rendering during the resize. Rendering an extra frame here in addition to the game
	// loop's own frames caused flickering (#2144).
	if runtime.GOOS != "darwin" && runtime.GOOS != "windows" {
		return
	}

	// Unless the main thread is polling events for the game loop, the game loop is not blocked
	// by this callback. Also, a frame must not run in the middle of another main-thread operation
	// invoking this callback, like setting a window size.
	if !u.pollingEvents {
		return
	}

	// Prevent recursive frames e.g. when the game's Update changes the window size.
	if u.forcingFrame {
		return
	}

	if !u.bufferOnceSwapped {
		return
	}

	// In the single-thread mode, the game loop runs on the main thread and cannot be resumed here.
	mainThread, ok := u.mainThread.(interface {
		NestedLoop(ctx stdcontext.Context) error
	})
	if !ok {
		return
	}

	u.forcingFrame = true
	defer func() {
		u.forcingFrame = false
	}()

	u.input.setContentSize(outsideWidth, outsideHeight)

	// Run the frame on another goroutine, as the game's Update and Draw must not run on the main
	// thread. Keep processing main-thread calls in a nested loop until the frame ends, since
	// running a frame can request them.
	var err error
	ctx, cancel := stdcontext.WithCancel(stdcontext.Background())
	go func() {
		defer cancel()
		err = u.context.forceUpdateFrame(u.graphicsDriver, outsideWidth, outsideHeight, screenWidth, screenHeight, deviceScaleFactor, u.UserInterface)
	}()
	_ = mainThread.NestedLoop(ctx)
	if err != nil {
		u.setError(err)
	}
}

func (u *glfwBackend) registerDropCallback() error {
	if u.dropCallback == nil {
		u.dropCallback = func(_ *glfw.Window, names []string) {
			fs, err := file.NewVirtualFS(names)
			if err != nil {
				u.setError(err)
				return
			}
			u.input.setDroppedFiles(fs)
		}
	}
	if _, err := u.window.SetDropCallback(u.dropCallback); err != nil {
		return err
	}
	return nil
}

// waitForFramebufferSizeCallback waits for GLFW's FramebufferSize callback.
// f is a process executed after registering the callback.
// If the callback is not invoked for a while, waitForFramebufferSizeCallback times out and returns.
//
// waitForFramebufferSizeCallback must be called from the main thread.
func (u *glfwBackend) waitForFramebufferSizeCallback(window *glfw.Window, f func() error) error {
	// The temporary callback suppresses normal resize handling, including on timeout.
	defer u.layoutSizeCache.invalidate()
	u.framebufferSizeCallbackCh = make(chan struct{}, 1)

	if u.framebufferSizeCallback == nil {
		u.framebufferSizeCallback = func(_ *glfw.Window, _, _ int) {
			u.layoutSizeCache.invalidate()
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

func (u *glfwBackend) initOnMainThread(options *RunOptions) (err error) {
	if err := u.ensureGLFWInit(); err != nil {
		return err
	}

	// GLFW is terminated at the end of the game loop. On a failure before the loop starts,
	// terminate GLFW here so that the window, the cursors, and the platform resources are released.
	defer func() {
		if err != nil {
			err = errors.Join(err, u.terminateGLFW())
		}
	}()

	// Center the window on the monitor if the position was not explicitly set.
	if !options.WindowPositionSet {
		m := u.getRequestedMonitor()
		if m != nil {
			sw, sh := m.sizeInDIP()
			x, y := InitialWindowPosition(int(sw), int(sh), options.InitWindowWidthInDIP, options.InitWindowHeightInDIP)
			p := image.Pt(x, y)
			u.desktopWindow.windowPositionInDIP.init(p)
		}
	}

	u.setApplePressAndHoldEnabled(options.ApplePressAndHoldEnabled)

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
		if u.desktopWindow.isWindowDecorated() {
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

	// The OpenGL driver needs a window with a GL context, unlike the other drivers.
	// Set the context-related hints before creating a window.
	if lib == GraphicsLibraryOpenGL {
		if err := u.setOpenGLWindowHints(); err != nil {
			return err
		}
	}

	// A window created without a redirection surface shows nothing unless its content is presented
	// through DirectComposition, and only the graphics driver can tell whether that works (#3489).
	noRedirectionBitmap := glfw.False
	if d, ok := g.(interface{ SupportsDirectComposition() bool }); ok && d.SupportsDirectComposition() {
		noRedirectionBitmap = glfw.True
	}
	if err := glfw.WindowHint(glfw.Win32NoRedirectionBitmap, noRedirectionBitmap); err != nil {
		return err
	}

	// internal/glfw is customized and the default client API is NoAPI, not OpenGLAPI.
	// Then, glfw.WindowHint(glfw.ClientAPI, glfw.NoAPI) doesn't have to be called.

	// Before creating a window, set it unresizable no matter what u.isInitWindowResizable() is (#1987).
	// Making the window resizable here doesn't work correctly when switching to enable resizing.
	resizable := glfw.False
	if u.desktopWindow.windowResizingMode.Load() == WindowResizingModeEnabled {
		resizable = glfw.True
	}
	if err := glfw.WindowHint(glfw.Resizable, resizable); err != nil {
		return err
	}

	floating := glfw.False
	if u.desktopWindow.isWindowFloating() {
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
	if u.desktopWindow.isWindowMousePassthrough() {
		mousePassthrough = glfw.True
	}
	if err := glfw.WindowHint(glfw.MousePassthrough, mousePassthrough); err != nil {
		return err
	}

	if err := u.createWindow(); err != nil {
		return err
	}

	if err := u.applyWindowSettings(); err != nil {
		return err
	}

	// Maximizing a window requires a proper size and position. Call Maximize here (#1117).
	if u.desktopWindow.isInitWindowMaximized() {
		if err := u.window.Maximize(); err != nil {
			return err
		}
	}

	if err := u.setWindowResizingModeForOS(u.desktopWindow.windowResizingMode.Load()); err != nil {
		return err
	}

	if options.SkipTaskbar {
		// Ignore the error.
		_ = u.skipTaskbar()
	}

	switch g := u.graphicsDriver.(type) {
	case interface{ SetPresenter(opengl.Presenter) }:
		g.SetPresenter(u.window)
	case interface{ SetWindow(uintptr) }:
		w, err := u.nativeWindow()
		if err != nil {
			return err
		}
		g.SetWindow(w)
	}

	if g, ok := u.graphicsDriver.(interface{ SetMainThreadRunner(func(func())) }); ok {
		g.SetMainThreadRunner(u.RunOnMainThread)
	}

	// Register callbacks after the window initialization is done.
	// The callback might cause swapping frames, which assumes the window is already set (#2137).
	if err := u.registerWindowCloseCallback(); err != nil {
		return err
	}
	if err := u.registerWindowFocusCallback(); err != nil {
		return err
	}
	if err := u.registerWindowIconifyCallback(); err != nil {
		return err
	}
	// Refresh the cached window states so that they are known before the first focus or iconify
	// callback.
	if err := u.refreshCachedWindowStates(); err != nil {
		return err
	}
	if err := u.registerWindowPosCallback(); err != nil {
		return err
	}
	if err := u.registerWindowSizeCallback(); err != nil {
		return err
	}
	if err := u.registerWindowContentScaleCallback(); err != nil {
		return err
	}
	if err := u.registerWindowFramebufferSizeCallback(); err != nil {
		return err
	}
	if _, err := u.refreshCachedWindowSizes(time.Now()); err != nil {
		return err
	}
	if _, err := u.refreshCachedNativeFullscreen(); err != nil {
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

// outsideSizeInDIP returns the size to give the game's Layout, in device-independent pixels.
func outsideSizeInDIP(windowWidth, windowHeight int, requestedWidthInDIP, requestedHeightInDIP int, fullscreen bool, deviceScaleFactor float64) (float64, float64) {
	// The requested size is a windowed size, unrelated to the size of a fullscreen window.
	if !fullscreen {
		// Report the requested size while the window has the pixel size that request produces, as
		// converting the pixel size back would not return it at a fractional scale factor (#2978).
		// Otherwise use the actual window size, which might not match the specified size on
		// Windows (#1163).
		if rw, rh := windowSizeInGLFWPixels(requestedWidthInDIP, requestedHeightInDIP, deviceScaleFactor); windowWidth == rw && windowHeight == rh {
			return float64(requestedWidthInDIP), float64(requestedHeightInDIP)
		}
	}
	return dipFromGLFWPixel(float64(windowWidth), deviceScaleFactor), dipFromGLFWPixel(float64(windowHeight), deviceScaleFactor)
}

type windowPropertyCache[T any] struct {
	value     T
	nextQuery time.Time
}

func (c *windowPropertyCache[T]) invalidate() {
	c.nextQuery = time.Time{}
}

func (c *windowPropertyCache[T]) get(now time.Time) (T, bool) {
	// The zero deadline forces initialization on first access. Monotonic time keeps
	// expiry independent of game ticks, including while the game is paused.
	return c.value, !now.After(c.nextQuery)
}

func (c *windowPropertyCache[T]) set(value T, now time.Time) {
	c.value = value
	c.nextQuery = now.Add(windowStateQueryInterval)
}

type windowSizes struct {
	window, framebuffer image.Point
}

func (u *glfwBackend) refreshCachedWindowSizes(now time.Time) (windowSizes, error) {
	ww, wh, err := u.window.GetSize()
	if err != nil {
		return windowSizes{}, err
	}
	fw, fh, err := u.window.GetFramebufferSize()
	if err != nil {
		return windowSizes{}, err
	}
	sizes := windowSizes{
		window:      image.Pt(ww, wh),
		framebuffer: image.Pt(fw, fh),
	}
	u.layoutSizeCache.set(sizes, now)
	return sizes, nil
}

func (u *glfwBackend) refreshCachedNativeFullscreen() (bool, error) {
	fullscreen, err := u.isNativeFullscreen()
	if err != nil {
		return false, err
	}
	u.nativeFullscreenCache.set(fullscreen, time.Now())
	return fullscreen, nil
}

func (u *glfwBackend) nativeFullscreenForLayout() (bool, error) {
	if runtime.GOOS != "darwin" {
		return u.isNativeFullscreen()
	}
	// AppKit changes the style mask asynchronously during a fullscreen transition.
	// Keep observing it until the completion notification, including failed transitions.
	if fullscreen, ok := u.nativeFullscreenCache.get(time.Now()); ok && !u.nativeFullscreenTransition {
		return fullscreen, nil
	}
	return u.refreshCachedNativeFullscreen()
}

// layoutSizes returns the size to give the game's Layout, in device-independent pixels, and the
// size of the final rendering destination, in pixels.
//
// layoutSizes must be called from the main thread.
func (u *glfwBackend) layoutSizes() (outsideWidth, outsideHeight float64, screenWidth, screenHeight int, err error) {
	m, err := u.currentMonitor()
	if err != nil {
		return 0, 0, 0, 0, err
	}
	if m == nil {
		return 0, 0, 0, 0, nil
	}
	s := m.DeviceScaleFactor()

	wf, err := u.isWindowedFullscreen()
	if err != nil {
		return 0, 0, 0, 0, err
	}
	nf, err := u.nativeFullscreenForLayout()
	if err != nil {
		return 0, 0, 0, 0, err
	}
	fullscreen := wf || nf
	if u.nativeFullscreenTransition {
		u.layoutSizeCache.invalidate()
	}

	// Query actual sizes together after callbacks have finished. Neither callback ordering
	// nor requested sizes determine the native window's final rendering destination (#2225).
	now := time.Now()
	sizes, ok := u.layoutSizeCache.get(now)
	if !ok {
		sizes, err = u.refreshCachedWindowSizes(now)
		if err != nil {
			return 0, 0, 0, 0, err
		}
	}
	fw, fh := sizes.framebuffer.X, sizes.framebuffer.Y

	iconified, err := u.isWindowIconified()
	if err != nil {
		return 0, 0, 0, 0, err
	}
	if iconified {
		// An iconified window has no size to lay out for; use the size it is restored to, which is
		// the monitor's size in fullscreen and the requested size otherwise. A minimized window
		// reports no client area on Windows, so the rendering destination comes from that same
		// source.
		if fullscreen {
			w, h := m.sizeInDIP()
			if fw == 0 || fh == 0 {
				fw, fh = m.boundsInGLFWPixels.Dx(), m.boundsInGLFWPixels.Dy()
			}
			return w, h, fw, fh, nil
		}
		w := float64(u.windowWidthInDIP)
		h := float64(u.windowHeightInDIP)
		if fw == 0 || fh == 0 {
			// The framebuffer needs at least one pixel in each dimension.
			fw, fh = max(1, int(math.Round(w*s))), max(1, int(math.Round(h*s)))
		}
		return w, h, fw, fh, nil
	}

	ww, wh := sizes.window.X, sizes.window.Y
	w, h := outsideSizeInDIP(ww, wh, u.windowWidthInDIP, u.windowHeightInDIP, fullscreen, s)
	return w, h, fw, fh, nil
}

// setFPSMode must be called from the main thread.
func (u *glfwBackend) setFPSMode(fpsMode FPSModeType) error {
	// The unchanged-mode case is filtered out by UserInterface.SetFPSMode, which updates u.fpsMode.
	// Do not compare fpsMode with u.fpsMode here.
	u.fpsModeInited = true

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

	graphicscommand.SetVsyncEnabled(fpsMode == FPSModeVsyncOn)

	return nil
}

// update must be called from the main thread.
func (u *glfwBackend) update() (outsideWidth, outsideHeight float64, screenWidth, screenHeight int, err error) {
	if err := u.error(); err != nil {
		return 0, 0, 0, 0, err
	}

	settingsChanged := u.desktopWindow.settingsChanged.Swap(false)
	if err := u.applyWindowSettings(); err != nil {
		return 0, 0, 0, 0, err
	}

	sc, err := u.window.ShouldClose()
	if err != nil {
		return 0, 0, 0, 0, err
	}
	if sc {
		return 0, 0, 0, 0, RegularTermination
	}

	// On macOS, one swapping buffers seems required before entering fullscreen (#2599).
	if u.isInitFullscreen() && (u.bufferOnceSwapped || runtime.GOOS != "darwin") {
		if err := u.setFullscreen(true); err != nil {
			return 0, 0, 0, 0, err
		}
		u.setInitFullscreen(false)
	}

	// Initialize vsync after SetMonitor is called.
	// Calling this inside setWindowSizeInDIP didn't work (#1363).
	if !u.fpsModeInited {
		if err := u.setFPSMode(FPSModeType(u.fpsMode.Load())); err != nil {
			return 0, 0, 0, 0, err
		}
	}

	// Settings can change while their application polls native events. Check again
	// before waiting, since those events may have consumed the setter's wakeup.
	settingsChanged = u.desktopWindow.settingsChanged.Swap(false) || settingsChanged
	if FPSModeType(u.fpsMode.Load()) != FPSModeVsyncOffMinimum || settingsChanged {
		// TODO: Updating the input can be skipped when clock.Update returns 0 (#1367).
		u.pollingEvents = true
		err := glfw.PollEvents()
		u.pollingEvents = false
		if err != nil {
			return 0, 0, 0, 0, err
		}
	} else {
		u.pollingEvents = true
		// A bounded wait lets caches recover from missed native notifications even
		// when the game requests frames only in response to events.
		err := glfw.WaitEventsTimeout(windowStateQueryInterval.Seconds())
		u.pollingEvents = false
		if err != nil {
			return 0, 0, 0, 0, err
		}
	}
	if err := u.applyWindowSettings(); err != nil {
		return 0, 0, 0, 0, err
	}
	u.syncModKeysFromOS()
	u.syncLockKeysFromOS()

	// If isRunnableOnUnfocused is false and the window is not focused, wait here.
	// For the first update, skip this check as the window might not be seen yet in some environments like ChromeOS (#3091).
	for !u.isRunnableOnUnfocused() && u.bufferOnceSwapped {
		if err := u.applyWindowSettings(); err != nil {
			return 0, 0, 0, 0, err
		}
		// In the initial state on macOS, the window is not shown (#2620).
		visible, err := u.isWindowVisible()
		if err != nil {
			return 0, 0, 0, 0, err
		}
		if !visible {
			break
		}

		focused, err := u.isWindowFocused()
		if err != nil {
			return 0, 0, 0, 0, err
		}
		if focused {
			break
		}

		shouldClose, err := u.window.ShouldClose()
		if err != nil {
			return 0, 0, 0, 0, err
		}
		if shouldClose {
			break
		}

		if err := hook.SuspendAudio(); err != nil {
			return 0, 0, 0, 0, err
		}
		// Wait for an arbitrary period to avoid busy loop.
		time.Sleep(time.Second / 60)
		if err := glfw.PollEvents(); err != nil {
			return 0, 0, 0, 0, err
		}
	}

	if err := hook.ResumeAudio(); err != nil {
		return 0, 0, 0, 0, err
	}

	return u.layoutSizes()
}

// terminateGLFW marks the UI terminated and terminates GLFW.
//
// terminateGLFW must be called from the main thread.
func (u *glfwBackend) terminateGLFW() error {
	// Mark termination before destroying GLFW so concurrent APIs stop accessing it.
	u.setTerminated()
	return glfw.Terminate()
}

func (u *glfwBackend) loopGame() (err error) {
	defer func() {
		graphicscommand.Terminate()
		if glfwErr := thread.CallWithArgAndResult(u.mainThread, (*glfwBackend).terminateGLFW, u); glfwErr != nil {
			err = errors.Join(err, glfwErr)
		}
	}()

	for {
		if err := u.updateGame(); err != nil {
			return err
		}
	}
}

// shouldPresentFrame reports whether a frame should be presented to the window.
func shouldPresentFrame(windowOnScreen, bufferOnceSwapped, windowVisible bool) bool {
	if windowOnScreen {
		return true
	}

	// A window that is to be shown at startup stays hidden until the first frame is presented (#2875).
	// That frame must still be presented (#3508): showing the window and, on macOS, entering the
	// fullscreen mode (#2599) both require buffers to have been swapped once.
	if !bufferOnceSwapped && windowVisible {
		return true
	}

	// Skip the buffer swap so that the tick rate stays at the specified TPS. On macOS, a present for
	// an occluded window waits for a display link that the OS throttles far below the refresh rate,
	// and the tick rate would drop with it (#3405).
	return false
}

func (u *glfwBackend) updateGame() error {
	type result struct {
		unfocused      bool
		present        bool
		monitorChanged bool

		outsideWidth, outsideHeight float64
		screenWidth, screenHeight   int
		deviceScaleFactor           float64
		err                         error
	}
	r := thread.CallWithArgAndResult(u.mainThread, func(u *glfwBackend) result {
		var r result

		// On Windows, the focusing state might be always false (#987).
		// On Windows, even if a window is in another workspace, vsync seems to work.
		// Then let's assume the window is always 'focused' as a workaround.
		if runtime.GOOS != "windows" {
			focused, e := u.isWindowFocused()
			if e != nil {
				r.err = e
				return r
			}
			r.unfocused = !focused
		}

		visible, e := u.isWindowVisible()
		if e != nil {
			r.err = e
			return r
		}
		occluded, e := u.isWindowOccluded()
		if e != nil && !errors.Is(e, errors.ErrUnsupported) {
			r.err = e
			return r
		}
		r.present = shouldPresentFrame(visible && !occluded, u.bufferOnceSwapped, u.desktopWindow.isWindowVisible())

		r.outsideWidth, r.outsideHeight, r.screenWidth, r.screenHeight, r.err = u.update()
		if r.err != nil {
			return r
		}
		u.input.setContentSize(r.outsideWidth, r.outsideHeight)
		var m *Monitor
		m, r.err = u.currentMonitor()
		if r.err != nil {
			return r
		}
		r.deviceScaleFactor = m.DeviceScaleFactor()
		u.setRefreshRate(m.RefreshRate())
		r.monitorChanged = m != u.lastFrameMonitor
		u.lastFrameMonitor = m

		// Pre-fetch cursor position and update gamepads to avoid
		// a second thread.CallWithArgAndResult round-trip in updateInputStateForFrame.
		var cx, cy float64
		cx, cy, r.err = u.window.GetCursorPos()
		if r.err != nil {
			return r
		}
		u.input.setRawCursorPos(cx, cy)
		var nativeWindow uintptr
		nativeWindow, r.err = u.nativeWindow()
		if r.err != nil {
			return r
		}
		if r.err = gamepad.Update(nativeWindow, nil); r.err != nil {
			return r
		}
		return r
	}, u)
	if r.err != nil {
		return r.err
	}

	// Whether swapping buffers waits for the display can differ per monitor, e.g. when the monitors
	// are driven by different GPUs. Measure it again on the new monitor.
	if r.monitorChanged {
		u.context.resetVsyncDetection()
	}

	if err := u.context.updateFrame(u.graphicsDriver, r.outsideWidth, r.outsideHeight, r.screenWidth, r.screenHeight, r.deviceScaleFactor, u.UserInterface, r.present); err != nil {
		return err
	}

	u.bufferOnceSwappedOnce.Do(func() {
		thread.CallWithArg(u.mainThread, func(u *glfwBackend) {
			u.bufferOnceSwapped = true
		}, u)
	})

	// When a window is not focused or in another space, SwapBuffers might return immediately and CPU might be busy.
	// Mitigate this by sleeping (#982, #2521).
	if r.unfocused {
		const wait = time.Second / 60
		now := time.Now()
		if next := u.unfocusedNextWake.Add(wait); next.After(now) {
			u.unfocusedNextWake = next
			time.Sleep(time.Until(next))
		} else {
			u.unfocusedNextWake = now
		}
	}

	return nil
}

func (u *glfwBackend) updateIconIfNeeded() error {
	imgs := u.desktopWindow.getIconImages()
	// A 0-size slice and nil are distinguished here.
	// A 0-size slice means a user indicates to reset the icon.
	// On the other hand, nil means a user didn't update the icon state.
	if imgs == nil {
		return nil
	}

	var newImgs []image.Image
	if len(*imgs) > 0 {
		newImgs = make([]image.Image, len(*imgs))
	}
	for i, img := range *imgs {
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

	type args struct {
		u       *glfwBackend
		imgs    *[]image.Image
		newImgs []image.Image
	}
	return thread.CallWithArgAndResult(u.mainThread, func(a args) error {
		u := a.u
		if u.isTerminated() {
			return nil
		}
		// Keep icon images pending while fullscreen, where SetIcon fails (#1578).
		f, err := u.isFullscreen()
		if err != nil {
			return err
		}
		if f {
			return nil
		}
		if err := u.window.SetIcon(a.newImgs); err != nil {
			return err
		}
		u.desktopWindow.resetIconImages(a.imgs)
		return nil
	}, args{u: u, imgs: imgs, newImgs: newImgs})
}

// updateWindowSizeLimits must be called from the main thread.
func (u *glfwBackend) updateWindowSizeLimits() error {
	defer u.layoutSizeCache.invalidate()
	m, err := u.currentMonitor()
	if err != nil {
		return err
	}
	minw, minh, maxw, maxh := u.desktopWindow.getWindowSizeLimitsInDIP()

	s := m.DeviceScaleFactor()
	if minw < 0 {
		// Always set the minimum window width.
		mw, err := u.minimumWindowWidth()
		if err != nil {
			return err
		}
		minw = max(1, int(math.Round(dipToGLFWPixel(float64(mw), s))))
	} else {
		minw = max(1, int(math.Round(dipToGLFWPixel(float64(minw), s))))
	}
	if minh < 0 {
		minh = glfw.DontCare
	} else {
		minh = max(1, int(math.Round(dipToGLFWPixel(float64(minh), s))))
	}
	if maxw < 0 {
		maxw = glfw.DontCare
	} else {
		maxw = max(1, int(math.Round(dipToGLFWPixel(float64(maxw), s))))
	}
	if maxh < 0 {
		maxh = glfw.DontCare
	} else {
		maxh = max(1, int(math.Round(dipToGLFWPixel(float64(maxh), s))))
	}
	if err := u.window.SetSizeLimits(minw, minh, maxw, maxh); err != nil {
		return err
	}

	// The window size limit affects the resizing mode, especially on macOS (#2260).
	if err := u.setWindowResizingModeForOS(u.desktopWindow.windowResizingMode.Load()); err != nil {
		return err
	}

	return nil
}

// disableWindowSizeLimits disables a window size limitation temporarily, especially for fullscreen
// In order to enable the size limitation, call updateWindowSizeLimits.
//
// disableWindowSizeLimits must be called from the main thread.
func (u *glfwBackend) disableWindowSizeLimits() error {
	return u.window.SetSizeLimits(glfw.DontCare, glfw.DontCare, glfw.DontCare, glfw.DontCare)
}

// windowSizeInGLFWPixels returns the window size in GLFW pixels for the given size in
// device-independent pixels.
func windowSizeInGLFWPixels(widthInDIP, heightInDIP int, deviceScaleFactor float64) (int, int) {
	// Native window dimensions must stay positive even at a low content scale.
	return max(1, int(math.Round(dipToGLFWPixel(float64(widthInDIP), deviceScaleFactor)))), max(1, int(math.Round(dipToGLFWPixel(float64(heightInDIP), deviceScaleFactor))))
}

// windowSizeToRestore returns the size to give the window on leaving fullscreen, in GLFW pixels.
//
// capturedWidth and capturedHeight are the size captured on entering fullscreen on
// capturedMonitor, or invalidSize when there is none.
func windowSizeToRestore(capturedWidth, capturedHeight int, capturedMonitor *Monitor, widthInDIP, heightInDIP int, monitor *Monitor) (int, int) {
	// Restore the captured pixel size, as converting a size in device-independent pixels back
	// would not return it at a fractional scale factor. A pixel count is that apparent size only
	// on the monitor it was captured on, so use the size in device-independent pixels on any
	// other one.
	if capturedWidth != invalidSize && capturedHeight != invalidSize && capturedMonitor == monitor {
		return capturedWidth, capturedHeight
	}
	return windowSizeInGLFWPixels(widthInDIP, heightInDIP, monitor.DeviceScaleFactor())
}

// setWindowSizeInDIP must be called from the main thread.
func (u *glfwBackend) setWindowSizeInDIP(width, height int, callSetSize bool) error {
	if microsoftgdk.IsXbox() {
		// Do nothing. The size is always fixed.
		return nil
	}

	width, height = u.desktopWindow.adjustWindowSizeBasedOnSizeLimitsInDIP(width, height)
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
	if u.windowWidthInDIP == width && u.windowHeightInDIP == height && u.lastDeviceScaleFactor == scale {
		return nil
	}
	u.lastDeviceScaleFactor = scale

	u.windowWidthInDIP = width
	u.windowHeightInDIP = height

	f, err := u.isFullscreen()
	if err != nil {
		return err
	}
	if f {
		// The window keeps its fullscreen size, so update the size it is restored to instead,
		// as setWindowPositionInDIP does for the position.
		w, h := windowSizeInGLFWPixels(width, height, scale)
		u.windowToRestore.size = image.Pt(w, h)
		u.windowToRestore.monitor = mon
	} else if callSetSize {
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
		newW, newH := windowSizeInGLFWPixels(width, height, s)
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

// captureWindowPosToRestore must be called from the main thread.
func (u *glfwBackend) captureWindowPosToRestore() error {
	if u.windowToRestore.pos.X == invalidPos || u.windowToRestore.pos.Y == invalidPos {
		x, y, err := u.window.GetPos()
		if err != nil {
			return err
		}
		u.windowToRestore.pos = image.Pt(x, y)
	}
	return nil
}

// setFullscreen must be called from the main thread.
func (u *glfwBackend) setFullscreen(fullscreen bool) error {
	defer u.layoutSizeCache.invalidate()
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
		u.input.saveCursorPos()
	}

	// Enter the fullscreen.
	if fullscreen {
		if err := u.disableWindowSizeLimits(); err != nil {
			return err
		}

		if u.windowToRestore.pos.X == invalidPos || u.windowToRestore.pos.Y == invalidPos {
			x, y, err := u.window.GetPos()
			if err != nil {
				return err
			}
			u.windowToRestore.pos = image.Pt(x, y)
		}

		w, h, err := u.window.GetSize()
		if err != nil {
			return err
		}
		m, err := u.currentMonitor()
		if err != nil {
			return err
		}
		u.windowToRestore.size = image.Pt(w, h)
		u.windowToRestore.monitor = m

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
		return nil
	}

	// Exit the fullscreen.
	if err := u.updateWindowSizeLimits(); err != nil {
		return err
	}

	restorePos := u.windowToRestore.pos
	restoreSize := u.windowToRestore.size
	restoreMonitor := u.windowToRestore.monitor

	m, err := u.currentMonitor()
	if err != nil {
		return err
	}
	ww, wh := windowSizeToRestore(restoreSize.X, restoreSize.Y, restoreMonitor, u.windowWidthInDIP, u.windowHeightInDIP, m)
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

	if restorePos.X != invalidPos && restorePos.Y != invalidPos {
		if err := u.window.SetPos(restorePos.X, restorePos.Y); err != nil {
			return err
		}
		// Dirty hack for macOS (#703). Rendering doesn't work correctly with one SetPos, but
		// work with two or more SetPos.
		if runtime.GOOS == "darwin" {
			if err := u.window.SetPos(restorePos.X+1, restorePos.Y); err != nil {
				return err
			}
			if err := u.window.SetPos(restorePos.X, restorePos.Y); err != nil {
				return err
			}
		}
		u.windowToRestore.pos = image.Pt(invalidPos, invalidPos)
	}

	if u.isNativeFullscreenAvailable() {
		// Set the window size after the position. The order matters.
		// In the opposite order, the window size might not be correct when going back from fullscreen with multi monitors.
		if err := u.window.SetSize(ww, wh); err != nil {
			return err
		}
	}

	u.windowToRestore.size = image.Pt(invalidSize, invalidSize)
	u.windowToRestore.monitor = nil

	return nil
}

func (u *glfwBackend) minimumWindowWidth() (int, error) {
	a, err := u.window.GetAttrib(glfw.Decorated)
	if err != nil {
		return 0, err
	}
	if a == glfw.False {
		return 1, nil
	}

	// On Windows, giving a too small width doesn't call a callback (#165).
	// To prevent hanging up, return asap if the width is too small.
	// 126 is an arbitrary number and I guess this is small enough.
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
func (u *glfwBackend) currentMonitor() (*Monitor, error) {
	if u.cachedCurrentMonitor != nil && u.cachedCurrentMonitorTime > u.Tick()-int64(clock.TPS()) && theMonitors.contains(u.cachedCurrentMonitor) {
		return u.cachedCurrentMonitor, nil
	}

	m, err := u.currentMonitorImpl()
	if err != nil {
		return nil, err
	}
	u.cachedCurrentMonitor = m
	u.cachedCurrentMonitorTime = u.Tick()
	return m, nil
}

// currentMonitorImpl must be called from the main thread.
func (u *glfwBackend) currentMonitorImpl() (*Monitor, error) {
	if u.window == nil {
		return u.getRequestedMonitor(), nil
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

	if m := theMonitors.primaryMonitor(); m != nil {
		return m, nil
	}

	// The primary monitor might be missing even after the initialization (#3094, #3241).
	// The reason is still unknown. As a workaround, return the last requested monitor.
	return u.getRequestedMonitor(), nil
}

// windowStateQueryInterval is how long a cached window state is trusted before it is re-queried as
// a safety net against a missed GLFW callback. One second bounds any divergence while leaving
// almost every tick query-free. The deadline is measured on a monotonic clock, so that the fallback
// works independently of the game ticks, which do not advance with TPS 0 or while the game is
// paused (#3318).
const windowStateQueryInterval = time.Second

// isWindowFocused reports whether the window has input focus.
//
// The result is cached to avoid querying the window system every tick (#3318). GLFW's focus
// callback and the places that change the focus update the cache, and the periodic re-query bounds
// any divergence.
//
// isWindowFocused must be called on the main thread.
func (u *glfwBackend) isWindowFocused() (bool, error) {
	if u.window == nil {
		return false, nil
	}
	now := time.Now()
	if focused, ok := u.focusedCache.get(now); ok {
		return focused, nil
	}
	a, err := u.window.GetAttrib(glfw.Focused)
	if err != nil {
		return false, err
	}
	focused := a == glfw.True
	u.focusedCache.set(focused, now)
	return focused, nil
}

// isWindowVisible reports whether the window is visible.
//
// The result is cached like isWindowFocused. GLFW's visible attribute reflects iconification on
// some platforms, so the iconify callback also refreshes it (#3318).
//
// isWindowVisible must be called on the main thread.
func (u *glfwBackend) isWindowVisible() (bool, error) {
	if u.window == nil {
		return false, nil
	}
	now := time.Now()
	if visible, ok := u.visibleCache.get(now); ok {
		return visible, nil
	}
	a, err := u.window.GetAttrib(glfw.Visible)
	if err != nil {
		return false, err
	}
	visible := a == glfw.True
	u.visibleCache.set(visible, now)
	return visible, nil
}

// isWindowIconified reports whether the window is iconified.
//
// The result is cached like isWindowFocused (#3318).
//
// isWindowIconified must be called on the main thread.
func (u *glfwBackend) isWindowIconified() (bool, error) {
	if u.window == nil {
		return false, nil
	}
	now := time.Now()
	if iconified, ok := u.iconifiedCache.get(now); ok {
		return iconified, nil
	}
	a, err := u.window.GetAttrib(glfw.Iconified)
	if err != nil {
		return false, err
	}
	iconified := a == glfw.True
	u.iconifiedCache.set(iconified, now)
	return iconified, nil
}

// isWindowIconifiedUncached reports whether the window is iconified, querying the window system
// directly without using the cache.
//
// A GLFW window event callback must not use the cached iconification, because the callbacks of a
// window state change are not ordered: on Windows, for example, WM_MOVE can arrive before the
// WM_SIZE which reports the iconification, so the cached value can still be false for a window
// which is being iconified. These callbacks are not called every tick, so the query is not harmful
// (#3318).
//
// isWindowIconifiedUncached must be called on the main thread.
func (u *glfwBackend) isWindowIconifiedUncached() (bool, error) {
	if u.window == nil {
		return false, nil
	}
	a, err := u.window.GetAttrib(glfw.Iconified)
	if err != nil {
		return false, err
	}
	return a == glfw.True, nil
}

// refreshCachedWindowStates queries the window's focus, visibility and iconification from the
// window system and updates the caches with the actual values.
//
// refreshCachedWindowStates must be called on the main thread.
func (u *glfwBackend) refreshCachedWindowStates() error {
	if u.window == nil {
		return nil
	}

	now := time.Now()
	focused, err := u.window.GetAttrib(glfw.Focused)
	if err != nil {
		return err
	}
	visible, err := u.window.GetAttrib(glfw.Visible)
	if err != nil {
		return err
	}
	iconified, err := u.window.GetAttrib(glfw.Iconified)
	if err != nil {
		return err
	}

	u.focusedCache.set(focused == glfw.True, now)
	u.visibleCache.set(visible == glfw.True, now)
	u.iconifiedCache.set(iconified == glfw.True, now)
	return nil
}

// setCachedFocus records the window's focus reported by a GLFW callback.
//
// setCachedFocus must be called on the main thread.
func (u *glfwBackend) setCachedFocus(focused bool) {
	u.focusedCache.set(focused, time.Now())
}

// setCachedIconified records the window's iconification reported by a GLFW callback.
//
// GLFW's visible attribute reflects iconification on some platforms, but no visibility callback
// reports the change there, so the cached visibility is invalidated here (#3318).
//
// setCachedIconified must be called on the main thread.
func (u *glfwBackend) setCachedIconified(iconified bool) {
	u.iconifiedCache.set(iconified, time.Now())
	u.visibleCache.invalidate()
}

// invalidateCachedWindowStates forces the cached window states to be re-queried on the next access.
// It is used where a change is expected but its new value is not known here, like a programmatic
// show or hide: GLFW reports the requested state only after the window system has applied it, and
// some calls are no-ops in some states, so the requested value must not be cached as the actual one
// (#3318).
//
// invalidateCachedWindowStates must be called on the main thread.
func (u *glfwBackend) invalidateCachedWindowStates() {
	u.layoutSizeCache.invalidate()
	u.occlusionCache.invalidate()
	u.focusedCache.invalidate()
	u.visibleCache.invalidate()
	u.iconifiedCache.invalidate()
}

func (u *glfwBackend) readInputState(inputState *InputState) {
	u.input.read(inputState)
}

func (u *glfwBackend) Window() backendWindow {
	return &u.backendWindow
}

// windowStateTimeout is the maximum duration to wait for the window manager to reflect
// a requested maximize, iconify, or restore in the window attributes.
// Some window managers ignore such requests, and waiting forever would hang the main thread.
const windowStateTimeout = time.Second

// GLFW's functions to manipulate a window can invoke the SetSize callback (#1576, #1585, #1606).
// As the callback must not be called in the frame (between BeginFrame and EndFrame),
// disable the callback temporarily.

// maximizeWindow must be called from the main thread.
func (u *glfwBackend) maximizeWindow() error {
	f, err := u.isFullscreen()
	if err != nil {
		return err
	}
	if f {
		return nil
	}

	supported, err := u.window.MaximizeSupported()
	if err != nil {
		return err
	}
	if !supported {
		return nil
	}

	if err := u.window.Maximize(); err != nil {
		return err
	}

	// On Linux/UNIX, maximizing might not finish even though Maximize returns. Just wait for its finish.
	// Do not check this in the fullscreen since apparently the condition can never be true.
	deadline := time.Now().Add(windowStateTimeout)
	for {
		a, err := u.window.GetAttrib(glfw.Maximized)
		if err != nil {
			return err
		}
		if a == glfw.True {
			break
		}
		if time.Now().After(deadline) {
			break
		}
		if err := glfw.PollEvents(); err != nil {
			return err
		}
		time.Sleep(time.Second / 60)
	}

	return nil
}

// iconifyWindow must be called from the main thread.
func (u *glfwBackend) iconifyWindow() error {
	// Iconifying a native fullscreen window on macOS is forbidden.
	n, err := u.isNativeFullscreen()
	if err != nil {
		return err
	}
	if n {
		return nil
	}

	supported, err := u.window.IconifySupported()
	if err != nil {
		return err
	}
	if !supported {
		return nil
	}

	if err := u.window.Iconify(); err != nil {
		return err
	}

	// On Linux/UNIX, iconifying might not finish even though Iconify returns. Just wait for its finish.
	deadline := time.Now().Add(windowStateTimeout)
	for {
		a, err := u.window.GetAttrib(glfw.Iconified)
		if err != nil {
			return err
		}
		if a == glfw.True {
			break
		}
		if time.Now().After(deadline) {
			break
		}
		if err := glfw.PollEvents(); err != nil {
			return err
		}
		time.Sleep(time.Second / 60)
	}

	return nil
}

// restoreWindow must be called from the main thread.
func (u *glfwBackend) restoreWindow() error {
	supported, err := u.window.RestoreSupported()
	if err != nil {
		return err
	}
	if !supported {
		return nil
	}

	if err := u.window.Restore(); err != nil {
		return err
	}

	// On Linux/UNIX, restoring might not finish even though Restore returns (#1608). Just wait for its finish.
	// On macOS, the restoring state might be the same as the maximized state. Skip this.
	if runtime.GOOS != "darwin" {
		deadline := time.Now().Add(windowStateTimeout)
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
			if time.Now().After(deadline) {
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

// setWindowVisible must be called from the main thread.
func (u *glfwBackend) setWindowVisible(visible bool) error {
	if microsoftgdk.IsXbox() {
		return nil
	}

	if !visible {
		if err := u.window.Hide(); err != nil {
			return err
		}
		// GLFW's callbacks do not report programmatic show or hide, and Hide can be a no-op
		// in GLFW fullscreen. Do not cache the requested state as the actual one, but
		// re-query the window system on the next access (#3318).
		u.invalidateCachedWindowStates()
		return nil
	}
	if err := u.window.Show(); err != nil {
		return err
	}
	// The window became visible, but GLFW's callbacks do not report programmatic show or hide.
	// Re-query the states on the next access so that the game loop presents without waiting
	// for the periodic re-query (#3318).
	u.invalidateCachedWindowStates()
	var err error
	u.showWindowOnce.Do(func() {
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
		newW, newH := windowSizeInGLFWPixels(u.windowWidthInDIP, u.windowHeightInDIP, s)

		// Even though a framebuffer callback is not called, waitForFramebufferSizeCallback returns by timeout,
		// so it is safe to use this.
		if err = u.waitForFramebufferSizeCallback(u.window, func() error {
			return u.window.SetSize(newW, newH)
		}); err != nil {
			return
		}
	})
	return err
}

// setWindowDecorated must be called from the main thread.
func (u *glfwBackend) setWindowDecorated(decorated bool) error {
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
		if err := u.window.SetTitle(u.desktopWindow.title.Load()); err != nil {
			return err
		}
	}

	return nil
}

// setWindowFloating must be called from the main thread.
func (u *glfwBackend) setWindowFloating(floating bool) error {
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
func (u *glfwBackend) setWindowResizingMode(mode WindowResizingMode) error {
	if microsoftgdk.IsXbox() {
		return nil
	}

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

// windowPositionInGLFWPixels returns the window position in GLFW pixels for the given position in
// device-independent pixels relative to the given monitor.
func windowPositionInGLFWPixels(xInDIP, yInDIP int, monitor *Monitor) (int, int) {
	s := monitor.DeviceScaleFactor()
	mx := monitor.boundsInGLFWPixels.Min.X
	my := monitor.boundsInGLFWPixels.Min.Y
	return mx + int(math.Round(dipToGLFWPixel(float64(xInDIP), s))), my + int(math.Round(dipToGLFWPixel(float64(yInDIP), s)))
}

// windowPositionInDIP returns the position to store for a window that moved to windowX, windowY in
// GLFW pixels, in device-independent pixels relative to the given monitor.
//
// xInDIP and yInDIP are the position stored so far.
func windowPositionInDIP(windowX, windowY int, xInDIP, yInDIP int, monitor *Monitor) (int, int) {
	// Keep the stored position while the window is at the pixel position it describes, as
	// converting the pixel position back would not return it at a fractional scale factor
	// (#2978). Otherwise the window moved for a reason other than a request, so use where it is.
	if px, py := windowPositionInGLFWPixels(xInDIP, yInDIP, monitor); windowX == px && windowY == py {
		return xInDIP, yInDIP
	}

	s := monitor.DeviceScaleFactor()
	mx := monitor.boundsInGLFWPixels.Min.X
	my := monitor.boundsInGLFWPixels.Min.Y
	return int(math.Round(dipFromGLFWPixel(float64(windowX-mx), s))), int(math.Round(dipFromGLFWPixel(float64(windowY-my), s)))
}

// setWindowPositionInDIP sets the window position.
//
// x and y are the position in device-independent pixels.
//
// callSetPos reports whether to move the window, and is false to record a position it already has.
//
// setWindowPositionInDIP must be called from the main thread.
func (u *glfwBackend) setWindowPositionInDIP(x, y int, monitor *Monitor, callSetPos bool) error {
	if microsoftgdk.IsXbox() {
		// Do nothing. The position is always fixed.
		return nil
	}

	u.windowXInDIP = x
	u.windowYInDIP = y

	if !callSetPos {
		return nil
	}

	f, err := u.isFullscreen()
	if err != nil {
		return err
	}

	px, py := windowPositionInGLFWPixels(x, y, monitor)
	px, py, err = u.adjustWindowPosition(px, py, monitor)
	if err != nil {
		return err
	}
	if f {
		// The window keeps its fullscreen position, so update the position it is restored to
		// instead, as setWindowSizeInDIP does for the size.
		u.windowToRestore.pos = image.Pt(px, py)
	} else {
		if err := u.window.SetPos(px, py); err != nil {
			return err
		}
	}

	return nil
}

// setWindowTitle must be called from the main thread.
func (u *glfwBackend) setWindowTitle(title string) error {
	return u.window.SetTitle(title)
}

// isWindowMaximized must be called from the main thread.
func (u *glfwBackend) isWindowMaximized() (bool, error) {
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

// setWindowMousePassthrough must be called from the main thread.
func (u *glfwBackend) setWindowMousePassthrough(enabled bool) error {
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

func (u *glfwBackend) RunOnMainThread(f func()) {
	thread.Call(u.mainThread, f)
}

func (u *glfwBackend) run(game Game, options *RunOptions) error {
	if options.SingleThread || buildTagSingleThread || runtime.GOOS == "js" {
		return u.runSingleThread(game, options)
	}
	return u.runMultiThread(game, options)
}

func (u *glfwBackend) runMultiThread(game Game, options *RunOptions) error {
	u.mainThread = thread.NewOSThread()
	graphicscommand.SetOSThreadAsRenderThread()

	u.context = newContext(game, options.ScreenTransparent)

	ctx, cancel := stdcontext.WithCancel(stdcontext.Background())
	defer cancel()

	var wg errgroup.Group

	// Run the render thread.
	wg.Go(func() error {
		defer cancel()

		graphicscommand.LoopRenderThread(ctx)
		return nil
	})

	// Run the game thread.
	wg.Go(func() error {
		defer cancel()

		// The backend is published at the window creation in initOnMainThread.
		defer u.setRunningBackend(nil)

		type args struct {
			u       *glfwBackend
			options *RunOptions
		}
		if err := thread.CallWithArgAndResult(u.mainThread, func(a args) error {
			return a.u.initOnMainThread(a.options)
		}, args{u: u, options: options}); err != nil {
			return err
		}

		return u.loopGame()
	})

	// Run the main thread. The loop is the thread's whole life, so a call arriving after
	// it ends is a no-op rather than a block forever.
	_ = u.mainThread.LoopAndStop(ctx)
	return wg.Wait()
}

func (u *glfwBackend) runSingleThread(game Game, options *RunOptions) error {
	// Initialize the main thread first so the thread is available at u.run (#809).
	u.mainThread = thread.NewNoopThread()

	// The backend is published at the window creation in initOnMainThread.
	defer u.setRunningBackend(nil)

	u.context = newContext(game, options.ScreenTransparent)

	if err := u.initOnMainThread(options); err != nil {
		return err
	}

	if err := u.loopGame(); err != nil {
		return err
	}

	return nil
}

func dipToNativePixels(x float64, scale float64) float64 {
	return dipToGLFWPixel(x, scale)
}
