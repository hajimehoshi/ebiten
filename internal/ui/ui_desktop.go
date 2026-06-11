// Copyright 2026 The Ebitengine Authors
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
	"sync"
	"sync/atomic"

	"github.com/hajimehoshi/ebiten/v2/internal/colormode"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/microsoftgdk"
)

// uiBackend is the platform UI implementation for the desktop build.
//
// A backend is created at Run and consumes the settings buffered in
// userInterfaceImpl during its initialization. All the methods except run
// are called only while the game runs.
type uiBackend interface {
	run(game Game, options *RunOptions) error
	readInputState(inputState *InputState)
	updateInputStateForFrame(deviceScaleFactor float64) error
	updateIconIfNeeded() error
	IsFocused() bool
	IsFullscreen() bool
	SetFullscreen(fullscreen bool)
	SetFPSMode(mode FPSModeType)
	ScheduleFrame()
	CursorMode() CursorMode
	SetCursorMode(mode CursorMode)
	SetCursorShape(shape CursorShape)
	Window() backendWindow
	Monitor() *Monitor
	RunOnMainThread(f func())
	KeyName(key Key) string
}

// backendWindow is the part of Window that a backend implements.
// The methods are called only while the game runs. The remaining part of
// Window is answered by desktopWindow from the settings in userInterfaceImpl.
type backendWindow interface {
	IsDecorated() bool
	SetDecorated(decorated bool)
	SetResizingMode(mode WindowResizingMode)
	SetMonitor(monitor *Monitor)
	Position() (int, int)
	SetPosition(x, y int)
	Size() (int, int)
	SetSize(width, height int)
	SetSizeLimits(minw, minh, maxw, maxh int)
	IsFloating() bool
	SetFloating(floating bool)
	Maximize()
	IsMaximized() bool
	Minimize()
	IsMinimized() bool
	SetTitle(title string)
	SetColorMode(mode colormode.ColorMode)
	Restore()
	SetClosingHandled(handled bool)
	SetMousePassthrough(enabled bool)
	IsMousePassthrough() bool
	RequestAttention()
}

var _ uiBackend = (*glfwBackend)(nil)

type userInterfaceImpl struct {
	// backend is set at Run and is never replaced nor cleared afterwards.
	//
	// Reading backend without synchronization is safe only after isRunning()
	// returns true: the backend assignment happens before the atomic store of
	// the running flag, so a goroutine observing the flag also observes the
	// assignment. Functions that skip this check must be called only from the
	// game loop, which the backend itself drives.
	backend uiBackend

	graphicsDriver graphicsdriver.Graphics
	context        *context

	// The atomic fields below hold the settings that can be set before the
	// backend exists. The backend consumes the init* fields at its
	// initialization, and reads the other fields whenever it needs them.
	// The window settings are held by desktopWindow.
	runnableOnUnfocused atomic.Bool
	fpsMode             atomic.Int32
	cursorShape         atomic.Int32

	initMonitor    atomic.Pointer[Monitor]
	initFullscreen atomic.Bool
	initCursorMode atomic.Int32

	desktopWindow desktopWindow

	glfwInitOnce sync.Once
}

func (u *UserInterface) init() error {
	u.runnableOnUnfocused.Store(true)
	u.initCursorMode.Store(int32(CursorModeVisible))

	u.desktopWindow.ui = u
	u.desktopWindow.init()

	return nil
}

func (u *UserInterface) Run(game Game, options *RunOptions) error {
	u.backend = newGLFWBackend(u)
	return u.backend.run(game, options)
}

func (u *UserInterface) setInitMonitor(m *Monitor) {
	u.initMonitor.Store(m)
}

func (u *UserInterface) getInitMonitor() *Monitor {
	return u.initMonitor.Load()
}

func (u *UserInterface) isInitFullscreen() bool {
	return u.initFullscreen.Load()
}

func (u *UserInterface) setInitFullscreen(initFullscreen bool) {
	u.initFullscreen.Store(initFullscreen)
}

func (u *UserInterface) getInitCursorMode() CursorMode {
	return CursorMode(u.initCursorMode.Load())
}

