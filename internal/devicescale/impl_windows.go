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
	"syscall"
	"unsafe"
)

const (
	logPixelsX              = 88
	monitorDefaultToNull    = 0
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
	user32 = syscall.NewLazyDLL("user32")
	gdi32  = syscall.NewLazyDLL("gdi32")
	shcore = syscall.NewLazyDLL("shcore")
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

func init() {
	if shcore.Load() == nil {
		shcoreAvailable = true
	}
}

func setProcessDPIAware() error {
	r, _, e := syscall.Syscall(procSetProcessDPIAware.Addr(), 0, 0, 0, 0)
	if e != 0 {
		return fmt.Errorf("devicescale: SetProcessDPIAware failed: error code: %d", e)
	}
	if r == 0 {
		return fmt.Errorf("devicescale: SetProcessDPIAware failed: returned value: %d", r)
	}
	return nil
}

func getWindowDC(hwnd uintptr) (uintptr, error) {
	r, _, e := syscall.Syscall(procGetWindowDC.Addr(), 1, hwnd, 0, 0)
	if e != 0 {
		return 0, fmt.Errorf("devicescale: GetWindowDC failed: error code: %d", e)
	}
	if r == 0 {
		return 0, fmt.Errorf("devicescale: GetWindowDC failed: returned value: %d", r)
	}
	return r, nil
}

func releaseDC(hwnd, hdc uintptr) error {
	r, _, e := syscall.Syscall(procReleaseDC.Addr(), 2, hwnd, hdc, 0)
	if e != 0 {
		return fmt.Errorf("devicescale: ReleaseDC failed: error code: %d", e)
	}
	if r == 0 {
		return fmt.Errorf("devicescale: ReleaseDC failed: returned value: %d", r)
	}
	return nil
}

func getDeviceCaps(hdc uintptr, nindex int) (int, error) {
	r, _, e := syscall.Syscall(procGetDeviceCaps.Addr(), 2, hdc, uintptr(nindex), 0)
	if e != 0 {
		return 0, fmt.Errorf("devicescale: GetDeviceCaps failed: error code: %d", e)
	}
	return int(r), nil
}

func monitorFromRect(lprc uintptr, dwFlags int) (uintptr, error) {
	r, _, e := syscall.Syscall(procMonitorFromRect.Addr(), 2, lprc, uintptr(dwFlags), 0)
	if e != 0 {
		return 0, fmt.Errorf("devicescale: MonitorFromRect failed: error code: %d", e)
	}
	if r == 0 {
		return 0, fmt.Errorf("devicescale: MonitorFromRect failed: returned value: %d", r)
	}
	return r, nil
}

func getMonitorInfo(hMonitor uintptr, lpMonitorInfo uintptr) error {
	r, _, e := syscall.Syscall(procGetMonitorInfo.Addr(), 2, hMonitor, lpMonitorInfo, 0)
	if e != 0 {
		return fmt.Errorf("devicescale: GetMonitorInfo failed: error code: %d", e)
	}
	if r == 0 {
		return fmt.Errorf("devicescale: GetMonitorInfo failed: returned value: %d", r)
	}
	return nil
}

func getDpiForMonitor(hMonitor uintptr, dpiType uintptr, dpiX, dpiY uintptr) error {
	r, _, e := syscall.Syscall6(procGetDpiForMonitor.Addr(), 4,
		hMonitor, dpiType, dpiX, dpiY, 0, 0)
	if e != 0 {
		return fmt.Errorf("devicescale: GetDpiForMonitor failed: error code: %d", e)
	}
	if r != 0 {
		return fmt.Errorf("devicescale: GetDpiForMonitor failed: returned value: %d", r)
	}
	return nil
}

func getFromLogPixelSx() float64 {
	dc, err := getWindowDC(0)
	if err != nil {
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
	m, err := monitorFromRect(uintptr(unsafe.Pointer(&lprc)), monitorDefaultToNull)
	if err != nil {
		panic(err)
	}

	dpiX := uint32(0)
	dpiY := uint32(0) // Passing dpiY is needed even though this is not used.
	if err := getDpiForMonitor(m, mdtEffectiveDpi, uintptr(unsafe.Pointer(&dpiX)), uintptr(unsafe.Pointer(&dpiY))); err != nil {
		panic(err)
	}
	return float64(dpiX) / 96
}
