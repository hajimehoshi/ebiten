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
// +build microsoftgdk

package glfwwin

// Unfortunately, some functions like XSystemGetDeviceType is not implemented in a DLL,
// so LoadLibrary is not available.
//
// When creating a c-archive file with the build tag microsoftgdk, create a dummy DLL
// from dummy.c, and link it.

// #include <stdint.h>
//
// uint32_t XSystemGetDeviceType(void);
import "C"

const (
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

func isXbox() bool {
	t := C.XSystemGetDeviceType()
	return t != _XSystemDeviceType_Unknown && t != _XSystemDeviceType_Pc
}
