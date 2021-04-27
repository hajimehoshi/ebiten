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
	"sync/atomic"

	"github.com/hajimehoshi/ebiten/v2/internal/clock"
	"github.com/hajimehoshi/ebiten/v2/internal/driver"
)

// Game defines necessary functions for a game.
type Game interface {
	// Update updates a game by one tick. The given argument represents a screen image.
	//
	// Update updates only the game logic and Draw draws the screen.
	//
	// In the first frame, it is ensured that Update is called at least once before Draw. You can use Update
	// to initialize the game state.
	//
	// After the first frame, Update might not be called or might be called once
	// or more for one frame. The frequency is determined by the current TPS (tick-per-second).
	Update() error

	// Draw draws the game screen by one frame.
	//
	// The give argument represents a screen image. The updated content is adopted as the game screen.
	Draw(screen *Image)

	// Layout accepts a native outside size in device-independent pixels and returns the game's logical screen
	// size.
	//
	// On desktops, the outside is a window or a monitor (fullscreen mode). On browsers, the outside is a body
	// element. On mobiles, the outside is the view's size.
	//
	// Even though the outside size and the screen size differ, the rendering scale is automatically adjusted to
	// fit with the outside.
	//
	// Layout is called almost every frame.
	//
	// It is ensured that Layout is invoked before Update is called in the first frame.
	//
	// If Layout returns non-positive numbers, the caller can panic.
	//
	// You can return a fixed screen size if you don't care, or you can also return a calculated screen size
	// adjusted with the given outside size.
	Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int)
}

// DefaultTPS represents a default ticks per second, that represents how many times game updating happens in a second.
const DefaultTPS = 60

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
	isScreenClearedEveryFrame = int32(1)
	isRunGameEnded_           = int32(0)
	currentMaxTPS             = int32(DefaultTPS)
)

// SetScreenClearedEveryFrame enables or disables the clearing of the screen at the beginning of each frame.
// The default value is true and the screen is cleared each frame by default.
//
// SetScreenClearedEveryFrame is concurrent-safe.
func SetScreenClearedEveryFrame(cleared bool) {
	v := int32(0)
	if cleared {
		v = 1
	}
	atomic.StoreInt32(&isScreenClearedEveryFrame, v)
	theUIContext.setScreenClearedEveryFrame(cleared)
}

// IsScreenClearedEveryFrame returns true if the frame isn't cleared at the beginning.
//
// IsScreenClearedEveryFrame is concurrent-safe.
func IsScreenClearedEveryFrame() bool {
	return atomic.LoadInt32(&isScreenClearedEveryFrame) != 0
}

type imageDumperGame struct {
	game Game
	d    *imageDumper
	err  error
}

func (i *imageDumperGame) Update() error {
	if i.err != nil {
		return i.err
	}
	if i.d == nil {
		i.d = &imageDumper{g: i.game}
	}
	return i.d.update()
}

func (i *imageDumperGame) Draw(screen *Image) {
	if i.err != nil {
		return
	}

	i.game.Draw(screen)
	i.err = i.d.dump(screen)
}

func (i *imageDumperGame) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return i.game.Layout(outsideWidth, outsideHeight)
}

