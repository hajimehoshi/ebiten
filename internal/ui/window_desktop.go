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
	"sync/atomic"

	"github.com/hajimehoshi/ebiten/v2/internal/colormode"
	"github.com/hajimehoshi/ebiten/v2/internal/glfw"
	"github.com/hajimehoshi/ebiten/v2/internal/microsoftgdk"
)

type windowSizeRange struct {
	minWidthInDIP  int
	minHeightInDIP int
	maxWidthInDIP  int
	maxHeightInDIP int
}

// desktopWindow is the Window implementation for the desktop build.
//
// desktopWindow holds the window settings, which can be set before the
// backend exists. Before the game starts, desktopWindow answers from these
// settings. While the game runs, desktopWindow delegates to the backend's
// window, and the backend consumes the init* settings during its
// initialization and reads the other settings whenever it needs them.
type desktopWindow struct {
	ui *UserInterface

	title atomic.Value

	windowSizeLimit atomic.Value

	iconImages           atomic.Pointer[[]image.Image]
	windowClosingHandled atomic.Bool
	windowResizingMode   atomic.Int32
	colorMode            atomic.Int32

	initWindowDecorated        atomic.Bool
	initWindowPositionInDIP    atomic.Value
	initWindowSizeInDIP        atomic.Value
	initWindowFloating         atomic.Bool
	initWindowMaximized        atomic.Bool
	initWindowMousePassthrough atomic.Bool
}

var _ Window = (*desktopWindow)(nil)

func (w *desktopWindow) init() {
	w.title.Store("")
	w.windowSizeLimit.Store(windowSizeRange{
		minWidthInDIP:  glfw.DontCare,
		minHeightInDIP: glfw.DontCare,
		maxWidthInDIP:  glfw.DontCare,
		maxHeightInDIP: glfw.DontCare,
	})
	w.initWindowDecorated.Store(true)
	w.initWindowPositionInDIP.Store(image.Pt(invalidPos, invalidPos))
	w.initWindowSizeInDIP.Store(image.Pt(640, 480))
}

func (w *desktopWindow) getWindowSizeLimitsInDIP() (minw, minh, maxw, maxh int) {
	if microsoftgdk.IsXbox() {
		return glfw.DontCare, glfw.DontCare, glfw.DontCare, glfw.DontCare
	}

	s := w.windowSizeLimit.Load().(windowSizeRange)
	return s.minWidthInDIP, s.minHeightInDIP, s.maxWidthInDIP, s.maxHeightInDIP
}

func (w *desktopWindow) setWindowSizeLimitsInDIP(minw, minh, maxw, maxh int) bool {
	if microsoftgdk.IsXbox() {
		// Do nothing. The size is always fixed.
		return false
	}

	newS := windowSizeRange{
		minWidthInDIP:  minw,
		minHeightInDIP: minh,
		maxWidthInDIP:  maxw,
		maxHeightInDIP: maxh,
	}
	return w.windowSizeLimit.Swap(newS) != newS
}

func (w *desktopWindow) isWindowMaximizable() bool {
	_, _, maxw, maxh := w.getWindowSizeLimitsInDIP()
	return maxw == glfw.DontCare && maxh == glfw.DontCare
}

// adjustWindowSizeBasedOnSizeLimitsInDIP adjust the size based on the window size limits.
// width and height are in device-independent pixels.
func (w *desktopWindow) adjustWindowSizeBasedOnSizeLimitsInDIP(width, height int) (int, int) {
	minw, minh, maxw, maxh := w.getWindowSizeLimitsInDIP()
	if minw >= 0 && width < minw {
		width = minw
	}
	if minh >= 0 && height < minh {
		height = minh
	}
	if maxw >= 0 && width > maxw {
		width = maxw
	}
	if maxh >= 0 && height > maxh {
		height = maxh
	}
	return width, height
}

func (w *desktopWindow) isInitWindowDecorated() bool {
	return w.initWindowDecorated.Load()
}

func (w *desktopWindow) setInitWindowDecorated(decorated bool) {
	w.initWindowDecorated.Store(decorated)
}

func (w *desktopWindow) getAndResetIconImages() []image.Image {
	images := w.iconImages.Swap(nil)
	if images == nil {
		return nil
	}
	return *images
}

func (w *desktopWindow) setIconImages(iconImages []image.Image) {
	// Even if iconImages is nil, always create a slice.
	// A 0-size slice and nil are distinguished.
	// See the comment in updateIconIfNeeded.
	newImages := make([]image.Image, len(iconImages))
	copy(newImages, iconImages)
	w.iconImages.Store(&newImages)
}

func (w *desktopWindow) getInitWindowPositionInDIP() (int, int) {
	if microsoftgdk.IsXbox() {
		return 0, 0
	}

	pt := w.initWindowPositionInDIP.Load().(image.Point)
	if pt.X != invalidPos && pt.Y != invalidPos {
		return pt.X, pt.Y
	}
	return invalidPos, invalidPos
}

func (w *desktopWindow) setInitWindowPositionInDIP(x, y int) {
	if microsoftgdk.IsXbox() {
		return
	}

	// TODO: Update initMonitor if necessary (#1575).
	w.initWindowPositionInDIP.Store(image.Pt(x, y))
}

func (w *desktopWindow) getInitWindowSizeInDIP() (int, int) {
	if microsoftgdk.IsXbox() {
		return microsoftgdk.MonitorResolution()
	}

	pt := w.initWindowSizeInDIP.Load().(image.Point)
	return pt.X, pt.Y
}

func (w *desktopWindow) setInitWindowSizeInDIP(width, height int) {
	if microsoftgdk.IsXbox() {
		return
	}

	w.initWindowSizeInDIP.Store(image.Pt(width, height))
}

