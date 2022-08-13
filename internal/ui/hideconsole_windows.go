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

//go:build !nintendosdk
// +build !nintendosdk

package ui

import (
	"fmt"

	"golang.org/x/sys/windows"
)

var (
	kernel32 = windows.NewLazySystemDLL("kernel32.dll")

	procFreeConsoleWindow = kernel32.NewProc("FreeConsole")
	procGetConsoleWindow  = kernel32.NewProc("GetConsoleWindow")
)

func freeConsole() error {
	r, _, e := procFreeConsoleWindow.Call()
	if int32(r) == 0 {
		if e != nil && e != windows.ERROR_SUCCESS {
			return fmt.Errorf("ui: FreeConsole failed: %w", e)
		}
		return fmt.Errorf("ui: FreeConsole returned 0")
	}
	return nil
}

func getConsoleWindow() windows.HWND {
	r, _, _ := procGetConsoleWindow.Call()
	return windows.HWND(r)
}

// hideConsoleWindowOnWindows will hide the console window that is showing when
// compiling on Windows without specifying the '-ldflags "-Hwindowsgui"' flag.
func hideConsoleWindowOnWindows() {
	// In Xbox, GetWindowThreadProcessId might not exist.
	if user32.NewProc("GetWindowThreadProcessId").Find() != nil {
		return
	}

	pid := windows.GetCurrentProcessId()

	// Get the process ID of the console's creator.
	var cpid uint32
	if _, err := windows.GetWindowThreadProcessId(getConsoleWindow(), &cpid); err != nil {
		// Even if closing the console fails, this is not harmful.
		// Ignore error.
		return
	}

	if pid == cpid {
		// The current process created its own console. Hide this.
		// Ignore error.
		freeConsole()
	}
}
