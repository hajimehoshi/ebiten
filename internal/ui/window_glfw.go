// Copyright 2019 The Ebiten Authors
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

//go:build !android && !ios && !js && !nintendosdk && !playstation5

package ui

import (
	"runtime"

	"github.com/hajimehoshi/ebiten/v2/internal/colormode"
	"github.com/hajimehoshi/ebiten/v2/internal/glfw"
)

type glfwWindow struct {
	ui *glfwBackend
}

var _ backendWindow = (*glfwWindow)(nil)

func (w *glfwWindow) IsDecorated() bool {
	var v bool
	w.ui.mainThread.Call(func() {
		if w.ui.isTerminated() {
			return
		}
		a, err := w.ui.window.GetAttrib(glfw.Decorated)
		if err != nil {
			w.ui.setError(err)
			return
		}
		v = a == glfw.True
	})
	return v
}

func (w *glfwWindow) SetDecorated(decorated bool) {
	w.ui.mainThread.Call(func() {
		if w.ui.isTerminated() {
			return
		}
		if err := w.ui.setWindowDecorated(decorated); err != nil {
			w.ui.setError(err)
			return
		}
	})
}

func (w *glfwWindow) SetResizingMode(mode WindowResizingMode) {
	w.ui.mainThread.Call(func() {
		if w.ui.isTerminated() {
			return
		}
		if err := w.ui.setWindowResizingMode(mode); err != nil {
			w.ui.setError(err)
			return
		}
	})
}

func (w *glfwWindow) IsFloating() bool {
	var v bool
	w.ui.mainThread.Call(func() {
		if w.ui.isTerminated() {
			return
		}
		a, err := w.ui.window.GetAttrib(glfw.Floating)
		if err != nil {
			w.ui.setError(err)
			return
		}
		v = a == glfw.True
	})
	return v
}

func (w *glfwWindow) SetFloating(floating bool) {
	w.ui.mainThread.Call(func() {
		if w.ui.isTerminated() {
			return
		}
		if err := w.ui.setWindowFloating(floating); err != nil {
			w.ui.setError(err)
			return
		}
	})
}

func (w *glfwWindow) IsMaximized() bool {
	var v bool
	w.ui.mainThread.Call(func() {
		if w.ui.isTerminated() {
			return
		}
		m, err := w.ui.isWindowMaximized()
		if err != nil {
			w.ui.setError(err)
			return
		}
		v = m
	})
	return v
}

func (w *glfwWindow) Maximize() {
	w.ui.mainThread.Call(func() {
		if w.ui.isTerminated() {
			return
		}
		if err := w.ui.maximizeWindow(); err != nil {
			w.ui.setError(err)
			return
		}
	})
}

func (w *glfwWindow) IsMinimized() bool {
	var v bool
	w.ui.mainThread.Call(func() {
		if w.ui.isTerminated() {
			return
		}
		a, err := w.ui.window.GetAttrib(glfw.Iconified)
		if err != nil {
			w.ui.setError(err)
			return
		}
		v = a == glfw.True
	})
	return v
}

func (w *glfwWindow) Minimize() {
	w.ui.mainThread.Call(func() {
		if w.ui.isTerminated() {
			return
		}
		if err := w.ui.iconifyWindow(); err != nil {
			w.ui.setError(err)
			return
		}
	})
}

func (w *glfwWindow) Restore() {
	w.ui.mainThread.Call(func() {
		if w.ui.isTerminated() {
			return
		}
		if err := w.ui.restoreWindow(); err != nil {
			w.ui.setError(err)
			return
		}
	})
}

func (w *glfwWindow) SetMonitor(monitor *Monitor) {
	w.ui.mainThread.Call(func() {
		if w.ui.isTerminated() {
			return
		}
		if err := w.ui.setWindowMonitor(monitor); err != nil {
			w.ui.setError(err)
			return
		}
	})
}