func (w *desktopWindow) isInitWindowFloating() bool {
	if microsoftgdk.IsXbox() {
		return false
	}
	return w.initWindowFloating.Load()
}

func (w *desktopWindow) setInitWindowFloating(floating bool) {
	if microsoftgdk.IsXbox() {
		return
	}

	w.initWindowFloating.Store(floating)
}

func (w *desktopWindow) isInitWindowMaximized() bool {
	// TODO: Is this always true on Xbox?
	return w.initWindowMaximized.Load()
}

func (w *desktopWindow) setInitWindowMaximized(maximized bool) {
	w.initWindowMaximized.Store(maximized)
}

func (w *desktopWindow) isInitWindowMousePassthrough() bool {
	return w.initWindowMousePassthrough.Load()
}

func (w *desktopWindow) setInitWindowMousePassthrough(enabled bool) {
	w.initWindowMousePassthrough.Store(enabled)
}

func (w *desktopWindow) isWindowClosingHandled() bool {
	return w.windowClosingHandled.Load()
}

func (w *desktopWindow) IsDecorated() bool {
	if w.ui.isTerminated() {
		return false
	}
	if !w.ui.isRunning() {
		return w.isInitWindowDecorated()
	}
	return w.ui.backend.Window().IsDecorated()
}

func (w *desktopWindow) SetDecorated(decorated bool) {
	if w.ui.isTerminated() {
		return
	}
	if !w.ui.isRunning() {
		w.setInitWindowDecorated(decorated)
		return
	}
	w.ui.backend.Window().SetDecorated(decorated)
}

func (w *desktopWindow) ResizingMode() WindowResizingMode {
	if w.ui.isTerminated() {
		return 0
	}
	return WindowResizingMode(w.windowResizingMode.Load())
}

func (w *desktopWindow) SetResizingMode(mode WindowResizingMode) {
	if w.ui.isTerminated() {
		return
	}
	if WindowResizingMode(w.windowResizingMode.Swap(int32(mode))) == mode {
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
		return w.isInitWindowFloating()
	}
	return w.ui.backend.Window().IsFloating()
}

func (w *desktopWindow) SetFloating(floating bool) {
	if w.ui.isTerminated() {
		return
	}
	if !w.ui.isRunning() {
		w.setInitWindowFloating(floating)
		return
	}
	w.ui.backend.Window().SetFloating(floating)
}

func (w *desktopWindow) IsMaximized() bool {
	if w.ui.isTerminated() {
		return false
	}
	if !w.ui.isRunning() {
		return w.isInitWindowMaximized()
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

	if !w.isWindowMaximizable() {
		return
	}

	if !w.ui.isRunning() {
		w.setInitWindowMaximized(true)
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
	if !w.isWindowMaximizable() {
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
		w.setInitWindowPositionInDIP(x, y)
		return
	}
	w.ui.backend.Window().SetPosition(x, y)
}

func (w *desktopWindow) Size() (int, int) {
	if w.ui.isTerminated() {
		return 0, 0
	}
	if !w.ui.isRunning() {
		ww, wh := w.getInitWindowSizeInDIP()
		return w.adjustWindowSizeBasedOnSizeLimitsInDIP(ww, wh)
	}
	return w.ui.backend.Window().Size()
}

func (w *desktopWindow) SetSize(width, height int) {
	if w.ui.isTerminated() {
		return
	}
	if !w.ui.isRunning() {
		// If the window is initially maximized, the set size is ignored anyway.
		w.setInitWindowSizeInDIP(width, height)
		return
	}
	w.ui.backend.Window().SetSize(width, height)
}

func (w *desktopWindow) SizeLimits() (minw, minh, maxw, maxh int) {
	return w.getWindowSizeLimitsInDIP()
}

func (w *desktopWindow) SetSizeLimits(minw, minh, maxw, maxh int) {
	if w.ui.isTerminated() {
		return
	}
	if !w.setWindowSizeLimitsInDIP(minw, minh, maxw, maxh) {
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
	w.setIconImages(iconImages)
}

func (w *desktopWindow) SetTitle(title string) {
	if w.ui.isTerminated() {
		return
	}
	if w.title.Swap(title) == title {
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
	return colormode.ColorMode(w.colorMode.Load())
}

func (w *desktopWindow) SetColorMode(mode colormode.ColorMode) {
	if w.ui.isTerminated() {
		return
	}
	if !w.ui.isRunning() {
		w.colorMode.Store(int32(mode))
		return
	}
	if colormode.ColorMode(w.colorMode.Swap(int32(mode))) == mode {
		return
	}
	w.ui.backend.Window().SetColorMode(mode)
}

func (w *desktopWindow) SetClosingHandled(handled bool) {
	if w.ui.isTerminated() {
		return
	}
	if w.windowClosingHandled.Swap(handled) == handled {
		return
	}
	if !w.ui.isRunning() {
		return
	}
	w.ui.backend.Window().SetClosingHandled(handled)
}

func (w *desktopWindow) IsClosingHandled() bool {
	return w.isWindowClosingHandled()
}

func (w *desktopWindow) SetMousePassthrough(enabled bool) {
	if w.ui.isTerminated() {
		return
	}
	if !w.ui.isRunning() {
		w.setInitWindowMousePassthrough(enabled)
		return
	}
	w.ui.backend.Window().SetMousePassthrough(enabled)
}

func (w *desktopWindow) IsMousePassthrough() bool {
	if w.ui.isTerminated() {
		return false
	}
	if !w.ui.isRunning() {
		return w.isInitWindowMousePassthrough()
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
