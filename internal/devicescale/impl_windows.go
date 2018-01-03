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
)

const logPixelSx = 88

var (
	user32 = syscall.NewLazyDLL("user32")
	gdi32  = syscall.NewLazyDLL("gdi32")
)

var (
	procSetProcessDPIAware = user32.NewProc("SetProcessDPIAware")
	procGetWindowDC        = user32.NewProc("GetWindowDC")
	procReleaseDC          = user32.NewProc("ReleaseDC")
	procGetDeviceCaps      = gdi32.NewProc("GetDeviceCaps")
)

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

func impl() float64 {
	if err := setProcessDPIAware(); err != nil {
		panic(err)
	}

	dc, err := getWindowDC(0)
	if err != nil {
		panic(err)
	}
	dpi, err := getDeviceCaps(dc, logPixelSx)
	if err != nil {
		panic(err)
	}
	if err := releaseDC(0, dc); err != nil {
		panic(err)
	}

	return float64(dpi) / 96
}
