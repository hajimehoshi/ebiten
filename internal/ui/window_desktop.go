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

	"github.com/hajimehoshi/ebiten/v2/internal/glfw"
	"github.com/hajimehoshi/ebiten/v2/internal/microsoftgdk"
)

// windowSetting holds a requested value and the last request applied by the backend.
type windowSetting[T comparable] struct {
	value   atomic.Pointer[T]
	applied atomic.Pointer[T]
}

func (s *windowSetting[T]) Load() T {
	if p := s.value.Load(); p != nil {
		return *p
	}
	var zero T
	return zero
}

func (s *windowSetting[T]) Store(value T) bool {
	for {
		old := s.value.Load()
		if old != nil && *old == value {
			return false
		}
		if s.value.CompareAndSwap(old, &value) {
			return true
		}
	}
}

func (s *windowSetting[T]) init(value T) {
	s.value.CompareAndSwap(nil, &value)
}

func (s *windowSetting[T]) pending() *T {
	p := s.value.Load()
	if p == s.applied.Load() {
		return nil
	}
	return p
}

func (s *windowSetting[T]) apply(f func(T) error) error {
	p := s.pending()
	if p == nil {
		return nil
	}
	if err := f(*p); err != nil {
		return err
	}
	s.markApplied(p)
	return nil
}

func (s *windowSetting[T]) markApplied(request *T) {
	// A newer request stored during application must remain pending.
	s.applied.Store(request)
}

type windowSizeRange struct {
	minWidthInDIP  int
	minHeightInDIP int
	maxWidthInDIP  int
	maxHeightInDIP int
}

// desktopWindow holds the requested desktop window settings.
type desktopWindow struct {
	ui *UserInterface

	title windowSetting[string]

	windowSizeLimit windowSetting[windowSizeRange]

	iconImages           atomic.Pointer[[]image.Image]
	windowClosingHandled windowSetting[bool]
	windowResizingMode   windowSetting[WindowResizingMode]

	windowDecorated        windowSetting[bool]
	windowVisible          windowSetting[bool]
	windowPositionInDIP    windowSetting[image.Point]
	windowSizeInDIP        windowSetting[image.Point]
	windowFloating         windowSetting[bool]
	windowMousePassthrough windowSetting[bool]

	colorModeChanged atomic.Bool
	settingsChanged  atomic.Bool

	initWindowMaximized atomic.Bool
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
	w.windowDecorated.Store(true)
	w.windowVisible.Store(true)
	p := image.Pt(640, 480)
	w.windowSizeInDIP.Store(p)
	w.settingsChanged.Store(true)
}

