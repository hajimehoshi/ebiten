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

//go:build !android && !ios && !js && !ebitencbackend
// +build !android,!ios,!js,!ebitencbackend

package ui

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2/internal/glfw"
)

type Window struct {
	ui *userInterfaceImpl
}

func (w *Window) IsDecorated() bool {
	if !w.ui.isRunning() {
		return w.ui.isInitWindowDecorated()
	}
	v := false
	w.ui.t.Call(func() {
		v = w.ui.window.GetAttrib(glfw.Decorated) == glfw.True
	})
	return v
}

func (w *Window) SetDecorated(decorated bool) {
	if !w.ui.isRunning() {
		w.ui.setInitWindowDecorated(decorated)
		return
	}

	w.ui.t.Call(func() {
		if w.ui.isNativeFullscreen() {
			return
		}

		w.ui.setWindowDecorated(decorated)
	})
}

func (w *Window) ResizingMode() WindowResizingMode {
	if !w.ui.isRunning() {
		w.ui.m.Lock()
		mode := w.ui.windowResizingMode
		w.ui.m.Unlock()
		return mode
	}
	var mode WindowResizingMode
	w.ui.t.Call(func() {
		mode = w.ui.windowResizingMode
	})
	return mode
}

func (w *Window) SetResizingMode(mode WindowResizingMode) {
	if !w.ui.isRunning() {
		w.ui.m.Lock()
		w.ui.windowResizingMode = mode
		w.ui.m.Unlock()
		return
	}
	w.ui.t.Call(func() {
		if w.ui.isNativeFullscreen() {
			return
		}
		w.ui.setWindowResizingMode(mode)
	})
}

func (w *Window) IsFloating() bool {
	if !w.ui.isRunning() {
		return w.ui.isInitWindowFloating()
	}
	var v bool
	w.ui.t.Call(func() {
		v = w.ui.window.GetAttrib(glfw.Floating) == glfw.True
	})
	return v
}

func (w *Window) SetFloating(floating bool) {
	if !w.ui.isRunning() {
		w.ui.setInitWindowFloating(floating)
		return
	}
	w.ui.t.Call(func() {
		if w.ui.isNativeFullscreen() {
			return
		}
		w.ui.setWindowFloating(floating)
	})
}

func (w *Window) IsMaximized() bool {
	if !w.ui.isRunning() {
		return w.ui.isInitWindowMaximized()
	}
	if w.ResizingMode() != WindowResizingModeEnabled {
		return false
	}
	var v bool
	w.ui.t.Call(func() {
		v = w.ui.window.GetAttrib(glfw.Maximized) == glfw.True
	})
	return v
}

func (w *Window) Maximize() {
	// Do not allow maximizing the window when the window is not resizable.
	// On Windows, it is possible to restore the window from being maximized by mouse-dragging,
	// and this can be an unexpected behavior.
	if w.ResizingMode() != WindowResizingModeEnabled {
		panic("ui: a window to maximize must be resizable")
	}
	if !w.ui.isRunning() {
		w.ui.setInitWindowMaximized(true)
		return
	}
	w.ui.t.Call(w.ui.maximizeWindow)
}

func (w *Window) IsMinimized() bool {
	if !w.ui.isRunning() {
		return false
	}
	var v bool
	w.ui.t.Call(func() {
		v = w.ui.window.GetAttrib(glfw.Iconified) == glfw.True
	})
	return v
}

func (w *Window) Minimize() {
	if !w.ui.isRunning() {
		// Do nothing
		return
	}
	w.ui.t.Call(w.ui.iconifyWindow)
}

func (w *Window) Restore() {
	if !w.ui.isRunning() {
		// Do nothing
		return
	}
	w.ui.t.Call(w.ui.restoreWindow)
}

func (w *Window) Position() (int, int) {
	if !w.ui.isRunning() {
		panic("ui: WindowPosition can't be called before the main loop starts")
	}
	x, y := 0, 0
	w.ui.t.Call(func() {
		var wx, wy int
		if w.ui.isFullscreen() {
			wx, wy = w.ui.origWindowPos()
		} else {
			wx, wy = w.ui.window.GetPos()
		}
		m := w.ui.currentMonitor()
		mx, my := m.GetPos()
		wx -= mx
		wy -= my
		xf := w.ui.dipFromGLFWPixel(float64(wx), m)
		yf := w.ui.dipFromGLFWPixel(float64(wy), m)
		x, y = int(xf), int(yf)
	})
	return x, y
}

func (w *Window) SetPosition(x, y int) {
	if !w.ui.isRunning() {
		w.ui.setInitWindowPositionInDIP(x, y)
		return
	}
	w.ui.t.Call(func() {
		w.ui.setWindowPositionInDIP(x, y, w.ui.currentMonitor())
	})
}

func (w *Window) Size() (int, int) {
	if !w.ui.isRunning() {
		ww, wh := w.ui.getInitWindowSizeInDIP()
		return w.ui.adjustWindowSizeBasedOnSizeLimitsInDIP(ww, wh)
	}
	ww, wh := 0, 0
	w.ui.t.Call(func() {
		// Unlike origWindowPos, windowWidth/HeightInDPI is always updated via the callback.
		ww = w.ui.windowWidthInDIP
		wh = w.ui.windowHeightInDIP
	})
	return ww, wh
}

func (w *Window) SetSize(width, height int) {
	if !w.ui.isRunning() {
		w.ui.setInitWindowSizeInDIP(width, height)
		return
	}
	w.ui.t.Call(func() {
		// When a window is a native fullscreen, forcing to resize the window might leave unexpected image lags.
		// Forbid this.
		// TODO: Remove this condition (#1590).
		if w.ui.isNativeFullscreen() {
			return
		}

		w.ui.setWindowSizeInDIP(width, height, w.ui.isFullscreen())
	})
}

func (w *Window) SizeLimits() (minw, minh, maxw, maxh int) {
	return w.ui.getWindowSizeLimitsInDIP()
}

func (w *Window) SetSizeLimits(minw, minh, maxw, maxh int) {
	if !w.ui.setWindowSizeLimitsInDIP(minw, minh, maxw, maxh) {
		return
	}
	if !w.ui.isRunning() {
		return
	}

	w.ui.t.Call(w.ui.updateWindowSizeLimits)
}

func (w *Window) SetIcon(iconImages []image.Image) {
	// The icons are actually set at (*UserInterface).loop.
	w.ui.setIconImages(iconImages)
}

func (w *Window) SetTitle(title string) {
	if !w.ui.isRunning() {
		w.ui.m.Lock()
		w.ui.title = title
		w.ui.m.Unlock()
		return
	}
	w.ui.title = title
	w.ui.t.Call(func() {
		w.ui.setWindowTitle(title)
	})
}

func (w *Window) IsBeingClosed() bool {
	return w.ui.isWindowBeingClosed()
}

func (w *Window) SetClosingHandled(handled bool) {
	w.ui.setWindowClosingHandled(handled)
}

func (w *Window) IsClosingHandled() bool {
	return w.ui.isWindowClosingHandled()
}
