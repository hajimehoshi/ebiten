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
	"sync"
)

const (
	maxInt     = int(^uint(0) >> 1)
	minInt     = -maxInt - 1
	invalidPos = minInt
)

// SetWindowDecorated sets the state if the window is decorated.
//
// The window is decorated by default.
//
// SetWindowDecorated works only on desktops.
// SetWindowDecorated does nothing on other platforms.
//
// SetWindowDecorated is concurrent-safe.
func SetWindowDecorated(decorated bool) {
	if w := uiDriver().Window(); w != nil {
		w.SetDecorated(decorated)
	}
}

// IsWindowDecorated reports whether the window is decorated.
//
// IsWindowDecorated is concurrent-safe.
func IsWindowDecorated() bool {
	if w := uiDriver().Window(); w != nil {
		return w.IsDecorated()
	}
	return false
}

// setWindowResizable is unexported until specification is determined (#320)
//
// setWindowResizable sets the state if the window is resizable.
//
// The window is not resizable by default.
//
// When the window is resizable, the image size given via the update function can be changed by resizing.
//
// setWindowResizable works only on desktops.
// setWindowResizable does nothing on other platforms.
//
// setWindowResizable panics if setWindowResizable is called after the main loop.
//
// setWindowResizable is concurrent-safe.
func setWindowResizable(resizable bool) {
	if w := uiDriver().Window(); w != nil {
		w.SetResizable(resizable)
	}
}

// IsWindowResizable reports whether the window is resizable by the user's dragging on desktops.
// On the other environments, IsWindowResizable always returns false.
//
// IsWindowResizable is concurrent-safe.
func IsWindowResizable() bool {
	if w := uiDriver().Window(); w != nil {
		return w.IsResizable()
	}
	return false
}

// SetWindowResizable sets whether the window is resizable by the user's dragging on desktops.
// On the other environments, SetWindowResizable does nothing.
//
// If SetWindowResizable is called with true and Run is used, SetWindowResizable panics. Use RunGame instead.
//
// SetWindowResizable is concurrent-safe.
func SetWindowResizable(resizable bool) {
	theUIContext.setWindowResizable(resizable)
}

// SetWindowTitle sets the title of the window.
//
// SetWindowTitle updated the title on browsers, but now does nothing on browsers as of 1.11.0-alpha.
//
// SetWindowTitle does nothing on mobiles.
//
// SetWindowTitle is concurrent-safe.
func SetWindowTitle(title string) {
	if w := uiDriver().Window(); w != nil {
		w.SetTitle(title)
	}
}

// SetWindowIcon sets the icon of the game window.
//
// If len(iconImages) is 0, SetWindowIcon reverts the icon to the default one.
//
// For desktops, see the document of glfwSetWindowIcon of GLFW 3.2:
//
//     This function sets the icon of the specified window.
//     If passed an array of candidate images, those of or closest to the sizes
//     desired by the system are selected.
//     If no images are specified, the window reverts to its default icon.
//
//     The desired image sizes varies depending on platform and system settings.
//     The selected images will be rescaled as needed.
//     Good sizes include 16x16, 32x32 and 48x48.
//
// As macOS windows don't have icons, SetWindowIcon doesn't work on macOS.
//
// SetWindowIcon doesn't work on browsers or mobiles.
//
// SetWindowIcon is concurrent-safe.
func SetWindowIcon(iconImages []image.Image) {
	if w := uiDriver().Window(); w != nil {
		w.SetIcon(iconImages)
	}
}

// WindowPosition returns the window position.
//
// WindowPosition panics if the main loop does not start yet.
//
// WindowPosition returns the last window position on fullscreen mode.
//
// WindowPosition returns (0, 0) on browsers and mobiles.
//
// WindowPosition is concurrent-safe.
func WindowPosition() (x, y int) {
	if x, y, ok := initWindowPosition(); ok {
		return x, y
	}
	if w := uiDriver().Window(); w != nil {
		return w.Position()
	}
	return 0, 0
}

// SetWindowPosition sets the window position.
//
// SetWindowPosition does nothing on fullscreen mode.
//
// SetWindowPosition does nothing on browsers and mobiles.
//
// SetWindowPosition is concurrent-safe.
func SetWindowPosition(x, y int) {
	if setInitWindowPosition(x, y) {
		return
	}
	if w := uiDriver().Window(); w != nil {
		w.SetPosition(x, y)
	}
}

var (
	windowM             sync.Mutex
	mainLoopStarted     bool
	initWindowPositionX = invalidPos
	initWindowPositionY = invalidPos
)

func initWindowPosition() (x, y int, ok bool) {
	windowM.Lock()
	defer windowM.Unlock()
	if mainLoopStarted {
		return 0, 0, false
	}
	if initWindowPositionX == invalidPos || initWindowPositionY == invalidPos {
		return 0, 0, false
	}
	return initWindowPositionX, initWindowPositionY, true
}

func setInitWindowPosition(x, y int) bool {
	windowM.Lock()
	defer windowM.Unlock()
	if mainLoopStarted {
		return false
	}
	initWindowPositionX, initWindowPositionY = x, y
	return true
}

func fixWindowPosition(width, height int) {
	windowM.Lock()
	defer windowM.Unlock()

	defer func() {
		mainLoopStarted = true
	}()

	w := uiDriver().Window()
	if w == nil {
		return
	}

	if initWindowPositionX == invalidPos || initWindowPositionY == invalidPos {
		mx, my := uiDriver().MonitorPosition()
		sw, sh := uiDriver().ScreenSizeInFullscreen()
		x := mx + (sw-width)/2
		y := my + (sh-height)/3
		w.SetPosition(x, y)
	} else {
		w.SetPosition(initWindowPositionX, initWindowPositionY)
	}
}

// WindowSize returns the window size on desktops.
// WindowSize returns (0, 0) on other environments.
//
// On fullscreen mode, WindowSize returns the original window size.
//
// WindowSize is concurrent-safe.
func WindowSize() (int, int) {
	if w := uiDriver().Window(); w != nil {
		return w.Size()
	}
	return 0, 0
}

// SetWindowSize sets the window size on desktops.
// SetWindowSize does nothing on other environments.
//
// On fullscreen mode, SetWindowSize sets the original window size.
//
// SetWindowSize panics if width or height is not a positive number.
//
// SetWindowSize is concurrent-safe.
func SetWindowSize(width, height int) {
	if width <= 0 || height <= 0 {
		panic("ebiten: width and height must be positive")
	}
	if w := uiDriver().Window(); w != nil {
		w.SetSize(width, height)
	}
}
