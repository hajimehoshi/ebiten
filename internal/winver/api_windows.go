// Copyright 2023 The Ebitengine Authors
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

package winver

import (
	"fmt"
	"runtime"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	_VER_BUILDNUMBER      = 0x00000004
	_VER_GREATER_EQUAL    = 3
	_VER_MAJORVERSION     = 0x00000002
	_VER_MINORVERSION     = 0x00000001
	_VER_SERVICEPACKMAJOR = 0x00000020
	_WIN32_WINNT_VISTA    = 0x0600
	_WIN32_WINNT_WIN7     = 0x0601
	_WIN32_WINNT_WIN8     = 0x0602
	_WIN32_WINNT_WINBLUE  = 0x0603
	_WIN32_WINNT_WINXP    = 0x0501
)

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

func _HIBYTE(wValue uint16) byte {
	return byte(wValue >> 8)
}

func _LOBYTE(wValue uint16) byte {
	return byte(wValue)
}

var (
	kernel32 = windows.NewLazySystemDLL("kernel32.dll")
	ntdll    = windows.NewLazySystemDLL("ntdll.dll")

	procVerSetConditionMask = kernel32.NewProc("VerSetConditionMask")

	procRtlVerifyVersionInfo = ntdll.NewProc("RtlVerifyVersionInfo")
)

func _RtlVerifyVersionInfo(versionInfo *_OSVERSIONINFOEXW, typeMask uint32, conditionMask uint64) int32 {
	var r uintptr
	if unsafe.Sizeof(uintptr(0)) == unsafe.Sizeof(uint64(0)) {
		r, _, _ = procRtlVerifyVersionInfo.Call(uintptr(unsafe.Pointer(versionInfo)), uintptr(typeMask), uintptr(conditionMask))
	} else {
		switch runtime.GOARCH {
		case "386":
			r, _, _ = procRtlVerifyVersionInfo.Call(uintptr(unsafe.Pointer(versionInfo)), uintptr(typeMask), uintptr(conditionMask), uintptr(conditionMask>>32))
		case "arm":
			// Adjust the alignment for ARM.
			r, _, _ = procRtlVerifyVersionInfo.Call(uintptr(unsafe.Pointer(versionInfo)), uintptr(typeMask), 0, uintptr(conditionMask), uintptr(conditionMask>>32))
		default:
			panic(fmt.Sprintf("winver: GOARCH=%s is not supported", runtime.GOARCH))
		}
	}
	return int32(r)
}

func _VerSetConditionMask(conditionMask uint64, typeMask uint32, condition byte) uint64 {
	if unsafe.Sizeof(uintptr(0)) == unsafe.Sizeof(uint64(0)) {
		r, _, _ := procVerSetConditionMask.Call(uintptr(conditionMask), uintptr(typeMask), uintptr(condition))
		return uint64(r)
	} else {
		r1, r2, _ := procVerSetConditionMask.Call(uintptr(conditionMask), uintptr(conditionMask>>32), uintptr(typeMask), uintptr(condition))
		return uint64(r1) | (uint64(r2) << 32)
	}
}
