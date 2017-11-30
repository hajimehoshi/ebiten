// Copyright 2014 Hajime Hoshi
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

	"github.com/hajimehoshi/ebiten/internal/audiobinding"
	"github.com/hajimehoshi/ebiten/internal/clock"
	"github.com/hajimehoshi/ebiten/internal/ui"
)

// FPS represents how many times game updating happens in a second (60).
const FPS = clock.FPS

// CurrentFPS returns the current number of frames per second of rendering.
//
// The returned value represents how many times rendering happens in a second and
// NOT how many times logical game updating (a passed function to Run) happens.
// Note that logical game updating is assured to happen 60 times in a second.
//
// This function is concurrent-safe.
func CurrentFPS() float64 {
	return clock.CurrentFPS()
}

var (
	isRunningSlowly = int32(0)
)

func setRunningSlowly(slow bool) {
	v := int32(0)
	if slow {
		v = 1
	}
	atomic.StoreInt32(&isRunningSlowly, v)
}

// IsRunningSlowly returns true if the game is running too slowly to keep 60 FPS of rendering.
// The game screen is not updated when IsRunningSlowly is true.
// It is recommended to skip heavy processing, especially drawing screen,
// when IsRunningSlowly is true.
//
// This function is concurrent-safe.
func IsRunningSlowly() bool {
	return atomic.LoadInt32(&isRunningSlowly) != 0
}

var theGraphicsContext atomic.Value

func run(width, height int, scale float64, title string, g *graphicsContext) error {
	if err := ui.Run(width, height, scale, title, &updater{g}); err != nil {
		if _, ok := err.(*ui.RegularTermination); ok {
			return nil
		}
		return err
	}
	return nil
}

type updater struct {
	g *graphicsContext
}

func (u *updater) SetSize(width, height int, scale float64) {
	u.g.SetSize(width, height, scale)
}

func (u *updater) Update(afterFrameUpdate func()) error {
	select {
	case err := <-audiobinding.Error():
		return err
	default:
	}
	n := clock.Update()
	if err := u.g.Update(n, afterFrameUpdate); err != nil {
		return err
	}
	return nil
}

func (u *updater) Invalidate() {
	u.g.Invalidate()
}

// Run runs the game.
// f is a function which is called at every frame.
// The argument (*Image) is the render target that represents the screen.
// The screen size is based on the given values (width and height).
//
// A window size is based on the given values (width, height and scale).
// scale is used to enlarge the screen.
//
// Run must be called from the OS main thread.
// Note that Ebiten bounds the main goroutine to the main OS thread by runtime.LockOSThread.
//
// The given function f is guaranteed to be called 60 times a second
// even if a rendering frame is skipped.
// f is not called when the window is in background by default.
// This setting is configurable with SetRunnableInBackground.
//
// The given scale is ignored on fullscreen mode.
//
// Run returns error when 1) OpenGL error happens, 2) audio error happens or 3) f returns error.
// In the case of 3), Run returns the same error.
//
// The size unit is device-independent pixel.
//
// Don't call Run twice or more in one process.
func Run(f func(*Image) error, width, height int, scale float64, title string) error {
	ch := make(chan error)
	go func() {
		defer close(ch)

		g := newGraphicsContext(f)
		theGraphicsContext.Store(g)
		if err := run(width, height, scale, title, g); err != nil {
			ch <- err
			return
		}
	}()
	// TODO: Use context in Go 1.7?
	if err := ui.RunMainThreadLoop(ch); err != nil {
		return err
	}
	return nil
}

// RunWithoutMainLoop runs the game, but don't call the loop on the main (UI) thread.
// Different from Run, this function returns immediately.
//
// Typically, Ebiten users don't have to call this directly.
// Instead, functions in github.com/hajimehoshi/ebiten/mobile module call this.
func RunWithoutMainLoop(f func(*Image) error, width, height int, scale float64, title string) <-chan error {
	ch := make(chan error)
	go func() {
		defer close(ch)

		g := newGraphicsContext(f)
		theGraphicsContext.Store(g)
		if err := run(width, height, scale, title, g); err != nil {
			ch <- err
			return
		}
	}()
	return ch
}

// SetScreenSize changes the (logical) size of the screen.
// This doesn't affect the current scale of the screen.
//
// Unit is device-independent pixel.
//
// This function is concurrent-safe.
func SetScreenSize(width, height int) {
	if width <= 0 || height <= 0 {
		panic("ebiten: width and height must be positive")
	}
	ui.SetScreenSize(width, height)
}

// SetScreenScale changes the scale of the screen.
//
// This function is concurrent-safe.
func SetScreenScale(scale float64) {
	if scale <= 0 {
		panic("ebiten: scale must be positive")
	}
	ui.SetScreenScale(scale)
}

// ScreenScale returns the current screen scale.
//
// If Run is not called, this returns 0.
//
// This function is concurrent-safe.
func ScreenScale() float64 {
	return ui.ScreenScale()
}

// IsCursorVisible returns a boolean value indicating whether
// the cursor is visible or not.
//
// IsCursorVisible always returns false on mobiles.
//
// This function is concurrent-safe.
func IsCursorVisible() bool {
	return ui.IsCursorVisible()
}

// SetCursorVisibility changes the state of cursor visiblity.
//
// SetCursorVisibility does nothing on mobiles.
//
// This function is concurrent-safe.
func SetCursorVisibility(visible bool) {
	ui.SetCursorVisibility(visible)
}

// IsFullscreen returns a boolean value indicating whether
// the current mode is fullscreen or not.
//
// IsFullscreen always returns false on mobiles.
//
// This function is concurrent-safe.
func IsFullscreen() bool {
	return ui.IsFullscreen()
}

// SetFullscreen changes the current mode to fullscreen or not.
//
// On fullscreen mode, the game screen is automatically enlarged
// to fit with the monitor. The current scale value is ignored.
//
// On desktops, Ebiten uses 'windowed' fullscreen mode, which doesn't change
// your monitor's resolution.
//
// On browsers, the game screen is resized to fit with the body element (client) size.
// Additionally, the game screen is automatically resized when the body element is resized.
//
// SetFullscreen does nothing on mobiles.
//
// This function is concurrent-safe.
func SetFullscreen(fullscreen bool) {
	ui.SetFullscreen(fullscreen)
}

// IsRunnableInBackground returns a boolean value indicating whether the game runs even in background.
//
// This function is concurrent-safe.
func IsRunnableInBackground() bool {
	return ui.IsRunnableInBackground()
}

// SetRunnableInBackground sets the state if the game runs even in background.
//
// If the given value is true, the game runs in background e.g. when losing focus.
// The initial state is false.
//
// Known issue: On browsers, even if the state is on, the game doesn't run in background tabs.
// This is because browsers throttles background tabs not to often update.
//
// SetRunnableInBackground does nothing on mobiles so far.
//
// This function is concurrent-safe.
func SetRunnableInBackground(runnableInBackground bool) {
	ui.SetRunnableInBackground(runnableInBackground)
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
// This function is concurrent-safe.
func SetWindowIcon(iconImages []image.Image) {
	ui.SetWindowIcon(iconImages)
}
