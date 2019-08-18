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
	"image"
	"math"
	"runtime"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/internal/devicescale"
	"github.com/hajimehoshi/ebiten/internal/driver"
	"github.com/hajimehoshi/ebiten/internal/glfw"
	"github.com/hajimehoshi/ebiten/internal/thread"
)

type UserInterface struct {
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

	graphics driver.Graphics
	input    Input

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
		origPosX:            invalidPos,
		origPosY:            invalidPos,
		initCursorVisible:   true,
		initWindowDecorated: true,
		vsync:               true,
	}
)

func init() {
	theUI.input.ui = theUI
}

func Get() *UserInterface {
	return theUI
}

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
	glfw.WindowHint(glfw.Visible, glfw.False)

	// Create a window to set the initial monitor.
	w, err := glfw.CreateWindow(16, 16, "", nil, nil)
	if err != nil {
		return err
	}
	if w == nil {
		// This can happen on Windows Remote Desktop (#903).
		panic("ui: glfw.CreateWindow must not return nil")
	}

	// TODO: Fix this hack. currentMonitorImpl now requires u.window on POSIX.
	theUI.window = w
	theUI.initMonitor = theUI.currentMonitorFromPosition()
	v := theUI.initMonitor.GetVideoMode()
	s := theUI.glfwScale()
	theUI.initFullscreenWidth = int(float64(v.Width) / s)
	theUI.initFullscreenHeight = int(float64(v.Height) / s)
	theUI.window.Destroy()
	theUI.window = nil

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

func (u *UserInterface) isInitCursorVisible() bool {
	u.m.RLock()
	v := u.initCursorVisible
	u.m.RUnlock()
	return v
}

