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

	"github.com/hajimehoshi/ebiten/v2/internal/ui"
)

// WindowResizingModeType represents a mode in which a user resizes the window.
//
// Regardless of the resizing mode, an Ebiten application can still change the window size or make
// the window fullscreen by calling Ebiten functions.
type WindowResizingModeType = ui.WindowResizingMode

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
// SetWindowDecorated does nothing on other platforms.
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
// SetWindowTitle does nothing on browsers or mobiles.
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
// SetWindowIcon doesn't work on browsers or mobiles.
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
// WindowPosition returns (0, 0) on browsers and mobiles.
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
// SetWindowPosition does nothing on browsers and mobiles.
//
// SetWindowPosition is concurrent-safe.
func SetWindowPosition(x, y int) {
	atomic.StoreUint32(&windowPositionSetExplicitly, 1)
	ui.Get().Window().SetPosition(x, y)
}

var (
	windowPositionSetExplicitly uint32
)

func initializeWindowPositionIfNeeded(width, height int) {
	if atomic.LoadUint32(&windowPositionSetExplicitly) == 0 {
		sw, sh := ui.Get().ScreenSizeInFullscreen()
		x := (sw - width) / 2
		y := (sh - height) / 3
		ui.Get().Window().SetPosition(x, y)
	}
}

// WindowSize returns the window size on desktops.
// WindowSize returns (0, 0) on other environments.
//
// WindowSize returns the original window size in fullscreen mode.
//
// WindowSize is concurrent-safe.
func WindowSize() (int, int) {
	return ui.Get().Window().Size()
}

// SetWindowSize sets the window size on desktops.
// SetWindowSize does nothing on other environments.
//
// SetWindowSize sets the original window size in fullscreen mode.
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
// IsWindowFloating returns false on browsers and mobiles.
//
// IsWindowFloating is concurrent-safe.
func IsWindowFloating() bool {
	return ui.Get().Window().IsFloating()
}

// SetWindowFloating sets the state whether the window is always shown above all the other windows.
//
// SetWindowFloating does nothing on browsers or mobiles.
//
// SetWindowFloating is concurrent-safe.
func SetWindowFloating(float bool) {
	ui.Get().Window().SetFloating(float)
}

// MaximizeWindow maximizes the window.
//
// MaximizeWindow does nothing when the window is not resizable (WindowResizingModeEnabled).
//
// MaximizeWindow does nothing on browsers or mobiles.
//
// MaximizeWindow is concurrent-safe.
func MaximizeWindow() {
	ui.Get().Window().Maximize()
}

// IsWindowMaximized reports whether the window is maximized or not.
//
// IsWindowMaximized returns false when the window is not resizable (WindowResizingModeEnabled).
//
// IsWindowMaximized always returns false on browsers and mobiles.
//
// IsWindowMaximized is concurrent-safe.
func IsWindowMaximized() bool {
	return ui.Get().Window().IsMaximized()
}

// MinimizeWindow minimizes the window.
//
// If the main loop does not start yet, MinimizeWindow does nothing.
//
// MinimizeWindow does nothing on browsers or mobiles.
//
// MinimizeWindow is concurrent-safe.
func MinimizeWindow() {
	ui.Get().Window().Minimize()
}

// IsWindowMinimized reports whether the window is minimized or not.
//
// IsWindowMinimized always returns false on browsers and mobiles.
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
// IsWindowBeingClosed always returns false on other platforms.
//
// IsWindowBeingClosed is concurrent-safe.
func IsWindowBeingClosed() bool {
	return ui.Get().Window().IsBeingClosed()
}

// SetWindowClosingHandled sets whether the window closing is handled or not on desktops. The default state is false.
//
// If the window closing is handled, the window is not closed immediately and
// the game can know whether the window is begin closed or not by IsWindowBeingClosed.
// In this case, the window is not closed automatically.
// To end the game, you have to return an error value at the Game's Update function.
//
// SetWindowClosingHandled works only on desktops.
// SetWindowClosingHandled does nothing on other platforms.
//
// SetWindowClosingHandled is concurrent-safe.
func SetWindowClosingHandled(handled bool) {
	ui.Get().Window().SetClosingHandled(handled)
}

// IsWindowClosingHandled reports whether the window closing is handled or not on desktops by SetWindowClosingHandled.
//
// IsWindowClosingHandled always returns false on other platforms.
//
// IsWindowClosingHandled is concurrent-safe.
func IsWindowClosingHandled() bool {
	return ui.Get().Window().IsClosingHandled()
}
