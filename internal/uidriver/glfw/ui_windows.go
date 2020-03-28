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

// +build !js

package glfw

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/hajimehoshi/ebiten/internal/glfw"
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
	procGetActiveWindow     = user32.NewProc("GetActiveWindow")
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

func getActiveWindow() (uintptr, error) {
	r, _, e := procGetActiveWindow.Call()
	if e != nil && e.(windows.Errno) != 0 {
		return 0, fmt.Errorf("ui: GetActiveWindow failed: error code: %d", e)
	}
	return r, nil
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

func (u *UserInterface) glfwScale() float64 {
	return u.deviceScaleFactor()
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

func (u *UserInterface) currentMonitorFromPosition() *glfw.Monitor {
	// TODO: Should we use u.window.GetWin32Window() here?
	w, err := getActiveWindow()
	if err != nil {
		panic(err)
	}
	if w == 0 {
		// There is no window at launching, but there is a hidden initialized window.
		// Get the foreground window, that is common among multiple processes.
		w, err = getForegroundWindow()
		if err != nil {
			panic(err)
		}
		if w == 0 {
			// GetForegroundWindow can return null according to the document. Use
			// the primary monitor instead.
			return glfw.GetPrimaryMonitor()
		}
	}

	// Get the current monitor by the window handle instead of the window position. It is because the window
	// position is not relaiable in some cases e.g. when the window is put across multiple monitors.

	m, err := monitorFromWindow(w, monitorDefaultToNearest)
	if err != nil {
		panic(err)
	}

	mi := monitorInfo{}
	mi.cbSize = uint32(unsafe.Sizeof(mi))
	if err := getMonitorInfoW(m, &mi); err != nil {
		panic(err)
	}

	x, y := int(mi.rcMonitor.left), int(mi.rcMonitor.top)
	for _, m := range glfw.GetMonitors() {
		mx, my := m.GetPos()
		if mx == x && my == y {
			return m
		}
	}
	return glfw.GetPrimaryMonitor()
}

func (u *UserInterface) nativeWindow() unsafe.Pointer {
	return u.window.GetWin32Window()
}
