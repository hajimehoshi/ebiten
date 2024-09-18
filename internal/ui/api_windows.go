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

package ui

import (
	"errors"
	"fmt"
	"runtime"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	_CLSCTX_INPROC_SERVER     = 0x1
	_CLSCTX_LOCAL_SERVER      = 0x4
	_CLSCTX_REMOTE_SERVER     = 0x10
	_CLSCTX_SERVER            = _CLSCTX_INPROC_SERVER | _CLSCTX_LOCAL_SERVER | _CLSCTX_REMOTE_SERVER
	_MONITOR_DEFAULTTONEAREST = 2
	_SM_CYCAPTION             = 4
)

var (
	_CLSID_TaskbarList = windows.GUID{
		Data1: 0x56FDF344,
		Data2: 0xFD6D,
		Data3: 0x11D0,
		Data4: [...]byte{0x95, 0x8A, 0x00, 0x60, 0x97, 0xC9, 0xA0, 0x90},
	}
	_IID_ITaskbarList = windows.GUID{
		Data1: 0x56FDF342,
		Data2: 0xFD6D,
		Data3: 0x11D0,
		Data4: [...]byte{0x95, 0x8A, 0x00, 0x60, 0x97, 0xC9, 0xA0, 0x90},
	}
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
	imm32  = windows.NewLazySystemDLL("imm32.dll")
	ole32  = windows.NewLazySystemDLL("ole32.dll")
	user32 = windows.NewLazySystemDLL("user32.dll")

	procImmAssociateContext = imm32.NewProc("ImmAssociateContext")

	procCoCreateInstance = ole32.NewProc("CoCreateInstance")

	procGetSystemMetrics  = user32.NewProc("GetSystemMetrics")
	procMonitorFromWindow = user32.NewProc("MonitorFromWindow")
	procGetMonitorInfoW   = user32.NewProc("GetMonitorInfoW")
	procGetCursorPos      = user32.NewProc("GetCursorPos")
)

func _ImmAssociateContext(hwnd windows.HWND, hIMC uintptr) (uintptr, error) {
	r, _, e := procImmAssociateContext.Call(uintptr(hwnd), hIMC)
	if e != nil && !errors.Is(e, windows.ERROR_SUCCESS) {
		return 0, fmt.Errorf("ui: ImmAssociateContext failed: error code: %w", e)
	}
	return r, nil
}

func _CoCreateInstance(rclsid *windows.GUID, pUnkOuter unsafe.Pointer, dwClsContext uint32, riid *windows.GUID) (unsafe.Pointer, error) {
	var ptr unsafe.Pointer
	r, _, _ := procCoCreateInstance.Call(uintptr(unsafe.Pointer(rclsid)), uintptr(pUnkOuter), uintptr(dwClsContext), uintptr(unsafe.Pointer(riid)), uintptr(unsafe.Pointer(&ptr)))
	runtime.KeepAlive(rclsid)
	runtime.KeepAlive(riid)
	if uint32(r) != uint32(windows.S_OK) {
		return nil, fmt.Errorf("ui: CoCreateInstance failed: error code: HRESULT(%d)", uint32(r))
	}
	return ptr, nil
}

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
		if e != nil && !errors.Is(e, windows.ERROR_SUCCESS) {
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
		if e != nil && !errors.Is(e, windows.ERROR_SUCCESS) {
			return 0, 0, fmt.Errorf("ui: GetCursorPos failed: error code: %w", e)
		}
		return 0, 0, fmt.Errorf("ui: GetCursorPos failed: returned 0")
	}
	return pt.x, pt.y, nil
}

type _ITaskbarList struct {
	vtbl *_ITaskbarList_Vtbl
}

type _ITaskbarList_Vtbl struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr

	HrInit       uintptr
	AddTab       uintptr
	DeleteTab    uintptr
	ActivateTab  uintptr
	SetActiveAlt uintptr
}

func (i *_ITaskbarList) DeleteTab(hwnd windows.HWND) error {
	r, _, _ := syscall.Syscall(i.vtbl.DeleteTab, 2, uintptr(unsafe.Pointer(i)), uintptr(hwnd), 0)
	if uint32(r) != uint32(windows.S_OK) {
		return fmt.Errorf("ui: ITaskbarList::DeleteTab failed: HRESULT(%d)", uint32(r))
	}
	return nil
}

func (i *_ITaskbarList) Release() {
	_, _, _ = syscall.Syscall(i.vtbl.Release, 1, uintptr(unsafe.Pointer(i)), 0, 0)
}
