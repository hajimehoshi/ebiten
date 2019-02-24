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

	"github.com/hajimehoshi/ebiten/internal/clock"
	"github.com/hajimehoshi/ebiten/internal/ui"
	"github.com/hajimehoshi/ebiten/internal/web"
)

var _ = __EBITEN_REQUIRES_GO_VERSION_1_11_OR_LATER__

// TPS represents a default ticks per second, that represents how many times game updating happens in a second.
const DefaultTPS = 60

// FPS is deprecated as of 1.8.0-alpha: Use DefaultTPS instead.
const FPS = DefaultTPS

// CurrentFPS returns the current number of FPS (frames per second), that represents
// how many swapping buffer happens per second.
//
// On some environments, CurrentFPS doesn't return a reliable value since vsync doesn't work well there.
// If you want to measure the application's speed, Use CurrentTPS.
//
// CurrentFPS is concurrent-safe.
func CurrentFPS() float64 {
	return clock.CurrentFPS()
}

var (
	isDrawingSkipped = int32(0)
	currentMaxTPS    = int32(DefaultTPS)
	isRunning        = int32(0)
)

func setDrawingSkipped(skipped bool) {
	v := int32(0)
	if skipped {
		v = 1
	}
	atomic.StoreInt32(&isDrawingSkipped, v)
}

// IsDrawingSkipped returns true if rendering result is not adopted.
// It is recommended to skip drawing images or screen
// when IsDrawingSkipped is true.
//
// The typical code with IsDrawingSkipped is this:
//
//    func update(screen *ebiten.Image) error {
//
//        // Update the state.
//
//        // When IsDrawingSkipped is true, the rendered result is not adopted.
//        // Skip rendering then.
//        if ebiten.IsDrawingSkipped() {
//            return nil
//        }
//
//        // Draw something to the screen.
//
//        return nil
//    }
//
// IsDrawingSkipped is concurrent-safe.
func IsDrawingSkipped() bool {
	return atomic.LoadInt32(&isDrawingSkipped) != 0
}

// IsRunningSlowly is deprecated as of 1.8.0-alpha.
// Use IsDrawingSkipped instead.
func IsRunningSlowly() bool {
	return IsDrawingSkipped()
}

var theGraphicsContext atomic.Value

func run(width, height int, scale float64, title string, g *graphicsContext, mainloop bool) error {
	atomic.StoreInt32(&isRunning, 1)
	// On GopherJS, run returns immediately.
	if !web.IsGopherJS() {
		defer atomic.StoreInt32(&isRunning, 0)
	}
	if err := ui.Run(width, height, scale, title, g, mainloop); err != nil {
		if err == ui.RegularTermination {
			return nil
		}
		return err
	}
	return nil
}

// Run runs the game.
// f is a function which is called at every frame.
// The argument (*Image) is the render target that represents the screen.
// The screen size is based on the given values (width and height).
//
// A window size is based on the given values (width, height and scale).
// scale is used to enlarge the screen.
// Note that the actual screen is multiplied not only by the given scale but also
// by the device scale on high-DPI display.
// If you pass inverse of the device scale,
// you can disable this automatical device scaling as a result.
// You can get the device scale by DeviceScaleFactor function.
//
// Run must be called on the main thread.
// Note that Ebiten bounds the main goroutine to the main OS thread by runtime.LockOSThread.
//
// Ebiten tries to call f 60 times a second by default. In other words,
// TPS (ticks per second) is 60 by default.
// This is not related to framerate (display's refresh rate).
//
// f is not called when the window is in background by default.
// This setting is configurable with SetRunnableInBackground.
//
// The given scale is ignored on fullscreen mode or gomobile-build mode.
//
// On non-GopherJS environments, Run returns error when 1) OpenGL error happens, 2) audio error happens or
// 3) f returns error. In the case of 3), Run returns the same error.
//
// On GopherJS, Run returns immediately.
// It is because the 'main' goroutine cannot be blocked on GopherJS due to the bug (gopherjs/gopherjs#826).
// When an error happens, this is shown as an error on the console.
//
// The size unit is device-independent pixel.
//
// Don't call Run twice or more in one process.
func Run(f func(*Image) error, width, height int, scale float64, title string) error {
	f = (&imageDumper{f: f}).update

	ch := make(chan error)
	go func() {
		defer close(ch)

		g := newGraphicsContext(f)
		theGraphicsContext.Store(g)
		if err := run(width, height, scale, title, g, true); err != nil {
			ch <- err
			return
		}
	}()
	// TODO: Use context in Go 1.7?
	if err := ui.Loop(ch); err != nil {
		return err
	}
	return nil
}

