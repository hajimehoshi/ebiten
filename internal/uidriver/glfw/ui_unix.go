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

// +build dragonfly freebsd linux netbsd openbsd solaris
// +build !js
// +build !android

package glfw

import (
	"math"

	"github.com/hajimehoshi/ebiten/internal/glfw"
)

// fromGLFWMonitorPixel must be called from the main thread.
func (u *UserInterface) fromGLFWMonitorPixel(x float64) float64 {
	return math.Ceil(x / u.deviceScaleFactor())
}

// fromGLFWPixel must be called from the main thread.
func (u *UserInterface) fromGLFWPixel(x float64) float64 {
	// deviceScaleFactor() is a scale by desktop environment (e.g., Cinnamon), while GetContentScale() is X's scale.
	// They are different things and then need to be treated different ways (#1350).
	s, _ := currentMonitor(u.window).GetContentScale()
	return x / float64(s)
}

// toGLFWPixel must be called from the main thread.
func (u *UserInterface) toGLFWPixel(x float64) float64 {
	s, _ := currentMonitor(u.window).GetContentScale()
	return x * float64(s)
}

// toFramebufferPixel must be called from the main thread.
func (u *UserInterface) toFramebufferPixel(x float64) float64 {
	s, _ := currentMonitor(u.window).GetContentScale()
	return math.Ceil(x * float64(s) / u.deviceScaleFactor())
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
