// Copyright 2022 The Ebiten Authors
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
	"unsafe"

	"golang.org/x/sys/windows"
)

const is64bit = unsafe.Sizeof(uintptr(0)) == 8

type handleError windows.Handle

func (h handleError) Error() string {
	return fmt.Sprintf("HANDLE(%d)", h)
}

type (
	_BOOL int32
)

func boolToUintptr(v bool) uintptr {
	if v {
		return 1
	}
	return 0
}

type _PAPPSTATE_CHANGE_ROUTINE func(quiesced bool, context unsafe.Pointer) uintptr

var (
	// https://github.com/MicrosoftDocs/sdk-api/blob/docs/sdk-api-src/content/appnotify/nf-appnotify-registerappstatechangenotification.md
	appnotify = windows.NewLazySystemDLL("API-MS-Win-Core-psm-appnotify-l1-1-0.dll")

	procRegisterAppStateChangeNotification = appnotify.NewProc("RegisterAppStateChangeNotification")
)

func _RegisterAppStateChangeNotification(routine _PAPPSTATE_CHANGE_ROUTINE, context unsafe.Pointer) (unsafe.Pointer, error) {
	cb := windows.NewCallback(routine)
	var registration unsafe.Pointer
	r, _, _ := procRegisterAppStateChangeNotification.Call(cb, uintptr(context), uintptr(unsafe.Pointer(&registration)))
	if windows.Errno(r) != windows.ERROR_SUCCESS {
		return nil, fmt.Errorf("directx: RegisterAppStateChangeNotification failed: %w", windows.Errno(r))
	}
	return registration, nil
}