// RunWithoutMainLoop runs the game, but don't call the loop on the main (UI) thread.
// Different from Run, RunWithoutMainLoop returns immediately.
//
// Ebiten users should NOT call RunWithoutMainLoop.
// Instead, functions in github.com/hajimehoshi/ebiten/mobile package calls this.
func RunWithoutMainLoop(f func(*Image) error, width, height int, scale float64, title string) <-chan error {
	f = (&imageDumper{f: f}).update

	ch := make(chan error)
	go func() {
		defer close(ch)

		g := newGraphicsContext(f)
		theGraphicsContext.Store(g)
		if err := run(width, height, scale, title, g, false); err != nil {
			ch <- err
			return
		}
	}()
	return ch
}

// ScreenSizeInFullscreen returns the size in device-independent pixels when the game is fullscreen.
// The adopted monitor is the 'current' monitor which the window belongs to.
// The returned value can be given to Run or SetSize function if the perfectly fit fullscreen is needed.
//
// On browsers, ScreenSizeInFullscreen returns the 'window' (global object) size, not 'screen' size since an Ebiten game
// should not know the outside of the window object.
// For more details, see SetFullscreen API comment.
//
// On mobiles, ScreenSizeInFullscreen returns (0, 0) so far.
//
// If you use this for screen size with SetFullscreen(true), you can get the fullscreen mode
// which size is well adjusted with the monitor.
//
//     w, h := ScreenSizeInFullscreen()
//     ebiten.SetFullscreen(true)
//     ebiten.Run(update, w, h, 1, "title")
//
// Furthermore, you can use them with DeviceScaleFactor(), you can get the finest
// fullscreen mode.
//
//     s := ebiten.DeviceScaleFactor()
//     w, h := ScreenSizeInFullscreen()
//     ebiten.SetFullscreen(true)
//     ebiten.Run(update, int(float64(w) * s), int(float64(h) * s), 1/s, "title")
//
// For actual example, see examples/fullscreen
//
// ScreenSizeInFullscreen must be called on the main thread before ebiten.Run, and is concurrent-safe after ebiten.Run.
func ScreenSizeInFullscreen() (int, int) {
	return ui.ScreenSizeInFullscreen()
}

// MonitorSize is deprecated as of 1.8.0-alpha. Use ScreenSizeInFullscreen instead.
func MonitorSize() (int, int) {
	return ScreenSizeInFullscreen()
}

// SetScreenSize changes the (logical) size of the screen.
// This doesn't affect the current scale of the screen.
//
// Unit is device-independent pixel.
//
// SetScreenSize is concurrent-safe.
func SetScreenSize(width, height int) {
	if width <= 0 || height <= 0 {
		panic("ebiten: width and height must be positive")
	}
	ui.SetScreenSize(width, height)
}

// SetScreenScale changes the scale of the screen.
//
// Note that the actual screen is multiplied not only by the given scale but also
// by the device scale on high-DPI display.
// If you pass inverse of the device scale,
// you can disable this automatical device scaling as a result.
// You can get the device scale by DeviceScaleFactor function.
//
// SetScreenScale is concurrent-safe.
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
// ScreenScale is concurrent-safe.
func ScreenScale() float64 {
	return ui.ScreenScale()
}

// IsCursorVisible returns a boolean value indicating whether
// the cursor is visible or not.
//
// IsCursorVisible always returns false on mobiles.
//
// IsCursorVisible is concurrent-safe.
func IsCursorVisible() bool {
	return ui.IsCursorVisible()
}

// SetCursorVisible changes the state of cursor visiblity.
//
// SetCursorVisible does nothing on mobiles.
//
// SetCursorVisible is concurrent-safe.
func SetCursorVisible(visible bool) {
	ui.SetCursorVisible(visible)
}

// SetCursorVisibility is deprecated as of 1.6.0-alpha. Use SetCursorVisible instead.
func SetCursorVisibility(visible bool) {
	SetCursorVisible(visible)
}

// IsFullscreen returns a boolean value indicating whether
// the current mode is fullscreen or not.
//
// IsFullscreen always returns false on mobiles.
//
// IsFullscreen is concurrent-safe.
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
// Note that this has nothing to do with 'screen' which is outside of 'window'.
// It is recommended to put Ebiten game in an iframe, and if you want to make the game 'fullscreen'
// on browsers with Fullscreen API, you can do this by applying the API to the iframe.
//
// SetFullscreen does nothing on mobiles.
//
// SetFullscreen is concurrent-safe.
func SetFullscreen(fullscreen bool) {
	ui.SetFullscreen(fullscreen)
}

