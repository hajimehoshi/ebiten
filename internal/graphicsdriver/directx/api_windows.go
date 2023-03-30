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
	"runtime"
	"unsafe"

	"golang.org/x/sys/windows"
)

const is64bit = unsafe.Sizeof(uintptr(0)) == 8

const (
	_VER_BUILDNUMBER   = 0x00000004
	_VER_GREATER_EQUAL = 3
	_VER_MAJORVERSION  = 0x00000002
	_VER_MINORVERSION  = 0x00000001
)

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

type _OSVERSIONINFOEXW struct {
	dwOSVersionInfoSize uint32
	dwMajorVersion      uint32
	dwMinorVersion      uint32
	dwBuildNumber       uint32
	dwPlatformId        uint32
	szCSDVersion        [128]uint16
	wServicePackMajor   uint16
	wServicePackMinor   uint16
	wSuiteMask          uint16
	wProductType        byte
	wReserved           byte
}

type _PAPPSTATE_CHANGE_ROUTINE func(quiesced bool, context unsafe.Pointer) uintptr

var (
	// https://github.com/MicrosoftDocs/sdk-api/blob/docs/sdk-api-src/content/appnotify/nf-appnotify-registerappstatechangenotification.md
	appnotify = windows.NewLazySystemDLL("API-MS-Win-Core-psm-appnotify-l1-1-0.dll")
	kernel32  = windows.NewLazySystemDLL("kernel32.dll")
	ntdll     = windows.NewLazySystemDLL("ntdll.dll")

	procRegisterAppStateChangeNotification = appnotify.NewProc("RegisterAppStateChangeNotification")

	procVerSetConditionMask = kernel32.NewProc("VerSetConditionMask")

	procRtlVerifyVersionInfo = ntdll.NewProc("RtlVerifyVersionInfo")
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

func _RtlVerifyVersionInfo(versionInfo *_OSVERSIONINFOEXW, typeMask uint32, conditionMask uint64) int32 {
	var r uintptr
	if is64bit {
		r, _, _ = procRtlVerifyVersionInfo.Call(uintptr(unsafe.Pointer(versionInfo)), uintptr(typeMask), uintptr(conditionMask))
	} else {
		switch runtime.GOARCH {
		case "386":
			r, _, _ = procRtlVerifyVersionInfo.Call(uintptr(unsafe.Pointer(versionInfo)), uintptr(typeMask), uintptr(conditionMask), uintptr(conditionMask>>32))
		case "arm":
			// Adjust the alignment for ARM.
			r, _, _ = procRtlVerifyVersionInfo.Call(uintptr(unsafe.Pointer(versionInfo)), uintptr(typeMask), 0, uintptr(conditionMask), uintptr(conditionMask>>32))
		default:
			panic(fmt.Sprintf("directx: GOARCH=%s is not supported", runtime.GOARCH))
		}
	}
	return int32(r)
}

func _VerSetConditionMask(conditionMask uint64, typeMask uint32, condition byte) uint64 {
	if is64bit {
		r, _, _ := procVerSetConditionMask.Call(uintptr(conditionMask), uintptr(typeMask), uintptr(condition))
		return uint64(r)
	} else {
		r1, r2, _ := procVerSetConditionMask.Call(uintptr(conditionMask), uintptr(conditionMask>>32), uintptr(typeMask), uintptr(condition))
		return uint64(r1) | (uint64(r2) << 32)
	}
}

func isWindows10OrGreaterWin32() bool {
	osvi := _OSVERSIONINFOEXW{
		dwMajorVersion: 10,
		dwMinorVersion: 0,
		dwBuildNumber:  0,
	}
	osvi.dwOSVersionInfoSize = uint32(unsafe.Sizeof(osvi))
	var mask uint32 = _VER_MAJORVERSION | _VER_MINORVERSION | _VER_BUILDNUMBER
	cond := _VerSetConditionMask(0, _VER_MAJORVERSION, _VER_GREATER_EQUAL)
	cond = _VerSetConditionMask(cond, _VER_MINORVERSION, _VER_GREATER_EQUAL)
	cond = _VerSetConditionMask(cond, _VER_BUILDNUMBER, _VER_GREATER_EQUAL)
	return _RtlVerifyVersionInfo(&osvi, mask, cond) == 0
}
