// Copyright 2016 Hajime Hoshi
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

//go:build (dragonfly || freebsd || linux || netbsd || openbsd || solaris) && !android
// +build dragonfly freebsd linux netbsd openbsd solaris
// +build !android

package glfw

import (
	"github.com/hajimehoshi/ebiten/v2/internal/driver"
	"github.com/hajimehoshi/ebiten/v2/internal/glfw"
)

// fromGLFWMonitorPixel must be called from the main thread.
func (u *UserInterface) fromGLFWMonitorPixel(x float64, videoModeScale float64) float64 {
	return x / (videoModeScale * u.deviceScaleFactor())
}

// fromGLFWPixel must be called from the main thread.
func (u *UserInterface) fromGLFWPixel(x float64) float64 {
	return x / u.deviceScaleFactor()
}

// toGLFWPixel must be called from the main thread.
func (u *UserInterface) toGLFWPixel(x float64) float64 {
	return x * u.deviceScaleFactor()
}

func (u *UserInterface) adjustWindowPosition(x, y int) (int, int) {
	return x, y
}

func currentMonitorByOS(_ *glfw.Window) *glfw.Monitor {
	// TODO: Implement this correctly. (#1119).
	return nil
}

func (u *UserInterface) nativeWindow() uintptr {
	// TODO: Implement this.
	return 0
}

func (u *UserInterface) isNativeFullscreen() bool {
	return false
}

func (u *UserInterface) setNativeCursor(shape driver.CursorShape) {
	// TODO: Use native API in the future (#1571)
	u.window.SetCursor(glfwSystemCursors[shape])
}
