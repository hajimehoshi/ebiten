// Copyright 2026 The Ebitengine Authors
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

package directx

import (
	"fmt"
	"runtime"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// DirectComposition composites the swap chain through a visual tree instead of binding it directly
// to the window. This lets the new content be shown in sync with the window bounds while the window
// is being resized, avoiding the momentary stretch that a plain HWND swap chain shows (#3477).

var (
	dcomp = windows.NewLazySystemDLL("dcomp.dll")

	procDCompositionCreateDevice = dcomp.NewProc("DCompositionCreateDevice")
)

var _IID_IDCompositionDevice = windows.GUID{Data1: 0xc37ea93a, Data2: 0xe7aa, Data3: 0x450d, Data4: [...]byte{0xb1, 0x6f, 0x97, 0x46, 0xcb, 0x04, 0x07, 0xf3}}

func _DCompositionCreateDevice(dxgiDevice unsafe.Pointer) (*_IDCompositionDevice, error) {
	if err := procDCompositionCreateDevice.Find(); err != nil {
		return nil, err
	}
	var device *_IDCompositionDevice
	r, _, _ := procDCompositionCreateDevice.Call(uintptr(dxgiDevice), uintptr(unsafe.Pointer(&_IID_IDCompositionDevice)), uintptr(unsafe.Pointer(&device)))
	if uint32(r) != uint32(windows.S_OK) {
		return nil, fmt.Errorf("directx: DCompositionCreateDevice failed: %w", handleError(windows.Handle(uint32(r))))
	}
	return device, nil
}

type _IDCompositionDevice struct {
	vtbl *_IDCompositionDevice_Vtbl
}

type _IDCompositionDevice_Vtbl struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr

	Commit                  uintptr
	WaitForCommitCompletion uintptr
	GetFrameStatistics      uintptr
	CreateTargetForHwnd     uintptr
	CreateVisual            uintptr
	// The rest of the methods are not used.
}

func (i *_IDCompositionDevice) Commit() error {
	r, _, _ := syscall.Syscall(i.vtbl.Commit, 1, uintptr(unsafe.Pointer(i)), 0, 0)
	if uint32(r) != uint32(windows.S_OK) {
		return fmt.Errorf("directx: IDCompositionDevice::Commit failed: %w", handleError(windows.Handle(uint32(r))))
	}
	return nil
}

func (i *_IDCompositionDevice) CreateTargetForHwnd(hwnd windows.HWND, topmost bool) (*_IDCompositionTarget, error) {
	var t uintptr
	if topmost {
		t = 1
	}
	var target *_IDCompositionTarget
	r, _, _ := syscall.Syscall6(i.vtbl.CreateTargetForHwnd, 4, uintptr(unsafe.Pointer(i)), uintptr(hwnd), t, uintptr(unsafe.Pointer(&target)), 0, 0)
	if uint32(r) != uint32(windows.S_OK) {
		return nil, fmt.Errorf("directx: IDCompositionDevice::CreateTargetForHwnd failed: %w", handleError(windows.Handle(uint32(r))))
	}
	return target, nil
}

func (i *_IDCompositionDevice) CreateVisual() (*_IDCompositionVisual, error) {
	var visual *_IDCompositionVisual
	r, _, _ := syscall.Syscall(i.vtbl.CreateVisual, 2, uintptr(unsafe.Pointer(i)), uintptr(unsafe.Pointer(&visual)), 0)
	if uint32(r) != uint32(windows.S_OK) {
		return nil, fmt.Errorf("directx: IDCompositionDevice::CreateVisual failed: %w", handleError(windows.Handle(uint32(r))))
	}
	return visual, nil
}

func (i *_IDCompositionDevice) Release() uint32 {
	r, _, _ := syscall.Syscall(i.vtbl.Release, 1, uintptr(unsafe.Pointer(i)), 0, 0)
	return uint32(r)
}

type _IDCompositionTarget struct {
	vtbl *_IDCompositionTarget_Vtbl
}

type _IDCompositionTarget_Vtbl struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr

	SetRoot uintptr
}

func (i *_IDCompositionTarget) SetRoot(visual *_IDCompositionVisual) error {
	r, _, _ := syscall.Syscall(i.vtbl.SetRoot, 2, uintptr(unsafe.Pointer(i)), uintptr(unsafe.Pointer(visual)), 0)
	runtime.KeepAlive(visual)
	if uint32(r) != uint32(windows.S_OK) {
		return fmt.Errorf("directx: IDCompositionTarget::SetRoot failed: %w", handleError(windows.Handle(uint32(r))))
	}
	return nil
}

func (i *_IDCompositionTarget) Release() uint32 {
	r, _, _ := syscall.Syscall(i.vtbl.Release, 1, uintptr(unsafe.Pointer(i)), 0, 0)
	return uint32(r)
}

type _IDCompositionVisual struct {
	vtbl *_IDCompositionVisual_Vtbl
}

type _IDCompositionVisual_Vtbl struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr

	SetOffsetX_Animation       uintptr
	SetOffsetX                 uintptr
	SetOffsetY_Animation       uintptr
	SetOffsetY                 uintptr
	SetTransform_Object        uintptr
	SetTransform               uintptr
	SetTransformParent         uintptr
	SetEffect                  uintptr
	SetBitmapInterpolationMode uintptr
	SetBorderMode              uintptr
	SetClip_Object             uintptr
	SetClip                    uintptr
	SetContent                 uintptr
	// The rest of the methods are not used.
}

func (i *_IDCompositionVisual) SetContent(content unsafe.Pointer) error {
	r, _, _ := syscall.Syscall(i.vtbl.SetContent, 2, uintptr(unsafe.Pointer(i)), uintptr(content), 0)
	runtime.KeepAlive(content)
	if uint32(r) != uint32(windows.S_OK) {
		return fmt.Errorf("directx: IDCompositionVisual::SetContent failed: %w", handleError(windows.Handle(uint32(r))))
	}
	return nil
}

func (i *_IDCompositionVisual) Release() uint32 {
	r, _, _ := syscall.Syscall(i.vtbl.Release, 1, uintptr(unsafe.Pointer(i)), 0, 0)
	return uint32(r)
}
