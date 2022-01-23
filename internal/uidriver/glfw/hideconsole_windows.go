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

package glfw

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

	getCurrentProcessIdProc      = kernel32.NewProc("GetCurrentProcessId")
	getConsoleWindowProc         = kernel32.NewProc("GetConsoleWindow")
	freeConsoleWindowProc        = kernel32.NewProc("FreeConsole")
	getWindowThreadProcessIdProc = user32.NewProc("GetWindowThreadProcessId")
)

func getCurrentProcessId() (uint32, error) {
	r, _, e := getCurrentProcessIdProc.Call()
	if e != nil && e != windows.ERROR_SUCCESS {
		return 0, fmt.Errorf("ui: GetCurrentProcessId failed: %w", e)
	}
	return uint32(r), nil
}

func getWindowThreadProcessId(hwnd uintptr) (uint32, error) {
	pid := uint32(0)
	r, _, e := getWindowThreadProcessIdProc.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
	if e != nil && e != windows.ERROR_SUCCESS {
		return 0, fmt.Errorf("ui: GetWindowThreadProcessId failed: %w", e)
	}
	if r == 0 {
		return 0, fmt.Errorf("ui: GetWindowThreadProcessId returned 0")
	}
	return pid, nil
}

func getConsoleWindow() (uintptr, error) {
	r, _, e := getConsoleWindowProc.Call()
	if e != nil && e != windows.ERROR_SUCCESS {
		return 0, fmt.Errorf("ui: GetConsoleWindow failed: %w", e)
	}
	return r, nil
}

func freeConsole() error {
	if _, _, e := freeConsoleWindowProc.Call(); e != nil && e != windows.ERROR_SUCCESS {
		return fmt.Errorf("ui: FreeConsole failed: %w", e)
	}
	return nil
}

// hideConsoleWindowOnWindows will hide the console window that is showing when
// compiling on Windows without specifying the '-ldflags "-Hwindowsgui"' flag.
func hideConsoleWindowOnWindows() {
	pid, err := getCurrentProcessId()
	if err != nil {
		// Ignore errors because:
		// 1. It is not critical if the console can't be hid.
		// 2. There is nothing to do when errors happen.
		return
	}
	w, err := getConsoleWindow()
	if err != nil {
		// Ignore errors
		return
	}
	// Get the process ID of the console's creator.
	cpid, err := getWindowThreadProcessId(w)
	if err != nil {
		// Ignore errors
		return
	}
	if pid == cpid {
		// The current process created its own console. Hide this.
		// Ignore error
		freeConsole()
	}
}
