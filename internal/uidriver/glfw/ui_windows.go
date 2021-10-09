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

package glfw

import (
	"fmt"
	"runtime"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/hajimehoshi/ebiten/v2/internal/driver"
	"github.com/hajimehoshi/ebiten/v2/internal/glfw"
)

const (
	smCyCaption             = 4
	monitorDefaultToNearest = 2
)

type rect struct {
	left   int32
	top    int32
	right  int32
	bottom int32
}

type monitorInfo struct {
	cbSize    uint32
	rcMonitor rect
	rcWork    rect
	dwFlags   uint32
}

var (
	// user32 is defined at hideconsole_windows.go
	procGetSystemMetrics    = user32.NewProc("GetSystemMetrics")
	procGetForegroundWindow = user32.NewProc("GetForegroundWindow")
	procMonitorFromWindow   = user32.NewProc("MonitorFromWindow")
	procGetMonitorInfoW     = user32.NewProc("GetMonitorInfoW")
)

func getSystemMetrics(nIndex int) (int, error) {
	r, _, e := procGetSystemMetrics.Call(uintptr(nIndex))
	if e != nil && e.(windows.Errno) != 0 {
		return 0, fmt.Errorf("ui: GetSystemMetrics failed: error code: %d", e)
	}
	return int(r), nil
}

func getForegroundWindow() (uintptr, error) {
	r, _, e := procGetForegroundWindow.Call()
	if e != nil && e.(windows.Errno) != 0 {
		return 0, fmt.Errorf("ui: GetForegroundWindow failed: error code: %d", e)
	}
	return r, nil
}

func monitorFromWindow(hwnd uintptr, dwFlags uint32) (uintptr, error) {
	r, _, e := procMonitorFromWindow.Call(hwnd, uintptr(dwFlags))
	if e != nil && e.(windows.Errno) != 0 {
		return 0, fmt.Errorf("ui: MonitorFromWindow failed: error code: %d", e)
	}
	if r == 0 {
		return 0, fmt.Errorf("ui: MonitorFromWindow failed: returned value: %d", r)
	}
	return r, nil
}

func getMonitorInfoW(hMonitor uintptr, lpmi *monitorInfo) error {
	r, _, e := procGetMonitorInfoW.Call(hMonitor, uintptr(unsafe.Pointer(lpmi)))
	if e != nil && e.(windows.Errno) != 0 {
		return fmt.Errorf("ui: GetMonitorInfoW failed: error code: %d", e)
	}
	if r == 0 {
		return fmt.Errorf("ui: GetMonitorInfoW failed: returned value: %d", r)
	}
	return nil
}

// clearVideoModeScaleCache must be called from the main thread.
func clearVideoModeScaleCache() {}

// fromGLFWMonitorPixel must be called from the main thread.
func (u *UserInterface) fromGLFWMonitorPixel(x float64, monitor *glfw.Monitor) float64 {
	return x / u.deviceScaleFactor(monitor)
}

// fromGLFWPixel must be called from the main thread.
func (u *UserInterface) fromGLFWPixel(x float64, monitor *glfw.Monitor) float64 {
	return x / u.deviceScaleFactor(monitor)
}

// toGLFWPixel must be called from the main thread.
func (u *UserInterface) toGLFWPixel(x float64, monitor *glfw.Monitor) float64 {
	return x * u.deviceScaleFactor(monitor)
}

func (u *UserInterface) adjustWindowPosition(x, y int) (int, int) {
	mx, my := u.currentMonitor().GetPos()
	// As the video width/height might be wrong,
	// adjust x/y at least to enable to handle the window (#328)
	if x < mx {
		x = mx
	}
	t, err := getSystemMetrics(smCyCaption)
	if err != nil {
		panic(err)
	}
	if y < my+t {
		y = my + t
	}
	return x, y
}

func initialMonitorByOS() *glfw.Monitor {
	// Get the foreground window, that is common among multiple processes.
	w, err := getForegroundWindow()
	if err != nil {
		panic(err)
	}
	if w == 0 {
		// GetForegroundWindow can return null according to the document.
		return nil
	}
	return monitorFromWin32Window(w)
}

func currentMonitorByOS(w *glfw.Window) *glfw.Monitor {
	return monitorFromWin32Window(w.GetWin32Window())
}

func monitorFromWin32Window(w uintptr) *glfw.Monitor {
	// Get the current monitor by the window handle instead of the window position. It is because the window
	// position is not relaiable in some cases e.g. when the window is put across multiple monitors.

	m, err := monitorFromWindow(w, monitorDefaultToNearest)
	if err != nil {
		// monitorFromWindow can return error on Wine. Ignore this.
		return nil
	}

	mi := monitorInfo{}
	mi.cbSize = uint32(unsafe.Sizeof(mi))
	if err := getMonitorInfoW(m, &mi); err != nil {
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

func (u *UserInterface) nativeWindow() uintptr {
	return u.window.GetWin32Window()
}

func (u *UserInterface) isNativeFullscreen() bool {
	return false
}

func (u *UserInterface) setNativeCursor(shape driver.CursorShape) {
	// TODO: Use native API in the future (#1571)
	u.window.SetCursor(glfwSystemCursors[shape])
}

func (u *UserInterface) isNativeFullscreenAvailable() bool {
	return false
}

func (u *UserInterface) setNativeFullscreen(fullscreen bool) {
	panic(fmt.Sprintf("glfw: setNativeFullscreen is not implemented in this environment: %s", runtime.GOOS))
}

func (u *UserInterface) adjustViewSize() {
}

func initializeWindowAfterCreation(w *glfw.Window) {
}
