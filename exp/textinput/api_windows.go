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
	"fmt"
	"runtime"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	_ATTR_TARGET_CONVERTED    = 0x01
	_ATTR_TARGET_NOTCONVERTED = 0x03

	_CFS_CANDIDATEPOS = 0x0040

	_GCS_COMPATTR   = 0x0010
	_GCS_COMPCLAUSE = 0x0020
	_GCS_COMPSTR    = 0x0008
	_GCS_RESULTSTR  = 0x0800

	_GWL_WNDPROC = -4

	_ISC_SHOWUICOMPOSITIONWINDOW = 0x80000000

	_UNICODE_NOCHAR = 0xffff

	_WM_CHAR            = 0x0102
	_WM_IME_COMPOSITION = 0x010F
	_WM_IME_SETCONTEXT  = 0x0281
	_WM_SYSCHAR         = 0x0106
	_WM_UNICHAR         = 0x0109
)

type (
	_HIMC uintptr
)

type _CANDIDATEFORM struct {
	dwIndex      uint32
	dwStyle      uint32
	ptCurrentPos _POINT
	rcArea       _RECT
}

type _POINT struct {
	x int32
	y int32
}

type _RECT struct {
	left   int32
	top    int32
	right  int32
	bottom int32
}

var (
	imm32  = windows.NewLazySystemDLL("imm32.dll")
	user32 = windows.NewLazySystemDLL("user32.dll")

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

func _ImmGetCompositionStringW(unnamedParam1 _HIMC, unnamedParam2 uint32, lpBuf unsafe.Pointer, dwBufLen uint32) (uint32, error) {
	r, _, e := procImmGetCompositionStringW.Call(uintptr(unnamedParam1), uintptr(unnamedParam2), uintptr(lpBuf), uintptr(dwBufLen))
	runtime.KeepAlive(lpBuf)
	if r < 0 {
		return 0, fmt.Errorf("textinput: ImmGetCompositionStringW failed: %d", r)
	}
	if e != nil && e != windows.ERROR_SUCCESS {
		return 0, fmt.Errorf("textinput: ImmGetCompositionStringW failed: %w", e)
	}
	return uint32(r), nil
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
