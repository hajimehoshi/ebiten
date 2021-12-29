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

//go:build !android && !js && !ios
// +build !android,!js,!ios

package glfw

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2/internal/glfw"
)

type window struct {
	ui *UserInterface
}

func (w *window) IsDecorated() bool {
	if !w.ui.isRunning() {
		return w.ui.isInitWindowDecorated()
	}
	v := false
	_ = w.ui.t.Call(func() error {
		v = w.ui.window.GetAttrib(glfw.Decorated) == glfw.True
		return nil
	})
	return v
}

func (w *window) SetDecorated(decorated bool) {
	if !w.ui.isRunning() {
		w.ui.setInitWindowDecorated(decorated)
		return
	}

	_ = w.ui.t.Call(func() error {
		if w.ui.isNativeFullscreen() {
			return nil
		}

		w.ui.setWindowDecorated(decorated)
		return nil
	})
}

func (w *window) IsResizable() bool {
	if !w.ui.isRunning() {
		return w.ui.isInitWindowResizable()
	}
	v := false
	_ = w.ui.t.Call(func() error {
		v = w.ui.window.GetAttrib(glfw.Resizable) == glfw.True
		return nil
	})
	return v
}

func (w *window) SetResizable(resizable bool) {
	if !w.ui.isRunning() {
		w.ui.setInitWindowResizable(resizable)
		return
	}
	_ = w.ui.t.Call(func() error {
		if w.ui.isNativeFullscreen() {
			return nil
		}
		w.ui.setWindowResizable(resizable)
		return nil
	})
}

func (w *window) IsFloating() bool {
	if !w.ui.isRunning() {
		return w.ui.isInitWindowFloating()
	}
	var v bool
	_ = w.ui.t.Call(func() error {
		v = w.ui.window.GetAttrib(glfw.Floating) == glfw.True
		return nil
	})
	return v
}

func (w *window) SetFloating(floating bool) {
	if !w.ui.isRunning() {
		w.ui.setInitWindowFloating(floating)
		return
	}
	_ = w.ui.t.Call(func() error {
		if w.ui.isNativeFullscreen() {
			return nil
		}
		w.ui.setWindowFloating(floating)
		return nil
	})
}

func (w *window) IsMaximized() bool {
	if !w.ui.isRunning() {
		return w.ui.isInitWindowMaximized()
	}
	var v bool
	_ = w.ui.t.Call(func() error {
		v = w.ui.window.GetAttrib(glfw.Maximized) == glfw.True
		return nil
	})
	return v
}

func (w *window) Maximize() {
	if !w.IsResizable() {
		panic("glfw: a window to maximize must be resizable")
	}
	if !w.ui.isRunning() {
		w.ui.setInitWindowMaximized(true)
		return
	}
	_ = w.ui.t.Call(func() error {
		w.ui.maximizeWindow()
		return nil
	})
}

func (w *window) IsMinimized() bool {
	if !w.ui.isRunning() {
		return false
	}
	var v bool
	_ = w.ui.t.Call(func() error {
		v = w.ui.window.GetAttrib(glfw.Iconified) == glfw.True
		return nil
	})
	return v
}

func (w *window) Minimize() {
	if !w.ui.isRunning() {
		// Do nothing
		return
	}
	_ = w.ui.t.Call(func() error {
		w.ui.iconifyWindow()
		return nil
	})
}

func (w *window) Restore() {
	if !w.ui.isRunning() {
		// Do nothing
		return
	}
	_ = w.ui.t.Call(func() error {
		w.ui.restoreWindow()
		return nil
	})
}

func (w *window) Position() (int, int) {
	if !w.ui.isRunning() {
		panic("glfw: WindowPosition can't be called before the main loop starts")
	}
	x, y := 0, 0
	_ = w.ui.t.Call(func() error {
		var wx, wy int
		if w.ui.isFullscreen() && !w.ui.isNativeFullscreenAvailable() {
			wx, wy = w.ui.origPos()
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
		return nil
	})
	return x, y
}

func (w *window) SetPosition(x, y int) {
	if !w.ui.isRunning() {
		w.ui.setInitWindowPositionInDIP(x, y)
		return
	}
	_ = w.ui.t.Call(func() error {
		w.ui.setWindowPositionInDIP(x, y, w.ui.currentMonitor())
		return nil
	})
}

func (w *window) Size() (int, int) {
	if !w.ui.isRunning() {
		ww, wh := w.ui.getInitWindowSizeInDIP()
		return w.ui.adjustWindowSizeBasedOnSizeLimitsInDIP(ww, wh)
	}
	ww, wh := 0, 0
	_ = w.ui.t.Call(func() error {
		ww = w.ui.windowWidthInDIP
		wh = w.ui.windowHeightInDIP
		return nil
	})
	return ww, wh
}

func (w *window) SetSize(width, height int) {
	if !w.ui.isRunning() {
		w.ui.setInitWindowSizeInDIP(width, height)
		return
	}
	_ = w.ui.t.Call(func() error {
		// When a window is a native fullscreen, forcing to resize the window might leave unexpected image lags.
		// Forbid this.
		if w.ui.isNativeFullscreen() {
			return nil
		}

		w.ui.setWindowSizeInDIP(width, height, w.ui.isFullscreen())
		return nil
	})
}

func (w *window) SizeLimits() (minw, minh, maxw, maxh int) {
	return w.ui.getWindowSizeLimitsInDIP()
}

func (w *window) SetSizeLimits(minw, minh, maxw, maxh int) {
	if !w.ui.setWindowSizeLimitsInDIP(minw, minh, maxw, maxh) {
		return
	}
	if !w.ui.isRunning() {
		return
	}

	_ = w.ui.t.Call(func() error {
		w.ui.updateWindowSizeLimits()
		return nil
	})
}

func (w *window) SetIcon(iconImages []image.Image) {
	// The icons are actually set at (*UserInterface).loop.
	w.ui.setIconImages(iconImages)
}

func (w *window) SetTitle(title string) {
	if !w.ui.isRunning() {
		w.ui.m.Lock()
		w.ui.title = title
		w.ui.m.Unlock()
		return
	}
	w.ui.title = title
	_ = w.ui.t.Call(func() error {
		w.ui.setWindowTitle(title)
		return nil
	})
}

func (w *window) IsBeingClosed() bool {
	return w.ui.isWindowBeingClosed()
}

func (w *window) SetClosingHandled(handled bool) {
	w.ui.setWindowClosingHandled(handled)
}

func (w *window) IsClosingHandled() bool {
	return w.ui.isWindowClosingHandled()
}