// IsRunnableInBackground returns a boolean value indicating whether
// the game runs even in background.
//
// IsRunnableInBackground is concurrent-safe.
func IsRunnableInBackground() bool {
	return ui.IsRunnableInBackground()
}

// SetWindowDecorated sets the state if the window is decorated.
//
// The window is decorated by default.
//
// SetWindowDecorated works only on desktops.
// SetWindowDecorated does nothing on other platforms.
//
// SetWindowDecorated panics if SetWindowDecorated is called after Run.
//
// SetWindowDecorated is concurrent-safe.
func SetWindowDecorated(decorated bool) {
	ui.SetWindowDecorated(decorated)
}

// IsWindowDecorated reports whether the window is decorated.
//
// IsWindowDecorated is concurrent-safe.
func IsWindowDecorated() bool {
	return ui.IsWindowDecorated()
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
	ui.SetWindowResizable(resizable)
}

// IsWindowResizable reports whether the window is resizable.
//
// IsWindowResizable is concurrent-safe.
func IsWindowResizable() bool {
	return ui.IsWindowResizable()
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
// SetRunnableInBackground is concurrent-safe.
func SetRunnableInBackground(runnableInBackground bool) {
	ui.SetRunnableInBackground(runnableInBackground)
}

// SetWindowTitle sets the title of the window.
//
// SetWindowTitle does nothing on mobiles.
//
// SetWindowTitle is concurrent-safe.
func SetWindowTitle(title string) {
	ui.SetWindowTitle(title)
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
	ui.SetWindowIcon(iconImages)
}

// DeviceScaleFactor returns a device scale factor value of the current monitor which the window belongs to.
//
// DeviceScaleFactor returns a meaningful value on high-DPI display environment,
// otherwise DeviceScaleFactor returns 1.
//
// DeviceScaleFactor might panic on init function on some devices like Android.
// Then, it is not recommended to call DeviceScaleFactor from init functions.
//
// DeviceScaleFactor must be called on the main thread before ebiten.Run, and is concurrent-safe after ebiten.Run.
func DeviceScaleFactor() float64 {
	return ui.DeviceScaleFactor()
}

// IsVsyncEnabled returns a boolean value indicating whether
// the game uses the display's vsync.
//
// IsVsyncEnabled is concurrent-safe.
func IsVsyncEnabled() bool {
	return ui.IsVsyncEnabled()
}

// SetVsyncEnabled sets a boolean value indicating whether
// the game uses the display's vsync.
//
// If the given value is true, the game tries to sync the display's refresh rate.
// If false, the game ignores the display's refresh rate.
// The initial value is true.
// By disabling vsync, the game works more efficiently but consumes more CPU.
//
// Note that the state doesn't affect TPS (ticks per second, i.e. how many the run function is
// updated per second).
//
// SetVsyncEnabled does nothing on mobiles so far.
//
// SetVsyncEnabled is concurrent-safe.
func SetVsyncEnabled(enabled bool) {
	ui.SetVsyncEnabled(enabled)
}

// MaxTPS returns the current maximum TPS.
//
// MaxTPS is concurrent-safe.
func MaxTPS() int {
	return int(atomic.LoadInt32(&currentMaxTPS))
}

// CurrentTPS returns the current TPS (ticks per second),
// that represents how many update function is called in a second.
//
// CurrentTPS is concurrent-safe.
func CurrentTPS() float64 {
	return clock.CurrentTPS()
}

// UncappedTPS is a special TPS value that means the game doesn't have limitation on TPS.
const UncappedTPS = clock.UncappedTPS

// SetMaxTPS sets the maximum TPS (ticks per second),
// that represents how many updating function is called per second.
// The initial value is 60.
//
// If tps is UncappedTPS, TPS is uncapped and the game is updated per frame.
// If tps is negative but not UncappedTPS, SetMaxTPS panics.
//
// SetMaxTPS is concurrent-safe.
func SetMaxTPS(tps int) {
	if tps < 0 && tps != UncappedTPS {
		panic("ebiten: tps must be >= 0 or UncappedTPS")
	}
	atomic.StoreInt32(&currentMaxTPS, int32(tps))
}
