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
	uiDriver().SetWindowDecorated(decorated)
}

// IsWindowDecorated reports whether the window is decorated.
//
// IsWindowDecorated is concurrent-safe.
func IsWindowDecorated() bool {
	return uiDriver().IsWindowDecorated()
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
// setWindowResizable panics if setWindowResizable is called after Run.
//
// setWindowResizable is concurrent-safe.
func setWindowResizable(resizable bool) {
	uiDriver().SetWindowResizable(resizable)
}

// IsWindowResizable reports whether the window is resizable.
//
// IsWindowResizable is concurrent-safe.
func IsWindowResizable() bool {
	return uiDriver().IsWindowResizable()
}

// SetWindowTitle sets the title of the window.
//
// SetWindowTitle does nothing on mobiles.
//
// SetWindowTitle is concurrent-safe.
func SetWindowTitle(title string) {
	uiDriver().SetWindowTitle(title)
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
	uiDriver().SetWindowIcon(iconImages)
}

// WindowPosition returns the window position.
//
// WindowPosition panics before Run is called.
//
// WindowPosition returns (0, 0) on browsers and mobiles.
//
// WindowPosition is concurrent-safe.
func WindowPosition() (x, y int) {
	return uiDriver().WindowPosition()
}

// SetWindowPosition sets the window position.
//
// SetWindowPosition works before and after Run is called.
//
// SetWindowPosition does nothing on browsers and mobiles.
//
// SetWindowPosition is concurrent-safe.
func SetWindowPosition(x, y int) {
	uiDriver().SetWindowPosition(x, y)
}
