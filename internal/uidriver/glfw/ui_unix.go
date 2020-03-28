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
	"unsafe"

	"github.com/hajimehoshi/ebiten/internal/glfw"
)

func (u *UserInterface) glfwScale() float64 {
	return u.deviceScaleFactor()
}

func (u *UserInterface) adjustWindowPosition(x, y int) (int, int) {
	return x, y
}

func (u *UserInterface) currentMonitorFromPosition() *glfw.Monitor {
	// TODO: Implement this correctly. (#1119).
	if cm, ok := getCachedMonitor(u.window.GetPos()); ok {
		return cm.m
	}
	return glfw.GetPrimaryMonitor()
}

func (u *UserInterface) nativeWindow() unsafe.Pointer {
	// TODO: Implement this.
	return nil
}
