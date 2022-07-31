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

//go:build !ios
// +build !ios

package metal

// #cgo CFLAGS: -x objective-c
// #cgo LDFLAGS: -framework Foundation
//
// #import <Foundation/Foundation.h>
//
// static int isOperatingSystemAtLeastVersion(NSOperatingSystemVersion v) {
//	return (int)[[NSProcessInfo processInfo] isOperatingSystemAtLeastVersion:v];
//}
import "C"

func supportsMetal() bool {
	// On macOS 10.11 El Capitan, there is a rendering issue on Metal (#781).
	// Use the OpenGL in macOS 10.11 or older.
	return C.isOperatingSystemAtLeastVersion(C.NSOperatingSystemVersion{majorVersion: 10, minorVersion: 11, patchVersion: 0}) != 0
}
