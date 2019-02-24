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

	"github.com/hajimehoshi/ebiten/internal/devicescale"
	"github.com/hajimehoshi/ebiten/internal/glfw"
	"github.com/hajimehoshi/ebiten/internal/graphicscommand"
	"github.com/hajimehoshi/ebiten/internal/hooks"
	"github.com/hajimehoshi/ebiten/internal/input"
	"github.com/hajimehoshi/ebiten/internal/mainthread"
)

type userInterface struct {
	title                string
	window               *glfw.Window
	width                int
	windowWidth          int
	height               int
	initMonitor          *glfw.Monitor
	initFullscreenWidth  int
	initFullscreenHeight int

	scale           float64
	fullscreenScale float64

	running              bool
	toChangeSize         bool
	origPosX             int
	origPosY             int
	runnableInBackground bool
	vsync                bool

	lastActualScale float64

	initFullscreen      bool
	initCursorVisible   bool
	initWindowDecorated bool
	initWindowResizable bool
	initIconImages      []image.Image

	reqWidth  int
	reqHeight int

	m sync.Mutex
}

const (
	maxInt     = int(^uint(0) >> 1)
	minInt     = -maxInt - 1
	invalidPos = minInt
)

var (
	currentUI = &userInterface{
		origPosX:            invalidPos,
		origPosY:            invalidPos,
		initCursorVisible:   true,
		initWindowDecorated: true,
		vsync:               true,
	}
)

func init() {
	hideConsoleWindowOnWindows()
	if err := initialize(); err != nil {
		panic(err)
	}
	glfw.SetMonitorCallback(func(monitor *glfw.Monitor, event glfw.MonitorEvent) {
		cacheMonitors()
	})
	cacheMonitors()
}

