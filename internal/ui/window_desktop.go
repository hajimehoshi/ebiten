// Copyright 2026 The Ebitengine Authors
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

	"github.com/hajimehoshi/ebiten/v2/internal/colormode"
)

// desktopWindow is the Window implementation for the desktop build.
// Before the game starts, desktopWindow answers from the settings in
// userInterfaceImpl. While the game runs, desktopWindow delegates to the
// backend's window.
type desktopWindow struct {
	ui *UserInterface
}

var _ Window = (*desktopWindow)(nil)

func (w *desktopWindow) IsDecorated() bool {
	if w.ui.isTerminated() {
		return false
	}
	if !w.ui.isRunning() {
		return w.ui.isInitWindowDecorated()
	}
	return w.ui.backend.Window().IsDecorated()
}

func (w *desktopWindow) SetDecorated(decorated bool) {
	if w.ui.isTerminated() {
		return
	}
	if !w.ui.isRunning() {
		w.ui.setInitWindowDecorated(decorated)
		return
	}
	w.ui.backend.Window().SetDecorated(decorated)
}

func (w *desktopWindow) ResizingMode() WindowResizingMode {
	if w.ui.isTerminated() {
		return 0
	}
	return WindowResizingMode(w.ui.windowResizingMode.Load())
}

func (w *desktopWindow) SetResizingMode(mode WindowResizingMode) {
	if w.ui.isTerminated() {
		return
	}
	if WindowResizingMode(w.ui.windowResizingMode.Swap(int32(mode))) == mode {
		return
	}
	if !w.ui.isRunning() {
		return
	}
	w.ui.backend.Window().SetResizingMode(mode)
}

func (w *desktopWindow) IsFloating() bool {
	if w.ui.isTerminated() {
		return false
	}
	if !w.ui.isRunning() {
		return w.ui.isInitWindowFloating()
	}
	return w.ui.backend.Window().IsFloating()
}

func (w *desktopWindow) SetFloating(floating bool) {
	if w.ui.isTerminated() {
		return
	}
	if !w.ui.isRunning() {
		w.ui.setInitWindowFloating(floating)
		return
	}
	w.ui.backend.Window().SetFloating(floating)
}

func (w *desktopWindow) IsMaximized() bool {
	if w.ui.isTerminated() {
		return false
	}
	if !w.ui.isRunning() {
		return w.ui.isInitWindowMaximized()
	}
	if w.ResizingMode() != WindowResizingModeEnabled {
		return false
	}
	return w.ui.backend.Window().IsMaximized()
}

func (w *desktopWindow) Maximize() {
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
	w.ui.backend.Window().Maximize()
}

func (w *desktopWindow) IsMinimized() bool {
	if !w.ui.isRunning() {
		return false
	}
	return w.ui.backend.Window().IsMinimized()
}

func (w *desktopWindow) Minimize() {
	if !w.ui.isRunning() {
		// Do nothing
		return
	}
	w.ui.backend.Window().Minimize()
}

func (w *desktopWindow) Restore() {
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
	w.ui.backend.Window().Restore()
}

func (w *desktopWindow) SetMonitor(monitor *Monitor) {
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
	w.ui.backend.Window().SetMonitor(monitor)
}

func (w *desktopWindow) Position() (int, int) {
	if w.ui.isTerminated() {
		return 0, 0
	}
	if !w.ui.isRunning() {
		panic("ui: WindowPosition can't be called before the main loop starts")
	}
	return w.ui.backend.Window().Position()
}

func (w *desktopWindow) SetPosition(x, y int) {
	if w.ui.isTerminated() {
		return
	}
	if !w.ui.isRunning() {
		w.ui.setInitWindowPositionInDIP(x, y)
		return
	}
	w.ui.backend.Window().SetPosition(x, y)
}

func (w *desktopWindow) Size() (int, int) {
	if w.ui.isTerminated() {
		return 0, 0
	}
	if !w.ui.isRunning() {
		ww, wh := w.ui.getInitWindowSizeInDIP()
		return w.ui.adjustWindowSizeBasedOnSizeLimitsInDIP(ww, wh)
	}
	return w.ui.backend.Window().Size()
}

func (w *desktopWindow) SetSize(width, height int) {
	if w.ui.isTerminated() {
		return
	}
	if !w.ui.isRunning() {
		// If the window is initially maximized, the set size is ignored anyway.
		w.ui.setInitWindowSizeInDIP(width, height)
		return
	}
	w.ui.backend.Window().SetSize(width, height)
}

func (w *desktopWindow) SizeLimits() (minw, minh, maxw, maxh int) {
	return w.ui.getWindowSizeLimitsInDIP()
}

func (w *desktopWindow) SetSizeLimits(minw, minh, maxw, maxh int) {
	if w.ui.isTerminated() {
		return
	}
	if !w.ui.setWindowSizeLimitsInDIP(minw, minh, maxw, maxh) {
		return
	}
	if !w.ui.isRunning() {
		return
	}
	w.ui.backend.Window().SetSizeLimits(minw, minh, maxw, maxh)
}

func (w *desktopWindow) SetIcon(iconImages []image.Image) {
	if w.ui.isTerminated() {
		return
	}
	// The icons are actually set at updateIconIfNeeded.
	w.ui.setIconImages(iconImages)
}

func (w *desktopWindow) SetTitle(title string) {
	if w.ui.isTerminated() {
		return
	}
	if w.ui.title.Swap(title) == title {
		return
	}
	if !w.ui.isRunning() {
		return
	}
	w.ui.backend.Window().SetTitle(title)
}

func (w *desktopWindow) ColorMode() colormode.ColorMode {
	if w.ui.isTerminated() {
		return colormode.Unknown
	}
	return colormode.ColorMode(w.ui.colorMode.Load())
}

func (w *desktopWindow) SetColorMode(mode colormode.ColorMode) {
	if w.ui.isTerminated() {
		return
	}
	if !w.ui.isRunning() {
		w.ui.colorMode.Store(int32(mode))
		return
	}
	if colormode.ColorMode(w.ui.colorMode.Swap(int32(mode))) == mode {
		return
	}
	w.ui.backend.Window().SetColorMode(mode)
}

func (w *desktopWindow) SetClosingHandled(handled bool) {
	if w.ui.isTerminated() {
		return
	}
	if w.ui.windowClosingHandled.Swap(handled) == handled {
		return
	}
	if !w.ui.isRunning() {
		return
	}
	w.ui.backend.Window().SetClosingHandled(handled)
}

func (w *desktopWindow) IsClosingHandled() bool {
	return w.ui.isWindowClosingHandled()
}

func (w *desktopWindow) SetMousePassthrough(enabled bool) {
	if w.ui.isTerminated() {
		return
	}
	if !w.ui.isRunning() {
		w.ui.setInitWindowMousePassthrough(enabled)
		return
	}
	w.ui.backend.Window().SetMousePassthrough(enabled)
}

func (w *desktopWindow) IsMousePassthrough() bool {
	if w.ui.isTerminated() {
		return false
	}
	if !w.ui.isRunning() {
		return w.ui.isInitWindowMousePassthrough()
	}
	return w.ui.backend.Window().IsMousePassthrough()
}

func (w *desktopWindow) RequestAttention() {
	if w.ui.isTerminated() {
		return
	}
	if !w.ui.isRunning() {
		// Do nothing
		return
	}
	w.ui.backend.Window().RequestAttention()
}
