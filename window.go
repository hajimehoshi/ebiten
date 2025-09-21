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

package ebiten

import (
	"image"
	"sync/atomic"

	"github.com/hajimehoshi/ebiten/v2/internal/inputstate"
	"github.com/hajimehoshi/ebiten/v2/internal/ui"
)

// WindowResizingModeType represents a mode in which a user resizes the window.
//
// Regardless of the resizing mode, an Ebitengine application can still change the window size or make
// the window fullscreen by calling Ebitengine functions.
type WindowResizingModeType int

// WindowResizingModeTypes
const (
	// WindowResizingModeDisabled indicates the mode to disallow resizing the window by a user.
	WindowResizingModeDisabled WindowResizingModeType = WindowResizingModeType(ui.WindowResizingModeDisabled)

	// WindowResizingModeOnlyFullscreenEnabled indicates the mode to disallow resizing the window,
	// but allow to make the window fullscreen by a user.
	// This works only on macOS so far.
	// On the other platforms, this is the same as WindowResizingModeDisabled.
	WindowResizingModeOnlyFullscreenEnabled WindowResizingModeType = WindowResizingModeType(ui.WindowResizingModeOnlyFullscreenEnabled)

	// WindowResizingModeEnabled indicates the mode to allow resizing the window by a user.
	WindowResizingModeEnabled WindowResizingModeType = WindowResizingModeType(ui.WindowResizingModeEnabled)
)

// IsWindowDecorated reports whether the window is decorated.
//
// IsWindowDecorated is concurrent-safe.
func IsWindowDecorated() bool {
	return ui.Get().Window().IsDecorated()
}

// SetWindowDecorated sets the state if the window is decorated.
//
// The window is decorated by default.
//
// SetWindowDecorated works only on desktops.
// SetWindowDecorated does nothing if the platform is not a desktop.
//
// SetWindowDecorated is concurrent-safe.
func SetWindowDecorated(decorated bool) {
	ui.Get().Window().SetDecorated(decorated)
}

// WindowResizingMode returns the current mode in which a user resizes the window.
//
// The default mode is WindowResizingModeDisabled.
//
// WindowResizingMode is concurrent-safe.
func WindowResizingMode() WindowResizingModeType {
	return WindowResizingModeType(ui.Get().Window().ResizingMode())
}

// SetWindowResizingMode sets the mode in which a user resizes the window.
//
// SetWindowResizingMode is concurrent-safe.
func SetWindowResizingMode(mode WindowResizingModeType) {
	ui.Get().Window().SetResizingMode(ui.WindowResizingMode(mode))
}

// IsWindowResizable reports whether the window is resizable by the user's dragging on desktops.
// On the other environments, IsWindowResizable always returns false.
//
// Deprecated: as of v2.3. Use WindowResizingMode instead.
func IsWindowResizable() bool {
	return ui.Get().Window().ResizingMode() == ui.WindowResizingModeEnabled
}

// SetWindowResizable sets whether the window is resizable by the user's dragging on desktops.
// On the other environments, SetWindowResizable does nothing.
//
// Deprecated: as of v2.3, Use SetWindowResizingMode instead.
func SetWindowResizable(resizable bool) {
	mode := ui.WindowResizingModeDisabled
	if resizable {
		mode = ui.WindowResizingModeEnabled
	}
	ui.Get().Window().SetResizingMode(mode)
}

// SetWindowTitle sets the title of the window.
//
// SetWindowTitle does nothing if the platform is not a desktop.
//
// SetWindowTitle is concurrent-safe.
func SetWindowTitle(title string) {
	ui.Get().Window().SetTitle(title)
}

