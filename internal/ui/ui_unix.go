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
	"github.com/go-gl/glfw/v3.2/glfw"

	"github.com/hajimehoshi/ebiten/internal/devicescale"
)

func glfwScale() float64 {
	// This function must be called on the main thread.
	return devicescale.GetAt(currentUI.currentMonitor().GetPos())
}

func adjustWindowPosition(x, y int) (int, int) {
	return x, y
}

func currentMonitor() *glfw.Monitor {
	// TODO: Return more appropriate display.
	w := glfw.GetCurrentContext()
	wx, wy := w.GetPos()
	for _, m := range glfw.GetMonitors() {
		mx, my := m.GetPos()
		v := m.GetVideoMode()
		if mx <= wx && wx < mx+v.Width && my <= wy && wy < my+v.Height {
			return m
		}
	}
	return nil
}