// RunGame starts the main loop and runs the game.
// game's Update function is called every tick to update the game logic.
// game's Draw function is, if it exists, called every frame to draw the screen.
// game's Layout function is called when necessary, and you can specify the logical screen size by the function.
//
// game must implement Game interface.
// Game's Draw function is optional, but it is recommended to implement Draw to seperate updating the logic and
// rendering.
//
// RunGame is a more flexibile form of Run due to game's Layout function.
// You can make a resizable window if you use RunGame, while you cannot if you use Run.
// RunGame is more sophisticated way than Run and hides the notion of 'scale'.
//
// While Run specifies the window size, RunGame does not.
// You need to call SetWindowSize before RunGame if you want.
// Otherwise, a default window size is adopted.
//
// Some functions (ScreenScale, SetScreenScale, SetScreenSize) are not available with RunGame.
//
// On browsers, it is strongly recommended to use iframe if you embed an Ebiten application in your website.
//
// RunGame must be called on the main thread.
// Note that Ebiten bounds the main goroutine to the main OS thread by runtime.LockOSThread.
//
// Ebiten tries to call game's Update function 60 times a second by default. In other words,
// TPS (ticks per second) is 60 by default.
// This is not related to framerate (display's refresh rate).
//
// RunGame returns error when 1) error happens in the underlying graphics driver, 2) audio error happens or
// 3) f returns error. In the case of 3), RunGame returns the same error.
//
// The size unit is device-independent pixel.
//
// Don't call RunGame twice or more in one process.
func RunGame(game Game) error {
	defer atomic.StoreInt32(&isRunGameEnded_, 1)

	initializeWindowPositionIfNeeded(WindowSize())
	theUIContext.set(&imageDumperGame{
		game: game,
	})
	if err := uiDriver().Run(theUIContext); err != nil {
		if err == driver.RegularTermination {
			return nil
		}
		return err
	}
	return nil
}

func isRunGameEnded() bool {
	return atomic.LoadInt32(&isRunGameEnded_) != 0
}

// RunGameWithoutMainLoop runs the game, but doesn't call the loop on the main (UI) thread.
// Different from Run, RunGameWithoutMainLoop returns immediately.
//
// Ebiten users should NOT call RunGameWithoutMainLoop.
// Instead, functions in github.com/hajimehoshi/ebiten/v2/mobile package calls this.
//
// TODO: Remove this. In order to remove this, the uiContext should be in another package.
func RunGameWithoutMainLoop(game Game) {
	initializeWindowPositionIfNeeded(WindowSize())
	theUIContext.set(&imageDumperGame{
		game: game,
	})
	uiDriver().RunWithoutMainLoop(theUIContext)
}

// ScreenSizeInFullscreen returns the size in device-independent pixels when the game is fullscreen.
// The adopted monitor is the 'current' monitor which the window belongs to.
// The returned value can be given to Run or SetSize function if the perfectly fit fullscreen is needed.
//
// On browsers, ScreenSizeInFullscreen returns the 'window' (global object) size, not 'screen' size since an Ebiten
// game should not know the outside of the window object.
//
// On mobiles, ScreenSizeInFullscreen returns (0, 0) so far.
//
// ScreenSizeInFullscreen's use cases are limited. If you are making a fullscreen application, you can use RunGame and
// the Game interface's Layout function instead. If you are making a not-fullscreen application but the application's
// behavior depends on the monitor size, ScreenSizeInFullscreen is useful.
//
// ScreenSizeInFullscreen must be called on the main thread before ebiten.Run, and is concurrent-safe after
// ebiten.Run.
func ScreenSizeInFullscreen() (int, int) {
	return uiDriver().ScreenSizeInFullscreen()
}

// CursorMode returns the current cursor mode.
//
// On browsers, only CursorModeVisible and CursorModeHidden are supported.
//
// CursorMode returns CursorModeHidden on mobiles.
//
// CursorMode is concurrent-safe.
func CursorMode() CursorModeType {
	return CursorModeType(uiDriver().CursorMode())
}

// SetCursorMode sets the render and capture mode of the mouse cursor.
// CursorModeVisible sets the cursor to always be visible.
// CursorModeHidden hides the system cursor when over the window.
// CursorModeCaptured hides the system cursor and locks it to the window.
//
// CursorModeCaptured also works on browsers.
// When the user exits the captured mode not by SetCursorMode but by the UI (e.g., pressing ESC),
// the previous cursor mode is set automatically.
//
// SetCursorMode does nothing on mobiles.
//
// SetCursorMode is concurrent-safe.
func SetCursorMode(mode CursorModeType) {
	uiDriver().SetCursorMode(driver.CursorMode(mode))
}

func CursorShape() CursorShapeType {
	return CursorShapeType(uiDriver().CursorShape())
}

func SetCursorShape(shape CursorShapeType) {
	uiDriver().SetCursorShape(driver.CursorShape(shape))
}