func initialize() error {
	if err := glfw.Init(); err != nil {
		return err
	}
	if !graphicscommand.Driver().IsGL() {
		glfw.WindowHint(glfw.ClientAPI, glfw.NoAPI)
	}
	glfw.WindowHint(glfw.Visible, glfw.False)

	// Create a window to set the initial monitor.
	w, err := glfw.CreateWindow(16, 16, "", nil, nil)
	if err != nil {
		return err
	}
	// TODO: Fix this hack. currentMonitorImpl now requires u.window on POSIX.
	currentUI.window = w
	currentUI.initMonitor = currentUI.currentMonitorFromPosition()
	v := currentUI.initMonitor.GetVideoMode()
	s := glfwScale()
	currentUI.initFullscreenWidth = int(float64(v.Width) / s)
	currentUI.initFullscreenHeight = int(float64(v.Height) / s)
	currentUI.window.Destroy()
	currentUI.window = nil

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

func Loop(ch <-chan error) error {
	currentUI.setRunning(true)
	if err := mainthread.Loop(ch); err != nil {
		return err
	}
	currentUI.setRunning(false)
	return nil
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

func (u *userInterface) isInitWindowDecorated() bool {
	u.m.Lock()
	v := u.initWindowDecorated
	u.m.Unlock()
	return v
}

func (u *userInterface) setInitWindowDecorated(decorated bool) {
	u.m.Lock()
	u.initWindowDecorated = decorated
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

func (u *userInterface) isInitWindowResizable() bool {
	u.m.Lock()
	v := u.initWindowResizable
	u.m.Unlock()
	return v
}

func (u *userInterface) setInitWindowResizable(resizable bool) {
	u.m.Lock()
	u.initWindowResizable = resizable
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

func ScreenSizeInFullscreen() (int, int) {
	u := currentUI
	if !u.isRunning() {
		return u.initFullscreenWidth, u.initFullscreenHeight
	}

	var v *glfw.VidMode
	s := 0.0
	_ = mainthread.Run(func() error {
		v = u.currentMonitor().GetVideoMode()
		s = glfwScale()
		return nil
	})
	return int(float64(v.Width) / s), int(float64(v.Height) / s)
}

func SetScreenSize(width, height int) {
	u := currentUI
	if !u.isRunning() {
		panic("ui: Run is not called yet")
	}
	_ = mainthread.Run(func() error {
		// TODO: What if the window is maximized? (#320)
		u.setScreenSize(width, height, u.scale, u.isFullscreen(), u.vsync)
		return nil
	})
}

func SetScreenScale(scale float64) bool {
	u := currentUI
	if !u.isRunning() {
		panic("ui: Run is not called yet")
	}
	r := false
	_ = mainthread.Run(func() error {
		// TODO: What if the window is maximized? (#320)
		r = u.setScreenSize(u.width, u.height, scale, u.isFullscreen(), u.vsync)
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
	_ = mainthread.Run(func() error {
		s = u.scale
		return nil
	})
	return s
}

// isFullscreen must be called from the main thread.
func (u *userInterface) isFullscreen() bool {
	if !u.isRunning() {
		panic("ui: the game must be running at isFullscreen")
	}
	return u.window.GetMonitor() != nil
}

func IsFullscreen() bool {
	u := currentUI
	if !u.isRunning() {
		return u.isInitFullscreen()
	}
	b := false
	_ = mainthread.Run(func() error {
		b = u.isFullscreen()
		return nil
	})
	return b
}

func SetFullscreen(fullscreen bool) {
	u := currentUI
	if !u.isRunning() {
		u.setInitFullscreen(fullscreen)
		return
	}
	_ = mainthread.Run(func() error {
		u := currentUI
		u.setScreenSize(u.width, u.height, u.scale, fullscreen, u.vsync)
		return nil
	})
}

func SetRunnableInBackground(runnableInBackground bool) {
	currentUI.setRunnableInBackground(runnableInBackground)
}

func IsRunnableInBackground() bool {
	return currentUI.isRunnableInBackground()
}

func SetVsyncEnabled(enabled bool) {
	u := currentUI
	if !u.isRunning() {
		// In general, m is used for locking init* values.
		// m is not used for updating vsync in setScreenSize so far, but
		// it should be OK since any goroutines can't reach here when
		// the game already starts and setScreenSize can be called.
		u.m.Lock()
		u.vsync = enabled
		u.m.Unlock()
		return
	}
	_ = mainthread.Run(func() error {
		u := currentUI
		u.setScreenSize(u.width, u.height, u.scale, u.isFullscreen(), enabled)
		return nil
	})
}

func IsVsyncEnabled() bool {
	u := currentUI
	u.m.Lock()
	r := u.vsync
	u.m.Unlock()
	return r
}

func SetWindowTitle(title string) {
	if !currentUI.isRunning() {
		return
	}
	_ = mainthread.Run(func() error {
		currentUI.window.SetTitle(title)
		return nil
	})
}

func SetWindowIcon(iconImages []image.Image) {
	if !currentUI.isRunning() {
		currentUI.setInitIconImages(iconImages)
		return
	}
	_ = mainthread.Run(func() error {
		currentUI.window.SetIcon(iconImages)
		return nil
	})
}

func ScreenPadding() (x0, y0, x1, y1 float64) {
	u := currentUI
	if !u.isRunning() {
		return 0, 0, 0, 0
	}
	if !IsFullscreen() {
		if u.width == u.windowWidth {
			return 0, 0, 0, 0
		}
		// The window width can be bigger than the game screen width (#444).
		ox := 0.0
		_ = mainthread.Run(func() error {
			ox = (float64(u.windowWidth)*u.actualScreenScale() - float64(u.width)*u.actualScreenScale()) / 2
			return nil
		})
		return ox, 0, ox, 0
	}

	m := u.window.GetMonitor()
	d := devicescale.GetAt(m.GetPos())
	v := m.GetVideoMode()

	sx := 0.0
	sy := 0.0
	gs := 0.0
	_ = mainthread.Run(func() error {
		sx = float64(u.width) * u.actualScreenScale()
		sy = float64(u.height) * u.actualScreenScale()
		gs = glfwScale()
		return nil
	})
	mx := float64(v.Width) * d / gs
	my := float64(v.Height) * d / gs

	ox := (mx - sx) / 2
	oy := (my - sy) / 2
	return ox, oy, (mx - sx) - ox, (my - sy) - oy
}

func AdjustedCursorPosition() (x, y int) {
	return adjustCursorPosition(input.Get().CursorPosition())
}

func adjustCursorPosition(x, y int) (int, int) {
	u := currentUI
	if !u.isRunning() {
		return x, y
	}
	ox, oy, _, _ := ScreenPadding()
	s := 0.0
	_ = mainthread.Run(func() error {
		s = currentUI.actualScreenScale()
		return nil
	})
	return x - int(ox/s), y - int(oy/s)
}

func AdjustedTouches() []*input.Touch {
	// TODO: Apply adjustCursorPosition
	return input.Get().Touches()
}

func IsCursorVisible() bool {
	u := currentUI
	if !u.isRunning() {
		return u.isInitCursorVisible()
	}
	v := false
	_ = mainthread.Run(func() error {
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
	_ = mainthread.Run(func() error {
		c := glfw.CursorNormal
		if !visible {
			c = glfw.CursorHidden
		}
		currentUI.window.SetInputMode(glfw.CursorMode, c)
		return nil
	})
}

func IsWindowDecorated() bool {
	u := currentUI
	if !u.isRunning() {
		return u.isInitWindowDecorated()
	}
	v := false
	_ = mainthread.Run(func() error {
		v = currentUI.window.GetAttrib(glfw.Decorated) == glfw.True
		return nil
	})
	return v
}

func SetWindowDecorated(decorated bool) {
	u := currentUI
	if !u.isRunning() {
		u.setInitWindowDecorated(decorated)
		return
	}

	panic("ui: SetWindowDecorated can't be called after Run so far.")

	// TODO: Now SetAttrib doesn't exist on GLFW 3.2. Revisit later (#556).
	// If SetAttrib exists, the implementation would be:
	//
	//     _ = mainthread.Run(func() error {
	//         v := glfw.False
	//         if decorated {
	//             v = glfw.True
	//         }
	//     })
	//     currentUI.window.SetAttrib(glfw.Decorated, v)
	//     return nil
}

func IsWindowResizable() bool {
	u := currentUI
	if !u.isRunning() {
		return u.isInitWindowResizable()
	}
	v := false
	_ = mainthread.Run(func() error {
		v = currentUI.window.GetAttrib(glfw.Resizable) == glfw.True
		return nil
	})
	return v
}

func SetWindowResizable(resizable bool) {
	if !currentUI.isRunning() {
		currentUI.setInitWindowResizable(resizable)
		return
	}

	panic("ui: SetWindowResizable can't be called after Run so far.")

	// TODO: Now SetAttrib doesn't exist on GLFW 3.2. Revisit later (#556).
}

func DeviceScaleFactor() float64 {
	f := 0.0
	u := currentUI
	if !u.isRunning() {
		return devicescale.GetAt(u.initMonitor.GetPos())
	}

	_ = mainthread.Run(func() error {
		m := u.currentMonitor()
		f = devicescale.GetAt(m.GetPos())
		return nil
	})
	return f
}

func Run(width, height int, scale float64, title string, g GraphicsContext, mainloop bool) error {
	u := currentUI
	_ = mainthread.Run(func() error {
		glfw.WindowHint(glfw.ContextVersionMajor, 2)
		glfw.WindowHint(glfw.ContextVersionMinor, 1)

		// 'decorated' must be solved before creating a window (#556).
		decorated := glfw.False
		if u.isInitWindowDecorated() {
			decorated = glfw.True
		}
		glfw.WindowHint(glfw.Decorated, decorated)

		resizable := glfw.False
		if u.isInitWindowResizable() {
			resizable = glfw.True
		}
		glfw.WindowHint(glfw.Resizable, resizable)

		// As a start, create a window with temporary size to create OpenGL context thread.
		window, err := glfw.CreateWindow(16, 16, "", nil, nil)
		if err != nil {
			return err
		}
		u.window = window

		if graphicscommand.Driver().IsGL() {
			u.window.MakeContextCurrent()
		}

		u.window.SetInputMode(glfw.StickyMouseButtonsMode, glfw.True)
		u.window.SetInputMode(glfw.StickyKeysMode, glfw.True)

		// Solve the initial properties of the window.
		mode := glfw.CursorNormal
		if !u.isInitCursorVisible() {
			mode = glfw.CursorHidden
		}
		u.window.SetInputMode(glfw.CursorMode, mode)

		if i := u.getInitIconImages(); i != nil {
			u.window.SetIcon(i)
		}

		// Get the monitor before showing the window.
		//
		// On Windows, there are two types of windows:
		//
		//   active window:     The window that has input-focus and attached to the calling thread.
		//   foreground window: The window that has input-focus: this can be in another process
		//
		// currentMonitor returns the monitor for the active window when possible and then the monitor for
		// the foreground window as fallback. In the current situation, the current window is hidden and
		// there is not the active window but the foreground window. After showing the current window, the
		// current window will be the active window. Thus, currentMonitor retuls varies before and after
		// showing the window.
		m := u.currentMonitor()
		mx, my := m.GetPos()
		v := m.GetVideoMode()

		// The game is in window mode (not fullscreen mode) at the first state.
		// Don't refer u.initFullscreen here to avoid some GLFW problems.
		u.setScreenSize(width, height, scale, false, u.vsync)
		// Get the window size before showing since window.Show might change the current
		// monitor which affects glfwSize result.
		w, h := u.glfwSize()

		u.title = title
		u.window.SetTitle(title)
		u.window.Show()

		x := mx + (v.Width-w)/2
		y := my + (v.Height-h)/3
		// Adjusting the position is needed only when the monitor is primary. (#829)
		if mx == 0 && my == 0 {
			x, y = adjustWindowPosition(x, y)
		}
		u.window.SetPos(x, y)

		u.window.SetSizeCallback(func(_ *glfw.Window, width, height int) {
			if u.window.GetAttrib(glfw.Resizable) == glfw.False {
				return
			}
			if u.isFullscreen() {
				return
			}

			s := glfwScale()
			w := int(float64(width) / u.scale / s)
			h := int(float64(height) / u.scale / s)
			u.reqWidth = w
			u.reqHeight = h
		})
		return nil
	})

	var w uintptr
	_ = mainthread.Run(func() error {
		w = u.nativeWindow()
		return nil
	})
	graphicscommand.Driver().SetWindow(w)
	return u.loop(g)
}

// getSize must be called from the main thread.
func (u *userInterface) glfwSize() (int, int) {
	w := int(float64(u.windowWidth) * u.getScale() * glfwScale())
	h := int(float64(u.height) * u.getScale() * glfwScale())
	return w, h
}

// getScale must be called from the main thread.
func (u *userInterface) getScale() float64 {
	if !u.isFullscreen() {
		return u.scale
	}
	if u.fullscreenScale == 0 {
		v := u.window.GetMonitor().GetVideoMode()
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

// actualScreenScale must be called from the main thread.
func (u *userInterface) actualScreenScale() float64 {
	// Avoid calling monitor.GetPos if we have the monitor position cached already.
	if cm, ok := getCachedMonitor(u.window.GetPos()); ok {
		return u.getScale() * devicescale.GetAt(cm.x, cm.y)
	}
	return u.getScale() * devicescale.GetAt(u.currentMonitor().GetPos())
}

// pollEvents must be called from the main thread.
func (u *userInterface) pollEvents() {
	glfw.PollEvents()
	input.Get().Update(u.window, u.getScale()*glfwScale())
}

func (u *userInterface) updateGraphicsContext(g GraphicsContext) {
	actualScale := 0.0
	sizeChanged := false
	// TODO: Is it possible to reduce 'runOnMainThread' calls?
	_ = mainthread.Run(func() error {
		actualScale = u.actualScreenScale()
		if u.lastActualScale != actualScale {
			u.forceSetScreenSize(u.width, u.height, u.scale, u.isFullscreen(), u.vsync)
		}
		u.lastActualScale = actualScale

		if !u.toChangeSize {
			return nil
		}

		u.toChangeSize = false
		sizeChanged = true
		return nil
	})
	if sizeChanged {
		g.SetSize(u.width, u.height, actualScale)
	}
}

func (u *userInterface) update(g GraphicsContext) error {
	shouldClose := false
	_ = mainthread.Run(func() error {
		shouldClose = u.window.ShouldClose()
		return nil
	})
	if shouldClose {
		return RegularTermination
	}

	_ = mainthread.Run(func() error {
		if u.isInitFullscreen() {
			u := currentUI
			u.setScreenSize(u.width, u.height, u.scale, true, u.vsync)
			u.setInitFullscreen(false)
		}
		return nil
	})

	// This call is needed for initialization.
	u.updateGraphicsContext(g)

	_ = mainthread.Run(func() error {
		u.pollEvents()
		defer hooks.ResumeAudio()
		for !u.isRunnableInBackground() && u.window.GetAttrib(glfw.Focused) == 0 {
			hooks.SuspendAudio()
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
		input.Get().ClearRuneBuffer()
		input.Get().ResetScrollValues()
		// The offscreens must be updated every frame (#490).
		u.updateGraphicsContext(g)
	}); err != nil {
		return err
	}

	// Update the screen size when the window is resizable.
	_ = mainthread.Run(func() error {
		w, h := u.reqWidth, u.reqHeight
		if w != 0 || h != 0 {
			u.setScreenSize(w, h, u.scale, u.isFullscreen(), u.vsync)
		}
		u.reqWidth = 0
		u.reqHeight = 0
		return nil
	})
	return nil
}

func (u *userInterface) loop(g GraphicsContext) error {
	defer func() {
		_ = mainthread.Run(func() error {
			glfw.Terminate()
			return nil
		})
	}()
	for {
		if err := u.update(g); err != nil {
			return err
		}

		u.m.Lock()
		vsync := u.vsync
		u.m.Unlock()

		_ = mainthread.Run(func() error {
			if !vsync {
				u.swapBuffers()
				return nil
			}
			u.swapBuffers()
			return nil
		})
	}
}

// swapBuffers must be called from the main thread.
func (u *userInterface) swapBuffers() {
	if graphicscommand.Driver().IsGL() {
		u.window.SwapBuffers()
	}
}

// setScreenSize must be called from the main thread.
func (u *userInterface) setScreenSize(width, height int, scale float64, fullscreen bool, vsync bool) bool {
	if u.width == width && u.height == height && u.scale == scale && u.isFullscreen() == fullscreen && u.vsync == vsync {
		return false
	}
	u.forceSetScreenSize(width, height, scale, fullscreen, vsync)
	return true
}

// forceSetScreenSize must be called from the main thread.
func (u *userInterface) forceSetScreenSize(width, height int, scale float64, fullscreen bool, vsync bool) {
	// On Windows, giving a too small width doesn't call a callback (#165).
	// To prevent hanging up, return asap if the width is too small.
	// 252 is an arbitrary number and I guess this is small enough.
	minWindowWidth := 252
	if currentUI.window.GetAttrib(glfw.Decorated) == glfw.False {
		minWindowWidth = 1
	}

	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}

	u.width = width
	u.windowWidth = width
	s := scale * devicescale.GetAt(u.currentMonitor().GetPos())
	if int(float64(width)*s) < minWindowWidth {
		u.windowWidth = int(math.Ceil(float64(minWindowWidth) / s))
	}
	u.height = height
	u.scale = scale
	u.fullscreenScale = 0
	u.vsync = vsync

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
	} else {
		if u.window.GetMonitor() != nil {
			// Give dummy values as the window position and size.
			// The new window position should be specifying after SetSize.
			u.window.SetMonitor(nil, 0, 0, 16, 16, 0)
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

		if u.origPosX != invalidPos && u.origPosY != invalidPos {
			x := u.origPosX
			y := u.origPosY
			u.window.SetPos(x, y)
			// Dirty hack for macOS (#703). Rendering doesn't work correctly with one SetPos, but work
			// with two or more SetPos.
			if runtime.GOOS == "darwin" {
				u.window.SetPos(x+1, y)
				u.window.SetPos(x, y)
			}
			u.origPosX = invalidPos
			u.origPosY = invalidPos
		}

		// Window title might be lost on macOS after coming back from fullscreen.
		u.window.SetTitle(u.title)
	}

	if graphicscommand.Driver().IsGL() {
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
	graphicscommand.Driver().SetVsyncEnabled(vsync)

	u.toChangeSize = true
}

// currentMonitor returns the monitor most suitable with the current window.
//
// currentMonitor must be called on the main thread.
func (u *userInterface) currentMonitor() *glfw.Monitor {
	w := u.window
	if m := w.GetMonitor(); m != nil {
		return m
	}
	// Get the monitor which the current window belongs to. This requires OS API.
	return u.currentMonitorFromPosition()
}
