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

package ui

import (
	"github.com/hajimehoshi/ebiten/internal/devicescale"
	"github.com/hajimehoshi/ebiten/internal/glfw"
)

func glfwScale() float64 {
	// This function must be called on the main thread.
	cm, ok := getCachedMonitor(currentUI.window.GetPos())
	if !ok {
		return devicescale.GetAt(currentUI.currentMonitor().GetPos())
	}
	return devicescale.GetAt(cm.x, cm.y)
}

func adjustWindowPosition(x, y int) (int, int) {
	return x, y
}

func (u *userInterface) currentMonitorFromPosition() *glfw.Monitor {
	// TODO: Return more appropriate display.
	if cm, ok := getCachedMonitor(u.window.GetPos()); ok {
		return cm.m
	}
	return glfw.GetPrimaryMonitor()
}

func (u *userInterface) nativeWindow() uintptr {
	// TODO: Implement this.
	return 0
}