func (w *glfwWindow) Position() (int, int) {
	var x, y int
	w.ui.mainThread.Call(func() {
		if w.ui.isTerminated() {
			return
		}
		f, err := w.ui.isFullscreen()
		if err != nil {
			w.ui.setError(err)
			return
		}

		var wx, wy int
		if f {
			wx, wy = w.ui.origWindowPos()
		} else {
			x, y, err := w.ui.window.GetPos()
			if err != nil {
				w.ui.setError(err)
				return
			}
			wx, wy = x, y
		}
		m, err := w.ui.currentMonitor()
		if err != nil {
			w.ui.setError(err)
			return
		}
		wx -= m.boundsInGLFWPixels.Min.X
		wy -= m.boundsInGLFWPixels.Min.Y
		s := m.DeviceScaleFactor()
		xf := dipFromGLFWPixel(float64(wx), s)
		yf := dipFromGLFWPixel(float64(wy), s)
		x, y = int(xf), int(yf)
	})
	return x, y
}

func (w *glfwWindow) SetPosition(x, y int) {
	w.ui.mainThread.Call(func() {
		if w.ui.isTerminated() {
			return
		}
		m, err := w.ui.currentMonitor()
		if err != nil {
			w.ui.setError(err)
			return
		}
		if err := w.ui.setWindowPositionInDIP(x, y, m); err != nil {
			w.ui.setError(err)
			return
		}
	})
}

func (w *glfwWindow) Size() (int, int) {
	var ww, wh int
	w.ui.mainThread.Call(func() {
		if w.ui.isTerminated() {
			return
		}
		// Unlike origWindowPos, origWindow{Width,Height}InDPI are always updated via the callback.
		ww = w.ui.origWindowWidthInDIP
		wh = w.ui.origWindowHeightInDIP
	})
	return ww, wh
}

func (w *glfwWindow) SetSize(width, height int) {
	w.ui.mainThread.Call(func() {
		if w.ui.isTerminated() {
			return
		}
		m, err := w.ui.isWindowMaximized()
		if err != nil {
			w.ui.setError(err)
			return
		}
		if m && runtime.GOOS != "darwin" {
			return
		}
		if err := w.ui.setWindowSizeInDIP(width, height, true); err != nil {
			w.ui.setError(err)
			return
		}
	})
}

func (w *glfwWindow) SetSizeLimits(minw, minh, maxw, maxh int) {
	w.ui.mainThread.Call(func() {
		if w.ui.isTerminated() {
			return
		}
		if err := w.ui.updateWindowSizeLimits(); err != nil {
			w.ui.setError(err)
			return
		}
	})
}

func (w *glfwWindow) SetTitle(title string) {
	w.ui.mainThread.Call(func() {
		if w.ui.isTerminated() {
			return
		}
		if err := w.ui.setWindowTitle(title); err != nil {
			w.ui.setError(err)
			return
		}
	})
}

func (w *glfwWindow) SetColorMode(mode colormode.ColorMode) {
	var err error
	w.ui.mainThread.Call(func() {
		if w.ui.isTerminated() {
			return
		}
		err = w.ui.setWindowColorModeImpl(mode)
	})
	if err != nil {
		w.ui.setError(err)
	}
}

func (w *glfwWindow) SetClosingHandled(handled bool) {
	w.ui.mainThread.Call(func() {
		if w.ui.isTerminated() {
			return
		}
		if err := w.ui.setDocumentEdited(handled); err != nil {
			w.ui.setError(err)
			return
		}
	})
}

func (w *glfwWindow) SetMousePassthrough(enabled bool) {
	w.ui.mainThread.Call(func() {
		if w.ui.isTerminated() {
			return
		}
		if err := w.ui.setWindowMousePassthrough(enabled); err != nil {
			w.ui.setError(err)
			return
		}
	})
}

func (w *glfwWindow) IsMousePassthrough() bool {
	var v bool
	w.ui.mainThread.Call(func() {
		if w.ui.isTerminated() {
			return
		}
		a, err := w.ui.window.GetAttrib(glfw.MousePassthrough)
		if err != nil {
			w.ui.setError(err)
			return
		}
		v = a == glfw.True
	})
	return v
}

func (w *glfwWindow) RequestAttention() {
	w.ui.mainThread.Call(func() {
		if w.ui.isTerminated() {
			return
		}
		if err := w.ui.window.RequestAttention(); err != nil {
			w.ui.setError(err)
			return
		}
	})
}
