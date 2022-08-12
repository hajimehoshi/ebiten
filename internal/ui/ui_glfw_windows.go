// Copyright 2016 Hajime Hoshi
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

//go:build !nintendosdk
// +build !nintendosdk

package ui

import (
	"errors"
	"fmt"
	"runtime"

	"golang.org/x/sys/windows"

	"github.com/hajimehoshi/ebiten/v2/internal/glfw"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/directx"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/opengl"
	"github.com/hajimehoshi/ebiten/v2/internal/microsoftgdk"
)

type graphicsDriverCreatorImpl struct {
	transparent bool
}

func (g *graphicsDriverCreatorImpl) newAuto() (graphicsdriver.Graphics, GraphicsLibrary, error) {
	d, err1 := g.newDirectX()
	if err1 == nil {
		return d, GraphicsLibraryDirectX, nil
	}
	o, err2 := g.newOpenGL()
	if err2 == nil {
		return o, GraphicsLibraryOpenGL, nil
	}
	return nil, GraphicsLibraryUnknown, fmt.Errorf("ui: failed to choose graphics drivers: DirectX: %v, OpenGL: %v", err1, err2)
}

func (*graphicsDriverCreatorImpl) newOpenGL() (graphicsdriver.Graphics, error) {
	return opengl.NewGraphics()
}

func (g *graphicsDriverCreatorImpl) newDirectX() (graphicsdriver.Graphics, error) {
	if g.transparent {
		return nil, fmt.Errorf("ui: DirectX is not available with a transparent window")
	}
	return directx.NewGraphics()
}

func (*graphicsDriverCreatorImpl) newMetal() (graphicsdriver.Graphics, error) {
	return nil, nil
}

type userInterfaceImplNative struct {
	origWindowPosX        int
	origWindowPosY        int
	origWindowWidthInDIP  int
	origWindowHeightInDIP int
}

// clearVideoModeScaleCache must be called from the main thread.
func clearVideoModeScaleCache() {}

// dipFromGLFWMonitorPixel must be called from the main thread.
func (u *userInterfaceImpl) dipFromGLFWMonitorPixel(x float64, monitor *glfw.Monitor) float64 {
	return x / u.deviceScaleFactor(monitor)
}

// dipFromGLFWPixel must be called from the main thread.
func (u *userInterfaceImpl) dipFromGLFWPixel(x float64, monitor *glfw.Monitor) float64 {
	return x / u.deviceScaleFactor(monitor)
}

// dipToGLFWPixel must be called from the main thread.
func (u *userInterfaceImpl) dipToGLFWPixel(x float64, monitor *glfw.Monitor) float64 {
	return x * u.deviceScaleFactor(monitor)
}

func (u *userInterfaceImpl) adjustWindowPosition(x, y int, monitor *glfw.Monitor) (int, int) {
	if microsoftgdk.IsXbox() {
		return x, y
	}

	mx, my := monitor.GetPos()
	// As the video width/height might be wrong,
	// adjust x/y at least to enable to handle the window (#328)
	if x < mx {
		x = mx
	}
	t, err := _GetSystemMetrics(_SM_CYCAPTION)
	if err != nil {
		panic(err)
	}
	if y < my+int(t) {
		y = my + int(t)
	}
	return x, y
}

func initialMonitorByOS() (*glfw.Monitor, error) {
	if microsoftgdk.IsXbox() {
		return glfw.GetPrimaryMonitor(), nil
	}

	px, py, err := _GetCursorPos()
	if err != nil {
		if errors.Is(err, windows.ERROR_ACCESS_DENIED) {
			return nil, nil
		}
		return nil, err
	}
	x, y := int(px), int(py)

	// Find the monitor including the cursor.
	for _, m := range ensureMonitors() {
		w, h := m.vm.Width, m.vm.Height
		if x >= m.x && x < m.x+w && y >= m.y && y < m.y+h {
			return m.m, nil
		}
	}

	return nil, nil
}

func monitorFromWindowByOS(w *glfw.Window) *glfw.Monitor {
	if microsoftgdk.IsXbox() {
		return glfw.GetPrimaryMonitor()
	}
	return monitorFromWin32Window(windows.HWND(w.GetWin32Window()))
}

func monitorFromWin32Window(w windows.HWND) *glfw.Monitor {
	// Get the current monitor by the window handle instead of the window position. It is because the window
	// position is not relaiable in some cases e.g. when the window is put across multiple monitors.

	m := _MonitorFromWindow(w, _MONITOR_DEFAULTTONEAREST)
	if m == 0 {
		// monitorFromWindow can return error on Wine. Ignore this.
		return nil
	}

	mi, err := _GetMonitorInfoW(m)
	if err != nil {
		panic(err)
	}

	x, y := int(mi.rcMonitor.left), int(mi.rcMonitor.top)
	for _, m := range ensureMonitors() {
		if m.x == x && m.y == y {
			return m.m
		}
	}
	return nil
}

func (u *userInterfaceImpl) nativeWindow() uintptr {
	return u.window.GetWin32Window()
}

func (u *userInterfaceImpl) isNativeFullscreen() bool {
	return false
}

func (u *userInterfaceImpl) setNativeCursor(shape CursorShape) {
	// TODO: Use native API in the future (#1571)
	u.window.SetCursor(glfwSystemCursors[shape])
}

func (u *userInterfaceImpl) isNativeFullscreenAvailable() bool {
	return false
}

func (u *userInterfaceImpl) setNativeFullscreen(fullscreen bool) {
	panic(fmt.Sprintf("ui: setNativeFullscreen is not implemented in this environment: %s", runtime.GOOS))
}

func (u *userInterfaceImpl) adjustViewSize() {
}

func (u *userInterfaceImpl) setWindowResizingModeForOS(mode WindowResizingMode) {
}

func initializeWindowAfterCreation(w *glfw.Window) {
}

func (u *userInterfaceImpl) origWindowPos() (int, int) {
	return u.native.origWindowPosX, u.native.origWindowPosY
}

func (u *userInterfaceImpl) setOrigWindowPos(x, y int) {
	u.native.origWindowPosX = x
	u.native.origWindowPosY = y
}

func (u *userInterfaceImpl) origWindowSizeInDIP() (int, int) {
	return u.native.origWindowWidthInDIP, u.native.origWindowHeightInDIP
}

func (u *userInterfaceImpl) setOrigWindowSizeInDIP(width, height int) {
	u.native.origWindowWidthInDIP = width
	u.native.origWindowHeightInDIP = height
}

func (u *userInterfaceImplNative) initialize() {
	u.origWindowPosX = invalidPos
	u.origWindowPosY = invalidPos
}
