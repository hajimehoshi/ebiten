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

// +build darwin freebsd linux windows
// +build !js
// +build !android
// +build !ios

package glfw

import (
	"image"

	"github.com/hajimehoshi/ebiten/internal/glfw"
)

type window struct {
	ui                *UserInterface
	setPositionCalled bool
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

		v := glfw.False
		if decorated {
			v = glfw.True
		}
		w.ui.window.SetAttrib(glfw.Decorated, v)

		// The title can be lost when the decoration is gone. Recover this.
		if v == glfw.True {
			w.ui.window.SetTitle(w.ui.title)
		}
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

		v := glfw.False
		if resizable {
			v = glfw.True
		}
		w.ui.window.SetAttrib(glfw.Resizable, v)
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

		v := glfw.False
		if floating {
			v = glfw.True
		}
		w.ui.window.SetAttrib(glfw.Floating, v)
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
		w.ui.window.Maximize()
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
		w.ui.window.Iconify()
		return nil
	})
}

func (w *window) Restore() {
	if !w.ui.isRunning() {
		// Do nothing
		return
	}
	_ = w.ui.t.Call(func() error {
		w.ui.window.Restore()
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
		if w.ui.isFullscreen() {
			wx, wy = w.ui.origPosX, w.ui.origPosY
		} else {
			wx, wy = w.ui.window.GetPos()
		}
		mx, my := currentMonitor(w.ui.window).GetPos()
		wx -= mx
		wy -= my
		xf := w.ui.fromGLFWPixel(float64(wx))
		yf := w.ui.fromGLFWPixel(float64(wy))
		x, y = int(xf), int(yf)
		return nil
	})
	return x, y
}

func (w *window) SetPosition(x, y int) {
	if !w.ui.isRunning() {
		w.ui.setInitWindowPosition(x, y)
		return
	}
	_ = w.ui.t.Call(func() error {
		defer func() {
			w.setPositionCalled = true
		}()
		mx, my := currentMonitor(w.ui.window).GetPos()
		xf := w.ui.toGLFWPixel(float64(x))
		yf := w.ui.toGLFWPixel(float64(y))
		x, y := w.ui.adjustWindowPosition(mx+int(xf), my+int(yf))
		if w.ui.isFullscreen() {
			w.ui.origPosX, w.ui.origPosY = x, y
		} else {
			w.ui.window.SetPos(x, y)
		}
		return nil
	})
}

func (w *window) Size() (int, int) {
	if !w.ui.isRunning() {
		return w.ui.getInitWindowSize()
	}
	ww := int(w.ui.fromGLFWPixel(float64(w.ui.windowWidth)))
	wh := int(w.ui.fromGLFWPixel(float64(w.ui.windowHeight)))
	return ww, wh
}

func (w *window) SetSize(width, height int) {
	if !w.ui.isRunning() {
		w.ui.setInitWindowSize(width, height)
		return
	}
	ww := int(w.ui.toGLFWPixel(float64(width)))
	wh := int(w.ui.toGLFWPixel(float64(height)))
	w.ui.setWindowSize(ww, wh, w.ui.isFullscreen())
}

func (w *window) SetIcon(iconImages []image.Image) {
	if !w.ui.isRunning() {
		w.ui.setInitIconImages(iconImages)
		return
	}
	_ = w.ui.t.Call(func() error {
		w.ui.window.SetIcon(iconImages)
		return nil
	})
}

func (w *window) SetTitle(title string) {
	if !w.ui.isRunning() {
		w.ui.setInitTitle(title)
		return
	}
	w.ui.title = title
	_ = w.ui.t.Call(func() error {
		w.ui.window.SetTitle(title)
		return nil
	})
}
