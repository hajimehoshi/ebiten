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

package ui

import (
	"errors"
	"fmt"
	"runtime"
	"syscall"

	"golang.org/x/sys/windows"

	"github.com/hajimehoshi/ebiten/v2/internal/glfw"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/directx"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/opengl"
	"github.com/hajimehoshi/ebiten/v2/internal/microsoftgdk"
	"github.com/hajimehoshi/ebiten/v2/internal/winver"
)

type graphicsDriverCreatorImpl struct {
	transparent bool
}

func (g *graphicsDriverCreatorImpl) newAuto() (graphicsdriver.Graphics, GraphicsLibrary, error) {
	var dxErr error
	var glErr error
	if winver.IsWindows10OrGreater() {
		d, err := g.newDirectX()
		if err == nil {
			return d, GraphicsLibraryDirectX, nil
		}
		dxErr = err

		o, err := g.newOpenGL()
		if err == nil {
			return o, GraphicsLibraryOpenGL, nil
		}
		glErr = err
	} else {
		// Creating a swap chain on an older machine than Windows 10 might fail (#2613).
		// Prefer OpenGL to DirectX.
		o, err := g.newOpenGL()
		if err == nil {
			return o, GraphicsLibraryOpenGL, nil
		}
		glErr = err

		// Initializing OpenGL can fail, though this is pretty rare.
		d, err := g.newDirectX()
		if err == nil {
			return d, GraphicsLibraryDirectX, nil
		}
		dxErr = err
	}

	return nil, GraphicsLibraryUnknown, fmt.Errorf("ui: failed to choose graphics drivers: DirectX: %v, OpenGL: %v", dxErr, glErr)
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
	for _, m := range theMonitors.append(nil) {
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
	// position is not reliable in some cases e.g. when the window is put across multiple monitors.

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
	for _, m := range theMonitors.append(nil) {
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

func (u *userInterfaceImpl) isNativeFullscreenAvailable() bool {
	return false
}

func (u *userInterfaceImpl) setNativeFullscreen(fullscreen bool) {
	panic(fmt.Sprintf("ui: setNativeFullscreen is not implemented in this environment: %s", runtime.GOOS))
}

func (u *userInterfaceImpl) adjustViewSizeAfterFullscreen() {
}

func (u *userInterfaceImpl) setWindowResizingModeForOS(mode WindowResizingMode) {
}

func initializeWindowAfterCreation(w *glfw.Window) {
}

func (u *userInterfaceImpl) skipTaskbar() error {
	// S_FALSE is returned when CoInitializeEx is nested. This is a successful case.
	if err := windows.CoInitializeEx(0, windows.COINIT_MULTITHREADED); err != nil && !errors.Is(err, syscall.Errno(windows.S_FALSE)) {
		return err
	}
	// CoUninitialize should be called even when CoInitializeEx returns S_FALSE.
	defer windows.CoUninitialize()

	ptr, err := _CoCreateInstance(&_CLSID_TaskbarList, nil, _CLSCTX_SERVER, &_IID_ITaskbarList)
	if err != nil {
		return err
	}

	t := (*_ITaskbarList)(ptr)
	defer t.Release()

	if err := t.DeleteTab(windows.HWND(u.window.GetWin32Window())); err != nil {
		return err
	}

	return nil
}
