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

//go:build !android && !ios && !js && !nintendosdk

package ui

import (
	"image"
	"runtime"

	"github.com/hajimehoshi/ebiten/v2/internal/glfw"
)

type glfwWindow struct {
	ui *userInterfaceImpl
}

func (w *glfwWindow) IsDecorated() bool {
	if w.ui.isTerminated() {
		return false
	}
	if !w.ui.isRunning() {
		return w.ui.isInitWindowDecorated()
	}
	var v bool
	w.ui.mainThread.Call(func() {
		if w.ui.isTerminated() {
			return
		}
		v = w.ui.window.GetAttrib(glfw.Decorated) == glfw.True
	})
	return v
}

func (w *glfwWindow) SetDecorated(decorated bool) {
	if w.ui.isTerminated() {
		return
	}
	if !w.ui.isRunning() {
		w.ui.setInitWindowDecorated(decorated)
		return
	}

	w.ui.mainThread.Call(func() {
		if w.ui.isTerminated() {
			return
		}
		w.ui.setWindowDecorated(decorated)
	})
}

func (w *glfwWindow) ResizingMode() WindowResizingMode {
	if w.ui.isTerminated() {
		return 0
	}
	if !w.ui.isRunning() {
		w.ui.m.Lock()
		mode := w.ui.windowResizingMode
		w.ui.m.Unlock()
		return mode
	}
	var mode WindowResizingMode
	w.ui.mainThread.Call(func() {
		if w.ui.isTerminated() {
			return
		}
		mode = w.ui.windowResizingMode
	})
	return mode
}

func (w *glfwWindow) SetResizingMode(mode WindowResizingMode) {
	if w.ui.isTerminated() {
		return
	}
	if !w.ui.isRunning() {
		w.ui.m.Lock()
		w.ui.windowResizingMode = mode
		w.ui.m.Unlock()
		return
	}
	w.ui.mainThread.Call(func() {
		if w.ui.isTerminated() {
			return
		}
		w.ui.setWindowResizingMode(mode)
	})
}

func (w *glfwWindow) IsFloating() bool {
	if w.ui.isTerminated() {
		return false
	}
	if !w.ui.isRunning() {
		return w.ui.isInitWindowFloating()
	}
	var v bool
	w.ui.mainThread.Call(func() {
		if w.ui.isTerminated() {
			return
		}
		v = w.ui.window.GetAttrib(glfw.Floating) == glfw.True
	})
	return v
}

func (w *glfwWindow) SetFloating(floating bool) {
	if w.ui.isTerminated() {
		return
	}
	if !w.ui.isRunning() {
		w.ui.setInitWindowFloating(floating)
		return
	}
	w.ui.mainThread.Call(func() {
		if w.ui.isTerminated() {
			return
		}
		w.ui.setWindowFloating(floating)
	})
}

func (w *glfwWindow) IsMaximized() bool {
	if w.ui.isTerminated() {
		return false
	}
	if !w.ui.isRunning() {
		return w.ui.isInitWindowMaximized()
	}
	if w.ResizingMode() != WindowResizingModeEnabled {
		return false
	}
	var v bool
	w.ui.mainThread.Call(func() {
		if w.ui.isTerminated() {
			return
		}
		v = w.ui.isWindowMaximized()
	})
	return v
}

func (w *glfwWindow) Maximize() {
	if w.ui.isTerminated() {
		return
	}

	// Do not allow maximizing the window when the window is not resizable.
	// On Windows, it is possible to restore the window from being maximized by mouse-dragging,
	// and this can be an unexpected behavior (#1990).
	if w.ResizingMode() != WindowResizingModeEnabled {
		return
	}

	if !w.ui.isWindowMaximizable() {
		return
	}

	if !w.ui.isRunning() {
		w.ui.setInitWindowMaximized(true)
		return
	}
	w.ui.mainThread.Call(func() {
		if w.ui.isTerminated() {
			return
		}
		w.ui.maximizeWindow()
	})
}

func (w *glfwWindow) IsMinimized() bool {
	if !w.ui.isRunning() {
		return false
	}
	var v bool
	w.ui.mainThread.Call(func() {
		if w.ui.isTerminated() {
			return
		}
		v = w.ui.window.GetAttrib(glfw.Iconified) == glfw.True
	})
	return v
}

func (w *glfwWindow) Minimize() {
	if !w.ui.isRunning() {
		// Do nothing
		return
	}
	w.ui.mainThread.Call(func() {
		if w.ui.isTerminated() {
			return
		}
		w.ui.iconifyWindow()
	})
}

