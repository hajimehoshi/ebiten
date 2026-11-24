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

package main

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	hcAction = 0

	swpNoMove     = 0x0002
	swpNoZOrder   = 0x0004
	swpNoActivate = 0x0010

	whGetMessage = 3

	wmNull = 0x0000
)

var (
	user32 = windows.NewLazySystemDLL("user32.dll")

	procCallNextHookEx    = user32.NewProc("CallNextHookEx")
	procFindWindowW       = user32.NewProc("FindWindowW")
	procPostMessageW      = user32.NewProc("PostMessageW")
	procSetWindowPos      = user32.NewProc("SetWindowPos")
	procSetWindowsHookExW = user32.NewProc("SetWindowsHookExW")
)

// callDuringEventPolling calls f on the main thread during the next event polling.
func callDuringEventPolling(f func(setSize func(width, height int))) error {
	title, err := windows.UTF16PtrFromString(windowTitle)
	if err != nil {
		return err
	}
	hwnd, _, err := procFindWindowW.Call(0, uintptr(unsafe.Pointer(title)))
	if hwnd == 0 {
		return fmt.Errorf("FindWindowW failed: %w", err)
	}
	tid, err := windows.GetWindowThreadProcessId(windows.HWND(hwnd), nil)
	if err != nil {
		return err
	}

	var called bool
	hookProc := windows.NewCallback(func(code, wParam, lParam uintptr) uintptr {
		if code == hcAction && !called {
			called = true
			f(func(width, height int) {
				_, _, _ = procSetWindowPos.Call(hwnd, 0, 0, 0, uintptr(width), uintptr(height), swpNoMove|swpNoZOrder|swpNoActivate)
			})
		}
		r, _, _ := procCallNextHookEx.Call(0, code, wParam, lParam)
		return r
	})
	if hook, _, err := procSetWindowsHookExW.Call(whGetMessage, hookProc, 0, uintptr(tid)); hook == 0 {
		return fmt.Errorf("SetWindowsHookExW failed: %w", err)
	}

	if r, _, err := procPostMessageW.Call(hwnd, wmNull, 0, 0); r == 0 {
		return fmt.Errorf("PostMessageW failed: %w", err)
	}
	return nil
}
