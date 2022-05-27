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

// This file does not include the headers of Microsoft GDK in order to compile easier.
// In order to compile this, create a dummy DLL with empty implementations like below,
// and link it.
//
//     #include <stdint.h>
//     __declspec(dllexport) __cdecl uint32_t XSystemGetDeviceType(void) {
//       return 0;
//     }
//
// Unfortunately, some functions like XSystemGetDeviceType is not implemented in a DLL,
// so LoadLibrary is not available.

// typedef enum {
//    Unknown              = 0x00,
//    Pc                   = 0x01,
//    XboxOne              = 0x02,
//    XboxOneS             = 0x03,
//    XboxOneX             = 0x04,
//    XboxOneXDevkit       = 0x05,
//    XboxScarlettLockhart = 0x06,
//    XboxScarlettAnaconda = 0x07,
//    XboxScarlettDevkit   = 0x08,
// } XSystemDeviceType;
//
// XSystemDeviceType XSystemGetDeviceType(void);
import "C"

func isXbox() bool {
	t := C.XSystemGetDeviceType()
	return t != C.Unknown && t != C.Pc
}