// IsFullscreen reports whether the current mode is fullscreen or not.
//
// IsFullscreen always returns false on browsers or mobiles.
//
// IsFullscreen is concurrent-safe.
func IsFullscreen() bool {
	return uiDriver().IsFullscreen()
}

// SetFullscreen changes the current mode to fullscreen or not on desktops.
//
// In fullscreen mode, the game screen is automatically enlarged
// to fit with the monitor. The current scale value is ignored.
//
// On desktops, Ebiten uses 'windowed' fullscreen mode, which doesn't change
// your monitor's resolution.
//
// SetFullscreen does nothing on browsers or mobiles.
//
// SetFullscreen does nothing on macOS when the window is fullscreened natively by the macOS desktop
// instead of SetFullscreen(true).
//
// SetFullscreen is concurrent-safe.
func SetFullscreen(fullscreen bool) {
	uiDriver().SetFullscreen(fullscreen)
}

// IsFocused returns a boolean value indicating whether
// the game is in focus or in the foreground.
//
// IsFocused will only return true if IsRunnableOnUnfocused is false.
//
// IsFocused is concurrent-safe.
func IsFocused() bool {
	return uiDriver().IsFocused()
}

// IsRunnableOnUnfocused returns a boolean value indicating whether
// the game runs even in background.
//
// IsRunnableOnUnfocused is concurrent-safe.
func IsRunnableOnUnfocused() bool {
	return uiDriver().IsRunnableOnUnfocused()
}

// SetRunnableOnUnfocused sets the state if the game runs even in background.
//
// If the given value is true, the game runs even in background e.g. when losing focus.
// The initial state is true.
//
// Known issue: On browsers, even if the state is on, the game doesn't run in background tabs.
// This is because browsers throttles background tabs not to often update.
//
// SetRunnableOnUnfocused does nothing on mobiles so far.
//
// SetRunnableOnUnfocused is concurrent-safe.
func SetRunnableOnUnfocused(runnableOnUnfocused bool) {
	uiDriver().SetRunnableOnUnfocused(runnableOnUnfocused)
}

// DeviceScaleFactor returns a device scale factor value of the current monitor which the window belongs to.
//
// DeviceScaleFactor returns a meaningful value on high-DPI display environment,
// otherwise DeviceScaleFactor returns 1.
//
// DeviceScaleFactor might panic on init function on some devices like Android.
// Then, it is not recommended to call DeviceScaleFactor from init functions.
//
// DeviceScaleFactor must be called on the main thread before the main loop, and is concurrent-safe after the main
// loop.
//
// DeviceScaleFactor is concurrent-safe.
//
// BUG: DeviceScaleFactor value is not affected by SetWindowPosition before RunGame (#1575).
func DeviceScaleFactor() float64 {
	return uiDriver().DeviceScaleFactor()
}

// IsVsyncEnabled returns a boolean value indicating whether
// the game uses the display's vsync.
//
// IsVsyncEnabled is concurrent-safe.
func IsVsyncEnabled() bool {
	return uiDriver().IsVsyncEnabled()
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
	uiDriver().SetVsyncEnabled(enabled)
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

// IsScreenTransparent reports whether the window is transparent.
//
// IsScreenTransparent is concurrent-safe.
func IsScreenTransparent() bool {
	return uiDriver().IsScreenTransparent()
}

// SetScreenTransparent sets the state if the window is transparent.
//
// SetScreenTransparent panics if SetScreenTransparent is called after the main loop.
//
// SetScreenTransparent does nothing on mobiles.
//
// SetScreenTransparent is concurrent-safe.
func SetScreenTransparent(transparent bool) {
	uiDriver().SetScreenTransparent(transparent)
}

// SetInitFocused sets whether the application is focused on show.
// The default value is true, i.e., the application is focused.
// Note that the application does not proceed if this is not focused by default.
// This behavior can be changed by SetRunnableInBackground.
//
// SetInitFocused does nothing on mobile.
//
// SetInitFocused panics if this is called after the main loop.
//
// SetInitFocused is cuncurrent-safe.
func SetInitFocused(focused bool) {
	uiDriver().SetInitFocused(focused)
}