func (u *UserInterface) setInitCursorMode(mode CursorMode) {
	u.initCursorMode.Store(int32(mode))
}

func (u *UserInterface) getCursorShape() CursorShape {
	return CursorShape(u.cursorShape.Load())
}

func (u *UserInterface) isRunnableOnUnfocused() bool {
	return u.runnableOnUnfocused.Load()
}

func (u *UserInterface) setRunnableOnUnfocused(runnableOnUnfocused bool) {
	u.runnableOnUnfocused.Store(runnableOnUnfocused)
}

func (u *UserInterface) readInputState(inputState *InputState) {
	u.backend.readInputState(inputState)
}

func (u *UserInterface) updateInputStateForFrame(deviceScaleFactor float64) error {
	return u.backend.updateInputStateForFrame(deviceScaleFactor)
}

func (u *UserInterface) updateIconIfNeeded() error {
	return u.backend.updateIconIfNeeded()
}

func (u *UserInterface) IsFocused() bool {
	if !u.isRunning() {
		return false
	}
	return u.backend.IsFocused()
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
	return u.backend.IsFullscreen()
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
	u.backend.SetFullscreen(fullscreen)
}

func (u *UserInterface) IsRunnableOnUnfocused() bool {
	return u.isRunnableOnUnfocused()
}

func (u *UserInterface) SetRunnableOnUnfocused(runnableOnUnfocused bool) {
	u.setRunnableOnUnfocused(runnableOnUnfocused)
}

func (u *UserInterface) FPSMode() FPSModeType {
	return FPSModeType(u.fpsMode.Load())
}

func (u *UserInterface) SetFPSMode(mode FPSModeType) {
	if u.isTerminated() {
		return
	}
	if FPSModeType(u.fpsMode.Swap(int32(mode))) == mode {
		return
	}
	if !u.isRunning() {
		return
	}
	u.backend.SetFPSMode(mode)
}

func (u *UserInterface) ScheduleFrame() {
	if !u.isRunning() {
		return
	}
	u.backend.ScheduleFrame()
}

func (u *UserInterface) CursorMode() CursorMode {
	if u.isTerminated() {
		return 0
	}
	if !u.isRunning() {
		return u.getInitCursorMode()
	}
	return u.backend.CursorMode()
}

func (u *UserInterface) SetCursorMode(mode CursorMode) {
	if u.isTerminated() {
		return
	}
	if !u.isRunning() {
		u.setInitCursorMode(mode)
		return
	}
	u.backend.SetCursorMode(mode)
}

func (u *UserInterface) CursorShape() CursorShape {
	return u.getCursorShape()
}

func (u *UserInterface) SetCursorShape(shape CursorShape) {
	if u.isTerminated() {
		return
	}
	if CursorShape(u.cursorShape.Swap(int32(shape))) == shape {
		return
	}
	if !u.isRunning() {
		return
	}
	u.backend.SetCursorShape(shape)
}

func (u *UserInterface) Window() Window {
	if microsoftgdk.IsXbox() {
		return &nullWindow{}
	}
	return &u.desktopWindow
}

// Monitor returns the window's current monitor. Returns nil if there is no current monitor yet.
func (u *UserInterface) Monitor() *Monitor {
	if !u.isRunning() {
		// Ensure GLFW is initialized so that the init monitor is available.
		if err := u.ensureGLFWInit(); err != nil {
			return nil
		}
		return u.getInitMonitor()
	}
	return u.backend.Monitor()
}

// AppendMonitors appends the current monitors to the passed in mons slice and returns it.
func (u *UserInterface) AppendMonitors(monitors []*Monitor) []*Monitor {
	// Ensure GLFW is initialized so that the monitor list is available.
	if err := u.ensureGLFWInit(); err != nil {
		return monitors
	}
	return theMonitors.append(monitors)
}

func (u *UserInterface) RunOnMainThread(f func()) {
	u.backend.RunOnMainThread(f)
}

func (u *UserInterface) KeyName(key Key) string {
	if !u.isRunning() {
		return ""
	}
	return u.backend.KeyName(key)
}
