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
	"image"
	"runtime"

	"github.com/hajimehoshi/ebiten/v2/internal/glfw"
	"github.com/hajimehoshi/ebiten/v2/internal/thread"
)

type glfwWindow struct {
	ui *glfwBackend
}

var _ backendWindow = (*glfwWindow)(nil)

func (w *glfwWindow) IsDecorated() bool {
	if p := w.ui.desktopWindow.windowDecorated.pending(); p != nil {
		return *p
	}
	return thread.CallWithArgAndResult(w.ui.mainThread, func(w *glfwWindow) bool {
		if w.ui.isTerminated() {
			return false
		}
		a, err := w.ui.window.GetAttrib(glfw.Decorated)
		if err != nil {
			w.ui.setError(err)
			return false
		}
		return a == glfw.True
	}, w)
}

func (w *glfwWindow) IsVisible() bool {
	if p := w.ui.desktopWindow.windowVisible.pending(); p != nil {
		return *p
	}
	return thread.CallWithArgAndResult(w.ui.mainThread, func(w *glfwWindow) bool {
		if w.ui.isTerminated() {
			return false
		}
		visible, err := w.ui.isWindowVisible()
		if err != nil {
			w.ui.setError(err)
			return false
		}
		return visible
	}, w)
}

func (w *glfwWindow) IsFloating() bool {
	if p := w.ui.desktopWindow.windowFloating.pending(); p != nil {
		return *p
	}
	return thread.CallWithArgAndResult(w.ui.mainThread, func(w *glfwWindow) bool {
		if w.ui.isTerminated() {
			return false
		}
		a, err := w.ui.window.GetAttrib(glfw.Floating)
		if err != nil {
			w.ui.setError(err)
			return false
		}
		return a == glfw.True
	}, w)
}

func (w *glfwWindow) IsMaximized() bool {
	return thread.CallWithArgAndResult(w.ui.mainThread, func(w *glfwWindow) bool {
		if w.ui.isTerminated() {
			return false
		}
		m, err := w.ui.isWindowMaximized()
		if err != nil {
			w.ui.setError(err)
			return false
		}
		return m
	}, w)
}

func (w *glfwWindow) Maximize() {
	thread.CallWithArg(w.ui.mainThread, func(w *glfwWindow) {
		if w.ui.isTerminated() {
			return
		}
		if err := w.ui.applyWindowSettings(); err != nil {
			w.ui.setError(err)
			return
		}
		if err := w.ui.maximizeWindow(); err != nil {
			w.ui.setError(err)
			return
		}
	}, w)
}

func (w *glfwWindow) IsMinimized() bool {
	return thread.CallWithArgAndResult(w.ui.mainThread, func(w *glfwWindow) bool {
		if w.ui.isTerminated() {
			return false
		}
		iconified, err := w.ui.isWindowIconified()
		if err != nil {
			w.ui.setError(err)
			return false
		}
		return iconified
	}, w)
}

func (w *glfwWindow) Minimize() {
	thread.CallWithArg(w.ui.mainThread, func(w *glfwWindow) {
		if w.ui.isTerminated() {
			return
		}
		if err := w.ui.applyWindowSettings(); err != nil {
			w.ui.setError(err)
			return
		}
		if err := w.ui.iconifyWindow(); err != nil {
			w.ui.setError(err)
			return
		}
	}, w)
}

func (w *glfwWindow) Restore() {
	thread.CallWithArg(w.ui.mainThread, func(w *glfwWindow) {
		if w.ui.isTerminated() {
			return
		}
		if err := w.ui.applyWindowSettings(); err != nil {
			w.ui.setError(err)
			return
		}
		if err := w.ui.restoreWindow(); err != nil {
			w.ui.setError(err)
			return
		}
	}, w)
}

func (w *glfwWindow) Position() (int, int) {
	if p := w.ui.desktopWindow.windowPositionInDIP.pending(); p != nil {
		return p.X, p.Y
	}
	p := thread.CallWithArgAndResult(w.ui.mainThread, func(w *glfwWindow) image.Point {
		if w.ui.isTerminated() {
			return image.Point{}
		}
		return image.Pt(w.ui.windowXInDIP, w.ui.windowYInDIP)
	}, w)
	return p.X, p.Y
}

