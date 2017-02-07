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

package ui

import (
	"syscall"
	"unsafe"
)

// hideConsoleWindowOnWindows will hide the console window that is showing when
// compiling on Windows without specifying the '-ldflags "-Hwindowsgui"' flag.
//
// The hiding is done if the console window exists for less than 1 second
// because during development you will want to run from the command line and it
// is not supposed to be hidden whenever you 'go run' or 'go test' the program.
// That is why the work-around is to assume that the developer console is always
// open for >= 1 sec begore running the program. The end user will double-click
// the executable, it will pop up a console window and then call this function
// right away so the time between creating the console and running this function
// should never surpass 1 second. In that case the console is then hidden.
func hideConsoleWindowOnWindows() {
	consoleWindow, _, _ := getConsoleWindow.Call()
	if consoleWindow == 0 {
		return // no console attached
	}

	var consoleProcessID uint32
	getWindowThreadProcessId.Call(
		consoleWindow,
		uintptr(unsafe.Pointer(&consoleProcessID)),
	)
	if consoleProcessID == 0 {
		return // error retrieving the console process ID
	}

	const (
		PROCESS_QUERY_INFORMATION = 0x0400
		FALSE                     = 0
	)
	consoleProcess, _, _ := openProcess.Call(
		PROCESS_QUERY_INFORMATION,
		FALSE,
		uintptr(consoleProcessID),
	)
	if consoleProcess == 0 {
		return // error retrieving the console process handle
	}

	var creationTime, ignore filetime
	ok, _, _ := getProcessTimes.Call(
		uintptr(consoleProcess),
		uintptr(unsafe.Pointer(&creationTime)),
		uintptr(unsafe.Pointer(&ignore)),
		uintptr(unsafe.Pointer(&ignore)),
		uintptr(unsafe.Pointer(&ignore)),
	)
	if ok == 0 {
		return // error retrieving the process creation time
	}

	var now filetime
	getSystemTimeAsFileTime.Call(uintptr(unsafe.Pointer(&now)))

	dt := now.in100ns() - creationTime.in100ns()
	const ms = 1000 // to convert dt to milliseconds
	if dt < 1000*ms {
		// Heuristic: if the console was active for a short period of time, it
		// was probably popped up with the window after double clicking the
		// executable and not the developer typing "go run ..." from the command
		// line.
		// In this case, hide the console as this is a user playing our game.
		const SW_HIDE = 0
		showWindowAsync.Call(consoleWindow, SW_HIDE)
	}
}

var (
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	user32   = syscall.NewLazyDLL("user32.dll")

	getConsoleWindow         = kernel32.NewProc("GetConsoleWindow")
	getWindowThreadProcessId = user32.NewProc("GetWindowThreadProcessId")
	openProcess              = kernel32.NewProc("OpenProcess")
	getProcessTimes          = kernel32.NewProc("GetProcessTimes")
	getSystemTimeAsFileTime  = kernel32.NewProc("GetSystemTimeAsFileTime")
	showWindowAsync          = user32.NewProc("ShowWindowAsync")
)

type filetime struct {
	lowDateTime  uint32
	highDateTime uint32
}

func (t filetime) in100ns() uint64 {
	return uint64(t.highDateTime)<<32 | uint64(t.lowDateTime)
}
