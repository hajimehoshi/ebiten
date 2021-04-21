// Copyright 2018 The Ebiten Authors
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

package devicescale

import (
	"fmt"
	"runtime"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	logPixelsX              = 88
	monitorDefaultToNearest = 2
	mdtEffectiveDpi         = 0
)

type rect struct {
	left   int32
	top    int32
	right  int32
	bottom int32
}

var (
	user32 = windows.NewLazySystemDLL("user32")
	gdi32  = windows.NewLazySystemDLL("gdi32")
	shcore = windows.NewLazySystemDLL("shcore")
)

var (
	procSetProcessDPIAware = user32.NewProc("SetProcessDPIAware")
	procGetWindowDC        = user32.NewProc("GetWindowDC")
	procReleaseDC          = user32.NewProc("ReleaseDC")
	procMonitorFromRect    = user32.NewProc("MonitorFromRect")
	procGetMonitorInfo     = user32.NewProc("GetMonitorInfoW")

	procGetDeviceCaps = gdi32.NewProc("GetDeviceCaps")

	// GetScaleFactorForMonitor function can return unrelaiavle value (e.g. returning 180
	// for 200% scale). Use GetDpiForMonitor instead.
	procGetDpiForMonitor = shcore.NewProc("GetDpiForMonitor")
)

var shcoreAvailable = false

type winErr struct {
	FuncName string
	Code     windows.Errno
	Return   uintptr
}

func (e *winErr) Error() string {
	return fmt.Sprintf("devicescale: %s failed: error code: %d", e.FuncName, e.Code)
}

func init() {
	if shcore.Load() == nil {
		shcoreAvailable = true
	}
}

func setProcessDPIAware() error {
	r, _, e := procSetProcessDPIAware.Call()
	if e != nil && e.(windows.Errno) != 0 {
		return &winErr{
			FuncName: "SetProcessDPIAware",
			Code:     e.(windows.Errno),
		}
	}
	if r == 0 {
		return &winErr{
			FuncName: "SetProcessDPIAware",
			Return:   r,
		}
	}
	return nil
}

func getWindowDC(hwnd uintptr) (uintptr, error) {
	r, _, e := procGetWindowDC.Call(hwnd)
	if e != nil && e.(windows.Errno) != 0 {
		return 0, &winErr{
			FuncName: "GetWindowDC",
			Code:     e.(windows.Errno),
		}
	}
	if r == 0 {
		return 0, &winErr{
			FuncName: "GetWindowDC",
			Return:   r,
		}
	}
	return r, nil
}

func releaseDC(hwnd, hdc uintptr) error {
	r, _, e := procReleaseDC.Call(hwnd, hdc)
	if e != nil && e.(windows.Errno) != 0 {
		return &winErr{
			FuncName: "ReleaseDC",
			Code:     e.(windows.Errno),
		}
	}
	if r == 0 {
		return &winErr{
			FuncName: "ReleaseDC",
			Return:   r,
		}
	}
	return nil
}

func getDeviceCaps(hdc uintptr, nindex int) (int, error) {
	r, _, e := procGetDeviceCaps.Call(hdc, uintptr(nindex))
	if e != nil && e.(windows.Errno) != 0 {
		return 0, &winErr{
			FuncName: "GetDeviceCaps",
			Code:     e.(windows.Errno),
		}
	}
	return int(r), nil
}

func monitorFromRect(lprc *rect, dwFlags int) (uintptr, error) {
	r, _, e := procMonitorFromRect.Call(uintptr(unsafe.Pointer(lprc)), uintptr(dwFlags))
	runtime.KeepAlive(lprc)
	if e != nil && e.(windows.Errno) != 0 {
		return 0, &winErr{
			FuncName: "MonitorFromRect",
			Code:     e.(windows.Errno),
		}
	}
	if r == 0 {
		return 0, &winErr{
			FuncName: "MonitorFromRect",
			Return:   r,
		}
	}
	return r, nil
}

func getMonitorInfo(hMonitor uintptr, lpMonitorInfo uintptr) error {
	r, _, e := procGetMonitorInfo.Call(hMonitor, lpMonitorInfo)
	if e != nil && e.(windows.Errno) != 0 {
		return &winErr{
			FuncName: "GetMonitorInfo",
			Code:     e.(windows.Errno),
		}
	}
	if r == 0 {
		return &winErr{
			FuncName: "GetMonitorInfo",
			Return:   r,
		}
	}
	return nil
}

func getDpiForMonitor(hMonitor uintptr, dpiType uintptr, dpiX, dpiY *uint32) error {
	r, _, e := procGetDpiForMonitor.Call(hMonitor, dpiType, uintptr(unsafe.Pointer(dpiX)), uintptr(unsafe.Pointer(dpiY)))
	if e != nil && e.(windows.Errno) != 0 {
		return &winErr{
			FuncName: "GetDpiForMonitor",
			Code:     e.(windows.Errno),
		}
	}
	if r != 0 {
		return &winErr{
			FuncName: "GetDpiForMonitor",
			Return:   r,
		}
	}
	return nil
}

func getFromLogPixelSx() float64 {
	dc, err := getWindowDC(0)
	if err != nil {
		const (
			errorInvalidWindowHandle  = 1400
			errorResourceDataNotFound = 1812
		)
		// On Wine, it looks like GetWindowDC(0) doesn't work (#738, #743).
		code := err.(*winErr).Code
		if code == errorInvalidWindowHandle {
			return 1
		}
		if code == errorResourceDataNotFound {
			return 1
		}
		panic(err)
	}

	// Note that GetDeviceCaps with LOGPIXELSX always returns a same value for any monitors
	// even if multiple monitors are used.
	dpi, err := getDeviceCaps(dc, logPixelsX)
	if err != nil {
		panic(err)
	}

	if err := releaseDC(0, dc); err != nil {
		panic(err)
	}

	return float64(dpi) / 96
}

func impl(x, y int) float64 {
	if err := setProcessDPIAware(); err != nil {
		panic(err)
	}

	// On Windows 7 or older, shcore.dll is not available.
	if !shcoreAvailable {
		return getFromLogPixelSx()
	}

	lprc := rect{
		left:   int32(x),
		right:  int32(x + 1),
		top:    int32(y),
		bottom: int32(y + 1),
	}

	// MonitorFromPoint requires to pass a POINT value, and there seems no portable way to
	// do this with Cgo. Use MonitorFromRect instead.
	m, err := monitorFromRect(&lprc, monitorDefaultToNearest)
	if err != nil {
		// monitorFromRect can fail in some environments (#1612)
		return getFromLogPixelSx()
	}

	dpiX := uint32(0)
	dpiY := uint32(0) // Passing dpiY is needed even though this is not used, or GetDpiForMonitor returns an error.
	if err := getDpiForMonitor(m, mdtEffectiveDpi, &dpiX, &dpiY); err != nil {
		// getDpiForMonitor can fail in some environments (#1612)
		return getFromLogPixelSx()
	}
	runtime.KeepAlive(dpiY)

	return float64(dpiX) / 96
}