func (w *glfwWindow) Size() (int, int) {
	if p := w.ui.desktopWindow.windowSizeInDIP.pending(); p != nil {
		return w.ui.desktopWindow.adjustWindowSizeBasedOnSizeLimitsInDIP(p.X, p.Y)
	}
	p := thread.CallWithArgAndResult(w.ui.mainThread, func(w *glfwWindow) image.Point {
		if w.ui.isTerminated() {
			return image.Point{}
		}
		return image.Pt(w.ui.windowWidthInDIP, w.ui.windowHeightInDIP)
	}, w)
	return p.X, p.Y
}

func (w *glfwWindow) IsMousePassthrough() bool {
	if p := w.ui.desktopWindow.windowMousePassthrough.pending(); p != nil {
		return *p
	}
	return thread.CallWithArgAndResult(w.ui.mainThread, func(w *glfwWindow) bool {
		if w.ui.isTerminated() {
			return false
		}
		a, err := w.ui.window.GetAttrib(glfw.MousePassthrough)
		if err != nil {
			w.ui.setError(err)
			return false
		}
		return a == glfw.True
	}, w)
}

func (w *glfwWindow) RequestAttention() {
	thread.CallWithArg(w.ui.mainThread, func(w *glfwWindow) {
		if w.ui.isTerminated() {
			return
		}
		if err := w.ui.applyWindowSettings(); err != nil {
			w.ui.setError(err)
			return
		}
		if err := w.ui.window.RequestAttention(); err != nil {
			w.ui.setError(err)
			return
		}
	}, w)
}

// applyWindowSettings must be called from the main thread.
func (u *glfwBackend) applyWindowSettings() error {
	w := &u.desktopWindow
	// macOS requires a buffer swap before initializing decoration (#2600).
	if runtime.GOOS != "darwin" || u.bufferOnceSwapped {
		if err := w.windowDecorated.apply(u.setWindowDecorated); err != nil {
			return err
		}
	}
	if err := w.windowFloating.apply(u.setWindowFloating); err != nil {
		return err
	}
	if err := w.windowMousePassthrough.apply(u.setWindowMousePassthrough); err != nil {
		return err
	}
	if err := w.title.apply(u.setWindowTitle); err != nil {
		return err
	}
	if err := w.windowClosingHandled.apply(u.setDocumentEdited); err != nil {
		return err
	}
	if w.colorModeChanged.Swap(false) {
		if err := u.setWindowColorModeImpl(u.PreferredColorMode()); err != nil {
			return err
		}
	}
	if err := w.windowResizingMode.apply(u.setWindowResizingMode); err != nil {
		return err
	}
	if err := w.windowSizeLimit.apply(func(_ windowSizeRange) error {
		return u.updateWindowSizeLimits()
	}); err != nil {
		return err
	}
	if err := u.requestedMonitor.apply(u.setWindowMonitor); err != nil {
		return err
	}
	// Position determines the device scale used when applying the size (#1982).
	if err := w.windowPositionInDIP.apply(func(p image.Point) error {
		m, err := u.currentMonitor()
		if err != nil {
			return err
		}
		return u.setWindowPositionInDIP(p.X, p.Y, m, true)
	}); err != nil {
		return err
	}
	if err := w.windowSizeInDIP.apply(func(p image.Point) error {
		maximized, err := u.isWindowMaximized()
		if err != nil {
			return err
		}
		if maximized && runtime.GOOS != "darwin" {
			return nil
		}
		return u.setWindowSizeInDIP(p.X, p.Y, true)
	}); err != nil {
		return err
	}
	// Present the first frame before showing the window (#2725).
	if u.bufferOnceSwapped || !w.isWindowVisible() {
		if err := w.windowVisible.apply(u.setWindowVisible); err != nil {
			return err
		}
	}
	return nil
}
