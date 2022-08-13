// Copyright 2022 The Ebitengine Authors
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
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

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
