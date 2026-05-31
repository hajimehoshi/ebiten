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

// Package exeicon extracts the icon embedded in the running executable's resources.
package exeicon

import (
	"fmt"
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	_LR_DEFAULTCOLOR = 0
)

var (
	user32 = windows.NewLazySystemDLL("user32.dll")

	procPrivateExtractIconsW = user32.NewProc("PrivateExtractIconsW")
)

// Extract returns the icon of the given pixel size embedded in the running
// executable, or a zero handle when no icon is available.
//
// The caller owns the returned handle and must release it with
// windows.DestroyIcon.
func Extract(width, height int) (windows.Handle, error) {
	// In Xbox, PrivateExtractIconsW might not exist.
	if procPrivateExtractIconsW.Find() != nil {
		return 0, nil
	}

	exe, err := os.Executable()
	if err != nil {
		return 0, fmt.Errorf("exeicon: failed to locate the executable: %w", err)
	}
	exePtr, err := windows.UTF16PtrFromString(exe)
	if err != nil {
		return 0, fmt.Errorf("exeicon: failed to encode the executable path %q: %w", exe, err)
	}

	var hicon windows.Handle
	r, _, e := procPrivateExtractIconsW.Call(
		uintptr(unsafe.Pointer(exePtr)),
		0, // nIconIndex: the first icon group, which Explorer also displays.
		uintptr(int32(width)),
		uintptr(int32(height)),
		uintptr(unsafe.Pointer(&hicon)),
		0, // piconid: not needed.
		1, // nIcons.
		_LR_DEFAULTCOLOR,
	)
	// PrivateExtractIconsW returns the number of icons extracted, 0 when the
	// executable embeds none, or 0xFFFFFFFF on error.
	if int32(r) < 0 {
		return 0, fmt.Errorf("exeicon: PrivateExtractIconsW failed: %w", e)
	}
	if hicon == 0 {
		return 0, nil
	}
	return hicon, nil
}
