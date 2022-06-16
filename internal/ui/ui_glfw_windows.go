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

//go:build !ebitencbackend
// +build !ebitencbackend

package ui

import (
	"fmt"
	"runtime"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/hajimehoshi/ebiten/v2/internal/glfw"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/directx"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/opengl"
	"github.com/hajimehoshi/ebiten/v2/internal/microsoftgdk"
)

type graphicsDriverGetterImpl struct {
	transparent bool
}

func (g *graphicsDriverGetterImpl) newAuto() (graphicsdriver.Graphics, error) {
	if d := g.getDirectX(); d != nil {
		return d, nil
	}
	return g.newOpenGL()
}

func (*graphicsDriverGetterImpl) newOpenGL() (graphicsdriver.Graphics, error) {
	return opengl.NewGraphics()
}

func (g *graphicsDriverGetterImpl) getDirectX() graphicsdriver.Graphics {
	if g.transparent {
		return nil
	}
	if d := directx.Get(); d != nil {
		return d
	}
	return nil
}

func (*graphicsDriverGetterImpl) getMetal() graphicsdriver.Graphics {
	return nil
}

const (
	_SM_CYCAPTION             = 4
	_MONITOR_DEFAULTTONEAREST = 2
)

type _RECT struct {
	left   int32
	top    int32
	right  int32
	bottom int32
}

type _MONITORINFO struct {
	cbSize    uint32
	rcMonitor _RECT
	rcWork    _RECT
	dwFlags   uint32
}

type _POINT struct {
	x int32
	y int32
}

var (
	user32 = windows.NewLazySystemDLL("user32.dll")

	procGetSystemMetrics  = user32.NewProc("GetSystemMetrics")
	procMonitorFromWindow = user32.NewProc("MonitorFromWindow")
	procGetMonitorInfoW   = user32.NewProc("GetMonitorInfoW")
	procGetCursorPos      = user32.NewProc("GetCursorPos")
)

func _GetSystemMetrics(nIndex int) (int32, error) {
	r, _, _ := procGetSystemMetrics.Call(uintptr(nIndex))
	if int32(r) == 0 {
		// GetLastError doesn't provide an extended information.
		// See https://docs.microsoft.com/en-us/windows/win32/api/winuser/nf-winuser-getsystemmetrics
		return 0, fmt.Errorf("ui: GetSystemMetrics returned 0")
	}
	return int32(r), nil
}

func _MonitorFromWindow(hwnd windows.HWND, dwFlags uint32) uintptr {
	r, _, _ := procMonitorFromWindow.Call(uintptr(hwnd), uintptr(dwFlags))
	return r
}

func _GetMonitorInfoW(hMonitor uintptr) (_MONITORINFO, error) {
	mi := _MONITORINFO{}
	mi.cbSize = uint32(unsafe.Sizeof(mi))

	r, _, e := procGetMonitorInfoW.Call(hMonitor, uintptr(unsafe.Pointer(&mi)))
	if int32(r) == 0 {
		if e != nil && e != windows.ERROR_SUCCESS {
			return _MONITORINFO{}, fmt.Errorf("ui: GetMonitorInfoW failed: error code: %w", e)
		}
		return _MONITORINFO{}, fmt.Errorf("ui: GetMonitorInfoW failed: returned 0")
	}
	return mi, nil
}

func _GetCursorPos() (int32, int32, error) {
	var pt _POINT
	r, _, e := procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	if int32(r) == 0 {
		if e != nil && e != windows.ERROR_SUCCESS {
			return 0, 0, fmt.Errorf("ui: GetCursorPos failed: error code: %w", e)
		}
		return 0, 0, fmt.Errorf("ui: GetCursorPos failed: returned 0")
	}
	return pt.x, pt.y, nil
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

func (u *userInterfaceImpl) origWindowPosByOS() (int, int, bool) {
	return 0, 0, false
}

func (u *userInterfaceImpl) setOrigWindowPosByOS(x, y int) bool {
	return false
}
