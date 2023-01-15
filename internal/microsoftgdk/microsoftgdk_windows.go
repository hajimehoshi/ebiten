// Copyright 2022 The Ebitengine Authors
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

//go:build microsoftgdk

package microsoftgdk

// Unfortunately, some functions like XSystemGetDeviceType is not implemented in a DLL,
// so LoadLibrary is not available.

// #include <stdint.h>
//
// uint32_t XGameRuntimeInitialize(void);
// uint32_t XSystemGetDeviceType(void);
import "C"

import (
	"fmt"

	"golang.org/x/sys/windows"
)

var (
	kernel32 = windows.NewLazyDLL("kernel32")

	procGetACP = kernel32.NewProc("GetACP")
)

const (
	_CP_UTF8 = 65001

	_XSystemDeviceType_Unknown              = 0x00
	_XSystemDeviceType_Pc                   = 0x01
	_XSystemDeviceType_XboxOne              = 0x02
	_XSystemDeviceType_XboxOneS             = 0x03
	_XSystemDeviceType_XboxOneX             = 0x04
	_XSystemDeviceType_XboxOneXDevkit       = 0x05
	_XSystemDeviceType_XboxScarlettLockhart = 0x06
	_XSystemDeviceType_XboxScarlettAnaconda = 0x07
	_XSystemDeviceType_XboxScarlettDevkit   = 0x08
)

func _GetACP() uint32 {
	r, _, _ := procGetACP.Call()
	return uint32(r)
}

func IsXbox() bool {
	t := C.XSystemGetDeviceType()
	return t != _XSystemDeviceType_Unknown && t != _XSystemDeviceType_Pc
}

func MonitorResolution() (int, int) {
	switch C.XSystemGetDeviceType() {
	case _XSystemDeviceType_Unknown, _XSystemDeviceType_Pc:
		return 1920, 1080
	case _XSystemDeviceType_XboxOne, _XSystemDeviceType_XboxOneS:
		return 1920, 1080
	case _XSystemDeviceType_XboxScarlettLockhart:
		// Series S
		return 2560, 1440
	case _XSystemDeviceType_XboxOneX, _XSystemDeviceType_XboxOneXDevkit, _XSystemDeviceType_XboxScarlettAnaconda, _XSystemDeviceType_XboxScarlettDevkit:
		// Series X
		fallthrough
	default:
		// Forward compatibility.
		return 3840, 2160
	}
}

func D3D12DLLName() string {
	switch C.XSystemGetDeviceType() {
	case _XSystemDeviceType_Unknown, _XSystemDeviceType_Pc:
		return ""
	case _XSystemDeviceType_XboxOne, _XSystemDeviceType_XboxOneS, _XSystemDeviceType_XboxOneX, _XSystemDeviceType_XboxOneXDevkit:
		return "d3d12_x.dll"
	case _XSystemDeviceType_XboxScarlettLockhart, _XSystemDeviceType_XboxScarlettAnaconda, _XSystemDeviceType_XboxScarlettDevkit:
		fallthrough
	default:
		// Forward compatibility.
		return "d3d12_xs.dll"
	}
}

func D3D12SDKVersion() uint32 {
	switch C.XSystemGetDeviceType() {
	case _XSystemDeviceType_Unknown, _XSystemDeviceType_Pc:
		return 0
	case _XSystemDeviceType_XboxOne, _XSystemDeviceType_XboxOneS, _XSystemDeviceType_XboxOneX, _XSystemDeviceType_XboxOneXDevkit:
		return (1 << 16) | 10
	case _XSystemDeviceType_XboxScarlettLockhart, _XSystemDeviceType_XboxScarlettAnaconda, _XSystemDeviceType_XboxScarlettDevkit:
		fallthrough
	default:
		// Forward compatibility.
		return (2 << 16) | 4
	}
}

func init() {
	if r := C.XGameRuntimeInitialize(); uint32(r) != uint32(windows.S_OK) {
		panic(fmt.Sprintf("microsoftgdk: XSystemGetDeviceType failed: HRESULT(%d)", uint32(r)))
	}
	if got, want := _GetACP(), uint32(_CP_UTF8); got != want {
		panic(fmt.Sprintf("microsoftgdk: GetACP(): got %d, want %d", got, want))
	}
}
