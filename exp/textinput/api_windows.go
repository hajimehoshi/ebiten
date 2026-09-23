// Copyright 2024 The Ebitengine Authors
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

package textinput

import (
	"errors"
	"fmt"
	"runtime"
	"structs"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	_ATTR_TARGET_CONVERTED    = 0x01
	_ATTR_TARGET_NOTCONVERTED = 0x03

	_CFS_EXCLUDE = 0x0080

	_GCS_COMPATTR   = 0x0010
	_GCS_COMPCLAUSE = 0x0020
	_GCS_COMPSTR    = 0x0008
	_GCS_RESULTSTR  = 0x0800

	_GWL_WNDPROC = -4

	_IMM_ERROR_NODATA = -1

	_ISC_SHOWUICOMPOSITIONWINDOW = 0x80000000

	_UNICODE_NOCHAR = 0xffff

	_WM_CHAR               = 0x0102
	_WM_IME_COMPOSITION    = 0x010F
	_WM_IME_ENDCOMPOSITION = 0x010e
	_WM_IME_SETCONTEXT     = 0x0281
	_WM_SYSCHAR            = 0x0106
	_WM_UNICHAR            = 0x0109
)

type (
	_HIMC uintptr
)

type _CANDIDATEFORM struct {
	_            structs.HostLayout
	dwIndex      uint32
	dwStyle      uint32
	ptCurrentPos _POINT
	rcArea       _RECT
}

type _POINT struct {
	_ structs.HostLayout
	x int32
	y int32
}

type _RECT struct {
	_      structs.HostLayout
	left   int32
	top    int32
	right  int32
	bottom int32
}

var (
	imm32  = windows.NewLazySystemDLL("imm32.dll")
	user32 = windows.NewLazySystemDLL("user32.dll")

	procImmAssociateContext      = imm32.NewProc("ImmAssociateContext")
	procImmGetCompositionStringW = imm32.NewProc("ImmGetCompositionStringW")
	procImmGetContext            = imm32.NewProc("ImmGetContext")
	procImmReleaseContext        = imm32.NewProc("ImmReleaseContext")
	procImmSetCandidateWindow    = imm32.NewProc("ImmSetCandidateWindow")

	procCallWindowProcW   = user32.NewProc("CallWindowProcW")
	procGetActiveWindow   = user32.NewProc("GetActiveWindow")
	procSetWindowLongW    = user32.NewProc("SetWindowLongW")    // 32-Bit Windows version.
	procSetWindowLongPtrW = user32.NewProc("SetWindowLongPtrW") // 64-Bit Windows version.
)

func _CallWindowProcW(lpPrevWndFunc uintptr, hWnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	r, _, _ := procCallWindowProcW.Call(lpPrevWndFunc, hWnd, uintptr(msg), wParam, lParam)
	return r
}

func _GetActiveWindow() windows.HWND {
	r, _, _ := procGetActiveWindow.Call()
	return windows.HWND(r)
}

func _ImmAssociateContext(hwnd windows.HWND, hIMC uintptr) (uintptr, error) {
	r, _, e := procImmAssociateContext.Call(uintptr(hwnd), hIMC)
	if e != nil && !errors.Is(e, windows.ERROR_SUCCESS) {
		return 0, fmt.Errorf("textinput: ImmAssociateContext failed: %w", e)
	}
	return r, nil
}

func _ImmGetCompositionStringW[T byte | uint16 | uint32](unnamedParam1 _HIMC, unnamedParam2 uint32, lpBuf []T) (uint32, error) {
	var p unsafe.Pointer
	if len(lpBuf) > 0 {
		p = unsafe.Pointer(&lpBuf[0])
	}
	r, _, e := procImmGetCompositionStringW.Call(uintptr(unnamedParam1), uintptr(unnamedParam2), uintptr(p), uintptr(len(lpBuf))*unsafe.Sizeof(T(0)))
	runtime.KeepAlive(lpBuf)
	// The return value is a LONG, so check the sign as a signed 32-bit integer.
	if size := int32(r); size < 0 {
		// IMM_ERROR_NODATA indicates that the requested information is not in the input context.
		// This is an absence of data rather than a failure.
		if size == _IMM_ERROR_NODATA {
			return 0, nil
		}
		return 0, fmt.Errorf("textinput: ImmGetCompositionStringW failed: %d", size)
	}
	if e != nil && e != windows.ERROR_SUCCESS {
		return 0, fmt.Errorf("textinput: ImmGetCompositionStringW failed: %w", e)
	}
	return uint32(r), nil
}

func immGetCompositionStringW[T byte | uint16 | uint32](hIMC _HIMC, dwIndex uint32) ([]T, error) {
	size, err := _ImmGetCompositionStringW[T](hIMC, dwIndex, nil)
	if err != nil {
		return nil, err
	}
	n := size / uint32(unsafe.Sizeof(T(0)))
	if n == 0 {
		return nil, nil
	}
	lpBuf := make([]T, n)
	if _, err := _ImmGetCompositionStringW(hIMC, dwIndex, lpBuf); err != nil {
		return nil, err
	}
	return lpBuf, nil
}

func _ImmGetContext(unnamedParam1 windows.HWND) _HIMC {
	r, _, _ := procImmGetContext.Call(uintptr(unnamedParam1))
	return _HIMC(r)
}

func _ImmReleaseContext(unnamedParam1 windows.HWND, unnamedParam2 _HIMC) error {
	r, _, e := procImmReleaseContext.Call(uintptr(unnamedParam1), uintptr(unnamedParam2))
	if int32(r) == 0 {
		if e != nil && e != windows.ERROR_SUCCESS {
			return fmt.Errorf("textinput: ImmReleaseContext failed: %w", e)
		}
		return fmt.Errorf("textinput: ImmReleaseContext returned 0")
	}
	return nil
}

func _ImmSetCandidateWindow(unnamedParam1 _HIMC, lpCandidate *_CANDIDATEFORM) error {
	r, _, e := procImmSetCandidateWindow.Call(uintptr(unnamedParam1), uintptr(unsafe.Pointer(lpCandidate)))
	runtime.KeepAlive(lpCandidate)
	if int32(r) == 0 {
		if e != nil && e != windows.ERROR_SUCCESS {
			return fmt.Errorf("textinput: ImmSetCandidateWindow failed: %w", e)
		}
		return fmt.Errorf("textinput: ImmSetCandidateWindow returned 0")
	}
	return nil
}

func _SetWindowLongPtrW(hWnd windows.HWND, nIndex int32, dwNewLong uintptr) (uintptr, error) {
	var p *windows.LazyProc
	if procSetWindowLongPtrW.Find() == nil {
		// 64-Bit Windows.
		p = procSetWindowLongPtrW
	} else {
		// 32-Bit Windows.
		p = procSetWindowLongW
	}
	h, _, e := p.Call(uintptr(hWnd), uintptr(nIndex), dwNewLong)
	if h == 0 {
		if e != nil && e != windows.ERROR_SUCCESS {
			return 0, fmt.Errorf("textinput: SetWindowLongPtrW failed: %w", e)
		}
		return 0, fmt.Errorf("textinput: SetWindowLongPtrW returned 0")
	}
	return h, nil
}