func (w *glfwWindow) Restore() {
	if w.ui.isTerminated() {
		return
	}
	if !w.ui.isWindowMaximizable() {
		return
	}
	if !w.ui.isRunning() {
		// Do nothing
		return
	}
	w.ui.mainThread.Call(func() {
		if w.ui.isTerminated() {
			return
		}
		w.ui.restoreWindow()
	})
}

func (w *glfwWindow) SetMonitor(monitor *Monitor) {
	if monitor == nil {
		panic("ui: monitor cannot be nil at SetMonitor")
	}
	if w.ui.isTerminated() {
		return
	}
	if !w.ui.isRunning() {
		w.ui.setInitMonitor(monitor)
		return
	}
	w.ui.mainThread.Call(func() {
		if w.ui.isTerminated() {
			return
		}
		w.ui.setWindowMonitor(monitor)
	})
}

func (w *glfwWindow) Position() (int, int) {
	if w.ui.isTerminated() {
		return 0, 0
	}
	if !w.ui.isRunning() {
		panic("ui: WindowPosition can't be called before the main loop starts")
	}
	var x, y int
	w.ui.mainThread.Call(func() {
		if w.ui.isTerminated() {
			return
		}
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

func (w *glfwWindow) SetPosition(x, y int) {
	if w.ui.isTerminated() {
		return
	}
	if !w.ui.isRunning() {
		w.ui.setInitWindowPositionInDIP(x, y)
		return
	}
	w.ui.mainThread.Call(func() {
		if w.ui.isTerminated() {
			return
		}
		w.ui.setWindowPositionInDIP(x, y, w.ui.currentMonitor())
	})
}

func (w *glfwWindow) Size() (int, int) {
	if w.ui.isTerminated() {
		return 0, 0
	}
	if !w.ui.isRunning() {
		ww, wh := w.ui.getInitWindowSizeInDIP()
		return w.ui.adjustWindowSizeBasedOnSizeLimitsInDIP(ww, wh)
	}
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
	if w.ui.isTerminated() {
		return
	}
	if !w.ui.isRunning() {
		// If the window is initially maximized, the set size is ignored anyway.
		w.ui.setInitWindowSizeInDIP(width, height)
		return
	}
	w.ui.mainThread.Call(func() {
		if w.ui.isTerminated() {
			return
		}
		if w.ui.isWindowMaximized() && runtime.GOOS != "darwin" {
			return
		}
		w.ui.setWindowSizeInDIP(width, height, true)
	})
}

func (w *glfwWindow) SizeLimits() (minw, minh, maxw, maxh int) {
	return w.ui.getWindowSizeLimitsInDIP()
}

func (w *glfwWindow) SetSizeLimits(minw, minh, maxw, maxh int) {
	if w.ui.isTerminated() {
		return
	}
	if !w.ui.setWindowSizeLimitsInDIP(minw, minh, maxw, maxh) {
		return
	}
	if !w.ui.isRunning() {
		return
	}

	w.ui.mainThread.Call(func() {
		if w.ui.isTerminated() {
			return
		}
		w.ui.updateWindowSizeLimits()
	})
}

func (w *glfwWindow) SetIcon(iconImages []image.Image) {
	if w.ui.isTerminated() {
		return
	}
	// The icons are actually set at (*UserInterface).loop.
	w.ui.setIconImages(iconImages)
}

func (w *glfwWindow) SetTitle(title string) {
	if w.ui.isTerminated() {
		return
	}
	if !w.ui.isRunning() {
		w.ui.m.Lock()
		w.ui.title = title
		w.ui.m.Unlock()
		return
	}
	w.ui.title = title
	w.ui.mainThread.Call(func() {
		if w.ui.isTerminated() {
			return
		}
		w.ui.setWindowTitle(title)
	})
}

func (w *glfwWindow) SetClosingHandled(handled bool) {
	w.ui.setWindowClosingHandled(handled)
}

func (w *glfwWindow) IsClosingHandled() bool {
	return w.ui.isWindowClosingHandled()
}

func (w *glfwWindow) SetMousePassthrough(enabled bool) {
	if w.ui.isTerminated() {
		return
	}
	if !w.ui.isRunning() {
		w.ui.setInitWindowMousePassthrough(enabled)
		return
	}
	w.ui.mainThread.Call(func() {
		if w.ui.isTerminated() {
			return
		}
		w.ui.setWindowMousePassthrough(enabled)
	})
}

func (w *glfwWindow) IsMousePassthrough() bool {
	if w.ui.isTerminated() {
		return false
	}
	if !w.ui.isRunning() {
		return w.ui.isInitWindowMousePassthrough()
	}
	var v bool
	w.ui.mainThread.Call(func() {
		if w.ui.isTerminated() {
			return
		}
		v = w.ui.window.GetAttrib(glfw.MousePassthrough) == glfw.True
	})
	return v
}
