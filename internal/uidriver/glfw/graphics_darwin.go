// Copyright 2018 The Ebiten Authors
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

// +build !js
// +build !ebitengl

package glfw

// #cgo CFLAGS: -x objective-c
// #cgo LDFLAGS: -framework Foundation
//
// #import <Foundation/Foundation.h>
//
// static int getMacOSMajorVersion() {
//   NSOperatingSystemVersion version = [[NSProcessInfo processInfo] operatingSystemVersion];
//   return (int)version.majorVersion;
// }
//
// static int getMacOSMinorVersion() {
//   NSOperatingSystemVersion version = [[NSProcessInfo processInfo] operatingSystemVersion];
//   return (int)version.minorVersion;
// }
import "C"

import (
	"github.com/hajimehoshi/ebiten/internal/driver"
	"github.com/hajimehoshi/ebiten/internal/graphicsdriver/metal"
	"github.com/hajimehoshi/ebiten/internal/graphicsdriver/metal/mtl"
	"github.com/hajimehoshi/ebiten/internal/graphicsdriver/opengl"
)

var graphics driver.Graphics

func supportsMetal() bool {
	// On old mac devices like iMac 2011, Metal is not supported (#779).
	if _, err := mtl.CreateSystemDefaultDevice(); err != nil {
		return false
	}

	// On macOS 10.11 El Capitan, there is a rendering issue on Metal (#781).
	// Use the OpenGL in macOS 10.11 or older.
	if C.getMacOSMajorVersion() <= 10 && C.getMacOSMinorVersion() <= 11 {
		return false
	}
	return true
}

func init() {
	if supportsMetal() {
		graphics = metal.Get()
		return
	}
	graphics = opengl.Get()
}

func (*UserInterface) Graphics() driver.Graphics {
	return graphics
}