func (u *UserInterface) setInitCursorVisible(visible bool) {
	u.m.Lock()
	u.initCursorVisible = visible
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

func (u *UserInterface) isRunnableInBackground() bool {
	u.m.RLock()
	v := u.runnableInBackground
	u.m.RUnlock()
	return v
}

func (u *UserInterface) setRunnableInBackground(runnableInBackground bool) {
	u.m.Lock()
	u.runnableInBackground = runnableInBackground
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

func (u *UserInterface) ScreenSizeInFullscreen() (int, int) {
	if !u.isRunning() {
		return u.initFullscreenWidth, u.initFullscreenHeight
	}

	var v *glfw.VidMode
	s := 0.0
	_ = u.t.Call(func() error {
		v = u.currentMonitor().GetVideoMode()
		s = u.glfwScale()
		return nil
	})
	return int(float64(v.Width) / s), int(float64(v.Height) / s)
}

func (u *UserInterface) SetScreenSize(width, height int) {
	if !u.isRunning() {
		panic("ui: Run is not called yet")
	}
	_ = u.t.Call(func() error {
		// TODO: What if the window is maximized? (#320)
		u.setScreenSize(width, height, u.scale, u.isFullscreen(), u.vsync)
		return nil
	})
}

func (u *UserInterface) SetScreenScale(scale float64) {
	if !u.isRunning() {
		panic("ui: Run is not called yet")
	}
	_ = u.t.Call(func() error {
		// TODO: What if the window is maximized? (#320)
		u.setScreenSize(u.width, u.height, scale, u.isFullscreen(), u.vsync)
		return nil
	})
}

func (u *UserInterface) ScreenScale() float64 {
	if !u.isRunning() {
		return 0
	}
	s := 0.0
	_ = u.t.Call(func() error {
		s = u.scale
		return nil
	})
	return s
}

// isFullscreen must be called from the main thread.
func (u *UserInterface) isFullscreen() bool {
	if !u.isRunning() {
		panic("ui: the game must be running at isFullscreen")
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
	_ = u.t.Call(func() error {
		u.setScreenSize(u.width, u.height, u.scale, fullscreen, u.vsync)
		return nil
	})
}

func (u *UserInterface) SetRunnableInBackground(runnableInBackground bool) {
	u.setRunnableInBackground(runnableInBackground)
}

func (u *UserInterface) IsRunnableInBackground() bool {
	return u.isRunnableInBackground()
}

func (u *UserInterface) SetVsyncEnabled(enabled bool) {
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
	_ = u.t.Call(func() error {
		u.setScreenSize(u.width, u.height, u.scale, u.isFullscreen(), enabled)
		return nil
	})
}

func (u *UserInterface) IsVsyncEnabled() bool {
	u.m.RLock()
	r := u.vsync
	u.m.RUnlock()
	return r
}

func (u *UserInterface) SetWindowTitle(title string) {
	if !u.isRunning() {
		return
	}
	_ = u.t.Call(func() error {
		u.window.SetTitle(title)
		return nil
	})
}

func (u *UserInterface) SetWindowIcon(iconImages []image.Image) {
	if !u.isRunning() {
		u.setInitIconImages(iconImages)
		return
	}
	_ = u.t.Call(func() error {
		u.window.SetIcon(iconImages)
		return nil
	})
}

func (u *UserInterface) ScreenPadding() (x0, y0, x1, y1 float64) {
	if !u.isRunning() {
		return 0, 0, 0, 0
	}
	if !u.IsFullscreen() {
		if u.width == u.windowWidth {
			return 0, 0, 0, 0
		}
		// The window width can be bigger than the game screen width (#444).
		ox := 0.0
		_ = u.t.Call(func() error {
			ox = (float64(u.windowWidth)*u.actualScreenScale() - float64(u.width)*u.actualScreenScale()) / 2
			return nil
		})
		return ox, 0, ox, 0
	}

	d := 0.0
	sx := 0.0
	sy := 0.0
	gs := 0.0
	vw := 0.0
	vh := 0.0
	_ = u.t.Call(func() error {
		m := u.window.GetMonitor()
		d = devicescale.GetAt(m.GetPos())
		sx = float64(u.width) * u.actualScreenScale()
		sy = float64(u.height) * u.actualScreenScale()
		gs = u.glfwScale()

		v := m.GetVideoMode()
		vw, vh = float64(v.Width), float64(v.Height)
		return nil
	})
	mx := vw * d / gs
	my := vh * d / gs

	ox := (mx - sx) / 2
	oy := (my - sy) / 2
	return ox, oy, (mx - sx) - ox, (my - sy) - oy
}

func (u *UserInterface) adjustPosition(x, y int) (int, int) {
	if !u.isRunning() {
		return x, y
	}
	ox, oy, _, _ := u.ScreenPadding()
	s := 0.0
	_ = u.t.Call(func() error {
		s = u.actualScreenScale()
		return nil
	})
	return x - int(ox/s), y - int(oy/s)
}

func (u *UserInterface) IsCursorVisible() bool {
	if !u.isRunning() {
		return u.isInitCursorVisible()
	}
	v := false
	_ = u.t.Call(func() error {
		v = u.window.GetInputMode(glfw.CursorMode) == glfw.CursorNormal
		return nil
	})
	return v
}

func (u *UserInterface) SetCursorVisible(visible bool) {
	if !u.isRunning() {
		u.setInitCursorVisible(visible)
		return
	}
	_ = u.t.Call(func() error {
		c := glfw.CursorNormal
		if !visible {
			c = glfw.CursorHidden
		}
		u.window.SetInputMode(glfw.CursorMode, c)
		return nil
	})
}

func (u *UserInterface) IsWindowDecorated() bool {
	if !u.isRunning() {
		return u.isInitWindowDecorated()
	}
	v := false
	_ = u.t.Call(func() error {
		v = u.window.GetAttrib(glfw.Decorated) == glfw.True
		return nil
	})
	return v
}

func (u *UserInterface) SetWindowDecorated(decorated bool) {
	if !u.isRunning() {
		u.setInitWindowDecorated(decorated)
		return
	}

	panic("ui: SetWindowDecorated can't be called after Run so far.")

	// TODO: Now SetAttrib doesn't exist on GLFW 3.2. Revisit later (#556).
	// If SetAttrib exists, the implementation would be:
	//
	//     _ = u.t.Call(func() error {
	//         v := glfw.False
	//         if decorated {
	//             v = glfw.True
	//         }
	//     })
	//     u.window.SetAttrib(glfw.Decorated, v)
	//     return nil
}

func (u *UserInterface) IsWindowResizable() bool {
	if !u.isRunning() {
		return u.isInitWindowResizable()
	}
	v := false
	_ = u.t.Call(func() error {
		v = u.window.GetAttrib(glfw.Resizable) == glfw.True
		return nil
	})
	return v
}

func (u *UserInterface) SetWindowResizable(resizable bool) {
	if !u.isRunning() {
		u.setInitWindowResizable(resizable)
		return
	}

	panic("ui: SetWindowResizable can't be called after Run so far.")

	// TODO: Now SetAttrib doesn't exist on GLFW 3.2. Revisit later (#556).
}

func (u *UserInterface) DeviceScaleFactor() float64 {
	f := 0.0
	if !u.isRunning() {
		return devicescale.GetAt(u.initMonitor.GetPos())
	}

	_ = u.t.Call(func() error {
		m := u.currentMonitor()
		f = devicescale.GetAt(m.GetPos())
		return nil
	})
	return f
}

func init() {
	// Lock the main thread.
	runtime.LockOSThread()
}

func (u *UserInterface) Run(width, height int, scale float64, title string, uicontext driver.UIContext, graphics driver.Graphics) error {
	// Initialize the main thread first so the thread is available at u.run (#809).
	u.t = thread.New()
	u.graphics = graphics
	u.graphics.SetThread(u.t)

	ctx, cancel := context.WithCancel(context.Background())

	ch := make(chan error, 1)
	go func() {
		defer cancel()
		defer close(ch)
		if err := u.run(width, height, scale, title, uicontext); err != nil {
			ch <- err
		}
	}()

	u.setRunning(true)
	u.t.Loop(ctx)
	u.setRunning(false)
	return <-ch
}

func (u *UserInterface) RunWithoutMainLoop(width, height int, scale float64, title string, context driver.UIContext, graphics driver.Graphics) <-chan error {
	panic("glfw: RunWithoutMainLoop is not implemented")
}

func (u *UserInterface) run(width, height int, scale float64, title string, context driver.UIContext) error {
	_ = u.t.Call(func() error {
		if u.graphics.IsGL() {
			glfw.WindowHint(glfw.ContextVersionMajor, 2)
			glfw.WindowHint(glfw.ContextVersionMinor, 1)
		} else {
			glfw.WindowHint(glfw.ClientAPI, glfw.NoAPI)
		}

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

		if u.graphics.IsGL() {
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

			s := u.glfwScale()
			w := int(float64(width) / u.scale / s)
			h := int(float64(height) / u.scale / s)
			u.reqWidth = w
			u.reqHeight = h
		})
		return nil
	})

	var w uintptr
	_ = u.t.Call(func() error {
		w = u.nativeWindow()
		return nil
	})
	u.graphics.SetWindow(w)
	return u.loop(context)
}

// getSize must be called from the main thread.
func (u *UserInterface) glfwSize() (int, int) {
	w := int(float64(u.windowWidth) * u.getScale() * u.glfwScale())
	h := int(float64(u.height) * u.getScale() * u.glfwScale())
	return w, h
}

// getScale must be called from the main thread.
func (u *UserInterface) getScale() float64 {
	if !u.isFullscreen() {
		return u.scale
	}
	if u.fullscreenScale == 0 {
		v := u.window.GetMonitor().GetVideoMode()
		sw := float64(v.Width) / u.glfwScale() / float64(u.width)
		sh := float64(v.Height) / u.glfwScale() / float64(u.height)
		s := sw
		if s > sh {
			s = sh
		}
		u.fullscreenScale = s
	}
	return u.fullscreenScale
}

// actualScreenScale must be called from the main thread.
func (u *UserInterface) actualScreenScale() float64 {
	// Avoid calling monitor.GetPos if we have the monitor position cached already.
	if cm, ok := getCachedMonitor(u.window.GetPos()); ok {
		return u.getScale() * devicescale.GetAt(cm.x, cm.y)
	}
	return u.getScale() * devicescale.GetAt(u.currentMonitor().GetPos())
}

func (u *UserInterface) updateSize(context driver.UIContext) {
	actualScale := 0.0
	sizeChanged := false
	// TODO: Is it possible to reduce 'runOnMainThread' calls?
	_ = u.t.Call(func() error {
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
		context.SetSize(u.width, u.height, actualScale)
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

	_ = u.t.Call(func() error {
		if u.isInitFullscreen() {
			u.setScreenSize(u.width, u.height, u.scale, true, u.vsync)
			u.setInitFullscreen(false)
		}
		return nil
	})

	// This call is needed for initialization.
	u.updateSize(context)

	_ = u.t.Call(func() error {
		glfw.PollEvents()

		u.input.update(u.window, u.getScale()*u.glfwScale())

		defer context.ResumeAudio()

		for !u.isRunnableInBackground() && u.window.GetAttrib(glfw.Focused) == 0 {
			context.SuspendAudio()
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
	_ = u.t.Call(func() error {
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

func (u *UserInterface) loop(context driver.UIContext) error {
	defer func() {
		_ = u.t.Call(func() error {
			glfw.Terminate()
			return nil
		})
	}()
	for {
		if err := u.update(context); err != nil {
			return err
		}

		u.m.RLock()
		vsync := u.vsync
		u.m.RUnlock()

		_ = u.t.Call(func() error {
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
func (u *UserInterface) swapBuffers() {
	if u.graphics.IsGL() {
		u.window.SwapBuffers()
	}
}

// setScreenSize must be called from the main thread.
func (u *UserInterface) setScreenSize(width, height int, scale float64, fullscreen bool, vsync bool) bool {
	if u.width == width && u.height == height && u.scale == scale && u.isFullscreen() == fullscreen && u.vsync == vsync {
		return false
	}
	u.forceSetScreenSize(width, height, scale, fullscreen, vsync)
	return true
}

// forceSetScreenSize must be called from the main thread.
func (u *UserInterface) forceSetScreenSize(width, height int, scale float64, fullscreen bool, vsync bool) {
	// On Windows, giving a too small width doesn't call a callback (#165).
	// To prevent hanging up, return asap if the width is too small.
	// 252 is an arbitrary number and I guess this is small enough.
	minWindowWidth := 252
	if u.window.GetAttrib(glfw.Decorated) == glfw.False {
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

	if u.graphics.IsGL() {
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
	u.graphics.SetVsyncEnabled(vsync)

	u.toChangeSize = true
}

// currentMonitor returns the monitor most suitable with the current window.
//
// currentMonitor must be called on the main thread.
func (u *UserInterface) currentMonitor() *glfw.Monitor {
	w := u.window
	if m := w.GetMonitor(); m != nil {
		return m
	}
	// Get the monitor which the current window belongs to. This requires OS API.
	return u.currentMonitorFromPosition()
}

func (u *UserInterface) Input() driver.Input {
	return &u.input
}