func (w *desktopWindow) getWindowSizeLimitsInDIP() (minw, minh, maxw, maxh int) {
	if microsoftgdk.IsXbox() {
		return glfw.DontCare, glfw.DontCare, glfw.DontCare, glfw.DontCare
	}

	s := w.windowSizeLimit.Load()
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
	return w.windowSizeLimit.Store(newS)
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

func (w *desktopWindow) isWindowDecorated() bool {
	return w.windowDecorated.Load()
}

func (w *desktopWindow) setWindowDecorated(decorated bool) bool {
	return w.windowDecorated.Store(decorated)
}

func (w *desktopWindow) isWindowVisible() bool {
	return w.windowVisible.Load()
}

func (w *desktopWindow) setWindowVisible(visible bool) bool {
	return w.windowVisible.Store(visible)
}

func (w *desktopWindow) getIconImages() *[]image.Image {
	return w.iconImages.Load()
}

// resetIconImages resets the pending icon images.
// If new icon images are set after imgs is obtained, nothing happens.
func (w *desktopWindow) resetIconImages(imgs *[]image.Image) {
	w.iconImages.CompareAndSwap(imgs, nil)
}

func (w *desktopWindow) setIconImages(iconImages []image.Image) {
	// Even if iconImages is nil, always create a slice.
	// A 0-size slice and nil are distinguished.
	// See the comment in updateIconIfNeeded.
	newImages := make([]image.Image, len(iconImages))
	copy(newImages, iconImages)
	w.iconImages.Store(&newImages)
}

func (w *desktopWindow) setWindowPositionInDIP(x, y int) bool {
	if microsoftgdk.IsXbox() {
		return false
	}

	// TODO: Update requestedMonitor if necessary (#1575).
	pt := image.Pt(x, y)
	return w.windowPositionInDIP.Store(pt)
}

func (w *desktopWindow) getWindowSizeInDIP() (int, int) {
	if microsoftgdk.IsXbox() {
		return microsoftgdk.MonitorResolution()
	}

	pt := w.windowSizeInDIP.Load()
	return pt.X, pt.Y
}

func (w *desktopWindow) setWindowSizeInDIP(width, height int) bool {
	if microsoftgdk.IsXbox() {
		return false
	}

	pt := image.Pt(width, height)
	return w.windowSizeInDIP.Store(pt)
}

func (w *desktopWindow) isWindowFloating() bool {
	if microsoftgdk.IsXbox() {
		return false
	}
	return w.windowFloating.Load()
}

func (w *desktopWindow) setWindowFloating(floating bool) bool {
	if microsoftgdk.IsXbox() {
		return false
	}

	return w.windowFloating.Store(floating)
}

func (w *desktopWindow) isInitWindowMaximized() bool {
	// TODO: Is this always true on Xbox?
	return w.initWindowMaximized.Load()
}

func (w *desktopWindow) setInitWindowMaximized(maximized bool) {
	w.initWindowMaximized.Store(maximized)
}

func (w *desktopWindow) isWindowMousePassthrough() bool {
	return w.windowMousePassthrough.Load()
}

func (w *desktopWindow) setWindowMousePassthrough(enabled bool) bool {
	return w.windowMousePassthrough.Store(enabled)
}

func (w *desktopWindow) isWindowClosingHandled() bool {
	return w.windowClosingHandled.Load()
}

func (w *desktopWindow) IsDecorated() bool {
	if w.ui.isTerminated() {
		return false
	}
	b := w.ui.runningBackend()
	if b == nil {
		return w.isWindowDecorated()
	}
	return b.Window().IsDecorated()
}

func (w *desktopWindow) SetDecorated(decorated bool) {
	if w.ui.isTerminated() {
		return
	}
	if !w.setWindowDecorated(decorated) {
		return
	}
	w.scheduleUpdate()
}

func (w *desktopWindow) IsVisible() bool {
	if w.ui.isTerminated() {
		return false
	}
	b := w.ui.runningBackend()
	if b == nil {
		return w.isWindowVisible()
	}
	return b.Window().IsVisible()
}

func (w *desktopWindow) SetVisible(visible bool) {
	if w.ui.isTerminated() {
		return
	}
	if !w.setWindowVisible(visible) {
		return
	}
	w.scheduleUpdate()
}

func (w *desktopWindow) ResizingMode() WindowResizingMode {
	if w.ui.isTerminated() {
		return 0
	}
	return w.windowResizingMode.Load()
}

func (w *desktopWindow) SetResizingMode(mode WindowResizingMode) {
	if w.ui.isTerminated() {
		return
	}
	if !w.windowResizingMode.Store(mode) {
		return
	}
	w.scheduleUpdate()
}

func (w *desktopWindow) IsFloating() bool {
	if w.ui.isTerminated() {
		return false
	}
	b := w.ui.runningBackend()
	if b == nil {
		return w.isWindowFloating()
	}
	return b.Window().IsFloating()
}

func (w *desktopWindow) SetFloating(floating bool) {
	if w.ui.isTerminated() {
		return
	}
	if !w.setWindowFloating(floating) {
		return
	}
	w.scheduleUpdate()
}

func (w *desktopWindow) IsMaximized() bool {
	if w.ui.isTerminated() {
		return false
	}
	b := w.ui.runningBackend()
	if b == nil {
		return w.isInitWindowMaximized()
	}
	if w.ResizingMode() != WindowResizingModeEnabled {
		return false
	}
	return b.Window().IsMaximized()
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

	w.setInitWindowMaximized(true)
	b := w.ui.runningBackend()
	if b == nil {
		return
	}
	b.Window().Maximize()
}

func (w *desktopWindow) IsMinimized() bool {
	b := w.ui.runningBackend()
	if b == nil {
		return false
	}
	return b.Window().IsMinimized()
}

func (w *desktopWindow) Minimize() {
	b := w.ui.runningBackend()
	if b == nil {
		// Do nothing
		return
	}
	b.Window().Minimize()
}

func (w *desktopWindow) Restore() {
	if w.ui.isTerminated() {
		return
	}
	if !w.isWindowMaximizable() {
		return
	}
	b := w.ui.runningBackend()
	if b == nil {
		// Do nothing
		return
	}
	b.Window().Restore()
}

func (w *desktopWindow) SetMonitor(monitor *Monitor) {
	if monitor == nil {
		panic("ui: monitor cannot be nil at SetMonitor")
	}
	if w.ui.isTerminated() {
		return
	}
	if !w.ui.setRequestedMonitor(monitor) {
		return
	}
	w.scheduleUpdate()
}

func (w *desktopWindow) Position() (int, int) {
	if w.ui.isTerminated() {
		return 0, 0
	}
	b := w.ui.runningBackend()
	if b == nil {
		// The default position depends on the monitor, and getting a monitor initializes GLFW,
		// which must not happen here. Only an explicitly set position is available.
		pt := w.windowPositionInDIP.value.Load()
		if pt == nil {
			return 0, 0
		}
		return pt.X, pt.Y
	}
	return b.Window().Position()
}

func (w *desktopWindow) SetPosition(x, y int) {
	if w.ui.isTerminated() {
		return
	}
	if !w.setWindowPositionInDIP(x, y) {
		return
	}
	w.scheduleUpdate()
}

func (w *desktopWindow) Size() (int, int) {
	if w.ui.isTerminated() {
		return 0, 0
	}
	b := w.ui.runningBackend()
	if b == nil {
		ww, wh := w.getWindowSizeInDIP()
		return w.adjustWindowSizeBasedOnSizeLimitsInDIP(ww, wh)
	}
	return b.Window().Size()
}

func (w *desktopWindow) SetSize(width, height int) {
	if w.ui.isTerminated() {
		return
	}
	if !w.setWindowSizeInDIP(width, height) {
		return
	}
	w.scheduleUpdate()
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
	w.scheduleUpdate()
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
	if !w.title.Store(title) {
		return
	}
	w.scheduleUpdate()
}

func (w *desktopWindow) applyColorMode() {
	if w.ui.isTerminated() {
		return
	}
	w.colorModeChanged.Store(true)
	w.scheduleUpdate()
}

func (w *desktopWindow) SetClosingHandled(handled bool) {
	if w.ui.isTerminated() {
		return
	}
	if !w.windowClosingHandled.Store(handled) {
		return
	}
	w.scheduleUpdate()
}

func (w *desktopWindow) IsClosingHandled() bool {
	return w.isWindowClosingHandled()
}

func (w *desktopWindow) SetMousePassthrough(enabled bool) {
	if w.ui.isTerminated() {
		return
	}
	if !w.setWindowMousePassthrough(enabled) {
		return
	}
	w.scheduleUpdate()
}

func (w *desktopWindow) IsMousePassthrough() bool {
	if w.ui.isTerminated() {
		return false
	}
	b := w.ui.runningBackend()
	if b == nil {
		return w.isWindowMousePassthrough()
	}
	return b.Window().IsMousePassthrough()
}

func (w *desktopWindow) RequestAttention() {
	if w.ui.isTerminated() {
		return
	}
	b := w.ui.runningBackend()
	if b == nil {
		// Do nothing
		return
	}
	b.Window().RequestAttention()
}

func (w *desktopWindow) scheduleUpdate() {
	w.settingsChanged.Store(true)
	w.ui.ScheduleFrame()
}