// SetWindowIcon sets the icon of the game window.
//
// If len(iconImages) is 0, SetWindowIcon reverts the icon to the default one.
//
// For desktops, see the document of glfwSetWindowIcon of GLFW 3.2:
//
//	This function sets the icon of the specified window.
//	If passed an array of candidate images, those of or closest to the sizes
//	desired by the system are selected.
//	If no images are specified, the window reverts to its default icon.
//
//	The desired image sizes varies depending on platform and system settings.
//	The selected images will be rescaled as needed.
//	Good sizes include 16x16, 32x32 and 48x48.
//
// As macOS windows don't have icons, SetWindowIcon doesn't work on macOS.
//
// SetWindowIcon doesn't work if the platform is not a desktop.
//
// SetWindowIcon is concurrent-safe.
func SetWindowIcon(iconImages []image.Image) {
	ui.Get().Window().SetIcon(iconImages)
}

// WindowPosition returns the window position.
// The origin position is the upper-left corner of the current monitor.
// The unit is device-independent pixels.
//
// WindowPosition panics if the main loop does not start yet.
//
// WindowPosition returns the original window position in fullscreen mode.
//
// WindowPosition returns (0, 0) if the platform is not a desktop.
//
// WindowPosition is concurrent-safe.
func WindowPosition() (x, y int) {
	return ui.Get().Window().Position()
}

// SetWindowPosition sets the window position.
// The origin position is the upper-left corner of the current monitor.
// The unit is device-independent pixels.
//
// SetWindowPosition sets the original window position in fullscreen mode.
//
// SetWindowPosition does nothing if the platform is not a desktop.
//
// SetWindowPosition is concurrent-safe.
func SetWindowPosition(x, y int) {
	windowPositionSetExplicitly.Store(true)
	ui.Get().Window().SetPosition(x, y)
}

var (
	windowPositionSetExplicitly atomic.Bool
)

func initializeWindowPositionIfNeeded(width, height int) {
	if !windowPositionSetExplicitly.Load() {
		sw, sh := ui.Get().Monitor().Size()
		x, y := ui.InitialWindowPosition(sw, sh, width, height)
		ui.Get().Window().SetPosition(x, y)
	}
}

// WindowSize returns the window size on desktops.
// WindowSize returns (0, 0) on other environments.
//
// Even if the application is in fullscreen mode, WindowSize returns the original window size.
// If you need the fullscreen dimensions, see Monitor().Size() instead.
//
// WindowSize is concurrent-safe.
func WindowSize() (int, int) {
	return ui.Get().Window().Size()
}

// SetWindowSize sets the window size on desktops.
// SetWindowSize does nothing on other environments.
//
// Even if the application is in fullscreen mode, SetWindowSize sets the original window size.
//
// SetWindowSize panics if width or height is not a positive number.
//
// SetWindowSize is concurrent-safe.
func SetWindowSize(width, height int) {
	if width <= 0 || height <= 0 {
		panic("ebiten: width and height must be positive")
	}
	ui.Get().Window().SetSize(width, height)
}

// WindowSizeLimits returns the limitation of the window size on desktops.
// A negative value indicates the size is not limited.
//
// WindowSizeLimits is concurrent-safe.
func WindowSizeLimits() (minw, minh, maxw, maxh int) {
	return ui.Get().Window().SizeLimits()
}

// SetWindowSizeLimits sets the limitation of the window size on desktops.
// A negative value indicates the size is not limited.
//
// SetWindowSizeLimits is concurrent-safe.
func SetWindowSizeLimits(minw, minh, maxw, maxh int) {
	ui.Get().Window().SetSizeLimits(minw, minh, maxw, maxh)
}

// IsWindowFloating reports whether the window is always shown above all the other windows.
//
// IsWindowFloating returns false if the platform is not a desktop.
//
// IsWindowFloating is concurrent-safe.
func IsWindowFloating() bool {
	return ui.Get().Window().IsFloating()
}

// SetWindowFloating sets the state whether the window is always shown above all the other windows.
//
// SetWindowFloating does nothing if the platform is not a desktop.
//
// SetWindowFloating is concurrent-safe.
func SetWindowFloating(float bool) {
	ui.Get().Window().SetFloating(float)
}

// MaximizeWindow maximizes the window.
//
// MaximizeWindow does nothing when the window is not resizable (WindowResizingModeEnabled).
//
// MaximizeWindow does nothing if the platform is not a desktop.
//
// MaximizeWindow is concurrent-safe.
func MaximizeWindow() {
	ui.Get().Window().Maximize()
}

