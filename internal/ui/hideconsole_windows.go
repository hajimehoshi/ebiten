// Copyright 2017 The Ebiten Authors
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

//go:build !ebitencbackend
// +build !ebitencbackend

package ui

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	processQueryLimitedInformation = 0x1000
)

var (
	kernel32 = windows.NewLazySystemDLL("kernel32.dll")
	user32   = windows.NewLazySystemDLL("user32.dll")

	procFreeConsoleWindow        = kernel32.NewProc("FreeConsole")
	procGetCurrentProcessId      = kernel32.NewProc("GetCurrentProcessId")
	procGetConsoleWindow         = kernel32.NewProc("GetConsoleWindow")
	procGetWindowThreadProcessId = user32.NewProc("GetWindowThreadProcessId")
)

func freeConsole() error {
	r, _, e := procFreeConsoleWindow.Call()
	if r == 0 {
		if e != nil && e != windows.ERROR_SUCCESS {
			return fmt.Errorf("ui: FreeConsole failed: %w", e)
		}
		return fmt.Errorf("ui: FreeConsole returned 0")
	}
	return nil
}

func getCurrentProcessId() uint32 {
	r, _, _ := procGetCurrentProcessId.Call()
	return uint32(r)
}

func getConsoleWindow() windows.HWND {
	r, _, _ := procGetConsoleWindow.Call()
	return windows.HWND(r)
}

func getWindowThreadProcessId(hwnd windows.HWND) (tid, pid uint32) {
	r, _, _ := procGetWindowThreadProcessId.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&pid)))
	tid = uint32(r)
	return
}

// hideConsoleWindowOnWindows will hide the console window that is showing when
// compiling on Windows without specifying the '-ldflags "-Hwindowsgui"' flag.
func hideConsoleWindowOnWindows() {
	pid := getCurrentProcessId()
	// Get the process ID of the console's creator.
	_, cpid := getWindowThreadProcessId(getConsoleWindow())
	if pid == cpid {
		// The current process created its own console. Hide this.
		// Ignore error.
		freeConsole()
	}
}
