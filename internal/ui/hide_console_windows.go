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
	"fmt"
	"path/filepath"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	processQueryLimitedInformation = 0x1000
)

var (
	kernel32 = windows.NewLazySystemDLL("kernel32.dll")
	user32   = windows.NewLazySystemDLL("user32.dll")

	getCurrentProcessIdProc       = kernel32.NewProc("GetCurrentProcessId")
	queryFullProcessImageNameProc = kernel32.NewProc("QueryFullProcessImageNameW")
	getConsoleWindowProc          = kernel32.NewProc("GetConsoleWindow")
	showWindowAsyncProc           = user32.NewProc("ShowWindowAsync")
)

func getCurrentProcessId() (uint32, error) {
	r, _, e := syscall.Syscall(getCurrentProcessIdProc.Addr(), 0, 0, 0, 0)
	if e != 0 {
		return 0, fmt.Errorf("ui: GetCurrentProcessId failed: %d", e)
	}
	return uint32(r), nil
}

func queryFullProcessImageName(h windows.Handle) (string, error) {
	const maxSize = 4096
	str := make([]uint16, maxSize)
	size := len(str)
	r, _, e := syscall.Syscall6(queryFullProcessImageNameProc.Addr(), 4,
		uintptr(h), 0, uintptr(unsafe.Pointer(&str[0])), uintptr(unsafe.Pointer(&size)),
		0, 0)
	if r == 0 {
		return "", fmt.Errorf("ui: QueryFullProcessImageName failed: %d", e)
	}
	return syscall.UTF16ToString(str[0:size]), nil
}

func getConsoleWindow() (uintptr, error) {
	r, _, e := syscall.Syscall(getConsoleWindowProc.Addr(), 0, 0, 0, 0)
	if e != 0 {
		return 0, fmt.Errorf("ui: GetConsoleWindow failed: %d", e)
	}
	return r, nil
}

func showWindowAsync(hwnd uintptr, show int) error {
	_, _, e := syscall.Syscall(showWindowAsyncProc.Addr(), 2, hwnd, uintptr(show), 0)
	if e != 0 {
		return fmt.Errorf("ui: ShowWindowAsync failed: %d", e)
	}
	return nil
}

func getParentProcessId() (uint32, error) {
	pid, err := getCurrentProcessId()
	if err != nil {
		return 0, err
	}
	pe := windows.ProcessEntry32{}
	pe.Size = uint32(unsafe.Sizeof(pe))
	h, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return 0, err
	}
	defer windows.CloseHandle(h)
	if err := windows.Process32First(h, &pe); err != nil {
		return 0, err
	}
	for {
		if pe.ProcessID == pid {
			return pe.ParentProcessID, nil
		}
		if err := windows.Process32Next(h, &pe); err != nil {
			return 0, err
		}
	}
	panic("not reach")
}

func getProcessName(pid uint32) (string, error) {
	h, err := windows.OpenProcess(processQueryLimitedInformation, false, pid)
	if err != nil {
		return "", err
	}
	defer windows.CloseHandle(h)
	return queryFullProcessImageName(h)
}

// hideConsoleWindowOnWindows will hide the console window that is showing when
// compiling on Windows without specifying the '-ldflags "-Hwindowsgui"' flag.
func hideConsoleWindowOnWindows() {
	ppid, err := getParentProcessId()
	if err != nil {
		// Ignore errors because:
		// 1. It is not critical if the console can't be hid.
		// 2. There is nothing to do when errors happen.
		return
	}
	name, err := getProcessName(ppid)
	if err != nil {
		// Ignore errors
		return
	}
	if filepath.Base(name) != "explorer.exe" {
		// Probably the parent process is console. The name might be
		// cmd.exe, go.exe or other terminal tools.
		return
	}
	w, err := getConsoleWindow()
	if err != nil {
		// Ignore errors
		return
	}
	showWindowAsync(w, windows.SW_HIDE)
}