// IsWindowMaximized reports whether the window is maximized or not.
//
// IsWindowMaximized returns false when the window is not resizable (WindowResizingModeEnabled).
//
// IsWindowMaximized always returns false if the platform is not a desktop.
//
// IsWindowMaximized is concurrent-safe.
func IsWindowMaximized() bool {
	return ui.Get().Window().IsMaximized()
}

// MinimizeWindow minimizes the window.
//
// If the main loop does not start yet, MinimizeWindow does nothing.
//
// MinimizeWindow does nothing if the platform is not a desktop.
//
// MinimizeWindow is concurrent-safe.
func MinimizeWindow() {
	ui.Get().Window().Minimize()
}

// IsWindowMinimized reports whether the window is minimized or not.
//
// IsWindowMinimized always returns false if the platform is not a desktop.
//
// IsWindowMinimized is concurrent-safe.
func IsWindowMinimized() bool {
	return ui.Get().Window().IsMinimized()
}

// RestoreWindow restores the window from its maximized or minimized state.
//
// RestoreWindow panics when the window is not maximized nor minimized.
//
// RestoreWindow is concurrent-safe.
func RestoreWindow() {
	if !IsWindowMaximized() && !IsWindowMinimized() {
		panic("ebiten: RestoreWindow must be called on a maximized or a minimized window")
	}
	ui.Get().Window().Restore()
}

// IsWindowBeingClosed returns true when the user is trying to close the window on desktops.
// As the window is closed immediately by default,
// you might want to call SetWindowClosingHandled(true) to prevent the window is automatically closed.
//
// IsWindowBeingClosed always returns false if the platform is not a desktop.
//
// IsWindowBeingClosed is concurrent-safe.
func IsWindowBeingClosed() bool {
	return inputstate.Get().WindowBeingClosed()
}

// SetWindowClosingHandled sets whether the window closing is handled or not on desktops. The default state is false.
//
// If the window closing is handled, the window is not closed immediately and
// the game can know whether the window is being closed or not by IsWindowBeingClosed.
// In this case, the window is not closed automatically.
// To end the game, you have to return an error value at the Game's Update function.
//
// SetWindowClosingHandled works only on desktops.
// SetWindowClosingHandled does nothing if the platform is not a desktop.
//
// SetWindowClosingHandled is concurrent-safe.
func SetWindowClosingHandled(handled bool) {
	ui.Get().Window().SetClosingHandled(handled)
}

// IsWindowClosingHandled reports whether the window closing is handled or not on desktops by SetWindowClosingHandled.
//
// IsWindowClosingHandled always returns false if the platform is not a desktop.
//
// IsWindowClosingHandled is concurrent-safe.
func IsWindowClosingHandled() bool {
	return ui.Get().Window().IsClosingHandled()
}

// SetWindowMousePassthrough sets whether a mouse cursor passthroughs the window or not on desktops. The default state is false.
//
// Even if this is set true, some platforms might require a window to be undecorated
// in order to make the mouse cursor passthrough the window.
//
// SetWindowMousePassthrough works only on desktops.
// SetWindowMousePassthrough does nothing if the platform is not a desktop.
//
// SetWindowMousePassthrough is concurrent-safe.
func SetWindowMousePassthrough(enabled bool) {
	ui.Get().Window().SetMousePassthrough(enabled)
}

// IsWindowMousePassthrough reports whether a mouse cursor passthroughs the window or not on desktops.
//
// IsWindowMousePassthrough always returns false if the platform is not a desktop.
//
// IsWindowMousePassthrough is concurrent-safe.
func IsWindowMousePassthrough() bool {
	return ui.Get().Window().IsMousePassthrough()
}

// RequestAttention requests user attention to the current window and/or the current application.
//
// RequestAttention works only on desktops.
// RequestAttention does nothing if the platform is not a desktop.
//
// RequestAttention is concurrent-safe.
func RequestAttention() {
	ui.Get().Window().RequestAttention()
}
