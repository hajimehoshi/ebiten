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
	"github.com/hajimehoshi/ebiten/v2/internal/ui"
)

// Game defines necessary functions for a game.
type Game interface {
	// Update updates a game by one tick. The given argument represents a screen image.
	//
	// Update updates only the game logic and Draw draws the screen.
	//
	// You can assume that Update is always called TPS-times per second (60 by default), and you can assume
	// that the time delta between two Updates is always 1 / TPS [s] (1/60[s] by default). As Ebitengine already
	// adjusts the number of Update calls, you don't have to measure time deltas in Update by e.g. OS timers.
	//
	// An actual TPS is available by ActualTPS(), and the result might slightly differ from your expected TPS,
	// but still, your game logic should stick to the fixed time delta and should not rely on ActualTPS() value.
	// This API is for just measurement and/or debugging. In the long run, the number of Update calls should be
	// adjusted based on the set TPS on average.
	//
	// An actual time delta between two Updates might be bigger than expected. In this case, your game's
	// Update or Draw takes longer than they should. In this case, there is nothing other than optimizing
	// your game implementation.
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
	//
	// The frequency of Draw calls depends on the user's environment, especially the monitors refresh rate.
	// For portability, you should not put your game logic in Draw in general.
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
const DefaultTPS = clock.DefaultTPS

// ActualFPS returns the current number of FPS (frames per second), that represents
// how many swapping buffer happens per second.
//
// On some environments, ActualFPS doesn't return a reliable value since vsync doesn't work well there.
// If you want to measure the application's speed, Use ActualTPS.
//
// This value is for measurement and/or debug, and your game logic should not rely on this value.
//
// ActualFPS is concurrent-safe.
func ActualFPS() float64 {
	return clock.ActualFPS()
}

// CurrentFPS returns the current number of FPS (frames per second), that represents
// how many swapping buffer happens per second.
//
// Deprecated: as of v2.4. Use ActualFPS instead.
func CurrentFPS() float64 {
	return ActualFPS()
}

var (
	isRunGameEnded_ = int32(0)
)

// SetScreenClearedEveryFrame enables or disables the clearing of the screen at the beginning of each frame.
// The default value is true and the screen is cleared each frame by default.
//
// SetScreenClearedEveryFrame is concurrent-safe.
func SetScreenClearedEveryFrame(cleared bool) {
	ui.SetScreenClearedEveryFrame(cleared)
}

// IsScreenClearedEveryFrame returns true if the frame isn't cleared at the beginning.
//
// IsScreenClearedEveryFrame is concurrent-safe.
func IsScreenClearedEveryFrame() bool {
	return ui.IsScreenClearedEveryFrame()
}

// SetScreenFilterEnabled enables/disables the use of the "screen" filter Ebitengine uses.
//
// The "screen" filter is a box filter from game to display resolution.
//
// If disabled, nearest-neighbor filtering will be used for scaling instead.
//
// The default state is true.
//
// SetScreenFilterEnabled is concurrent-safe, but takes effect only at the next Draw call.
func SetScreenFilterEnabled(enabled bool) {
	ui.SetScreenFilterEnabled(enabled)
}

// IsScreenFilterEnabled returns true if Ebitengine's "screen" filter is enabled.
//
// IsScreenFilterEnabled is concurrent-safe.
func IsScreenFilterEnabled() bool {
	return ui.IsScreenFilterEnabled()
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
// game's Draw function is called every frame to draw the screen.
// game's Layout function is called when necessary, and you can specify the logical screen size by the function.
//
// game's functions are called on the same goroutine.
//
// On browsers, it is strongly recommended to use iframe if you embed an Ebitengine application in your website.
//
// RunGame must be called on the main thread.
// Note that Ebitengine bounds the main goroutine to the main OS thread by runtime.LockOSThread.
//
// Ebitengine tries to call game's Update function 60 times a second by default. In other words,
// TPS (ticks per second) is 60 by default.
// This is not related to framerate (display's refresh rate).
//
// RunGame returns error when 1) an error happens in the underlying graphics driver, 2) an audio error happens
// or 3) f returns an error. In the case of 3), RunGame returns the same error so far, but it is recommended to
// use errors.Is when you check the returned error is the error you want, rather than comparing the values
// with == or != directly.
//
// If you want to terminate a game on desktops, it is totally fine to define your own error value, return it at
// Update, and check whether the returned error value from RunGame is the same as the value you defined.
//
// The size unit is device-independent pixel.
//
// Don't call RunGame twice or more in one process.
func RunGame(game Game) error {
	defer atomic.StoreInt32(&isRunGameEnded_, 1)

	initializeWindowPositionIfNeeded(WindowSize())
	g := newGameForUI(&imageDumperGame{
		game: game,
	})
	if err := ui.Get().Run(g); err != nil {
		if err == ui.RegularTermination {
			return nil
		}
		return err
	}
	return nil
}

func isRunGameEnded() bool {
	return atomic.LoadInt32(&isRunGameEnded_) != 0
}

// ScreenSizeInFullscreen returns the size in device-independent pixels when the game is fullscreen.
// The adopted monitor is the 'current' monitor which the window belongs to.
// The returned value can be given to Run or SetSize function if the perfectly fit fullscreen is needed.
//
// On browsers, ScreenSizeInFullscreen returns the 'window' (global object) size, not 'screen' size.
// ScreenSizeInFullscreen's returning value is different from the actual screen size and this is a known issue (#2145).
// For browsers, it is recommended to use Screen API (https://developer.mozilla.org/en-US/docs/Web/API/Screen) if needed.
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
	return ui.Get().ScreenSizeInFullscreen()
}

// CursorMode returns the current cursor mode.
//
// CursorMode returns CursorModeHidden on mobiles.
//
// CursorMode is concurrent-safe.
func CursorMode() CursorModeType {
	return ui.Get().CursorMode()
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
	ui.Get().SetCursorMode(mode)
}

// CursorShape returns the current cursor shape.
//
// CursorShape returns CursorShapeDefault on mobiles.
//
// CursorShape is concurrent-safe.
func CursorShape() CursorShapeType {
	return ui.Get().CursorShape()
}

// SetCursorShape sets the cursor shape.
//
// SetCursorShape is concurrent-safe.
func SetCursorShape(shape CursorShapeType) {
	ui.Get().SetCursorShape(shape)
}

// IsFullscreen reports whether the current mode is fullscreen or not.
//
// IsFullscreen always returns false on mobiles.
//
// IsFullscreen is concurrent-safe.
func IsFullscreen() bool {
	return ui.Get().IsFullscreen()
}

// SetFullscreen changes the current mode to fullscreen or not on desktops and browsers.
//
// In fullscreen mode, the game screen is automatically enlarged
// to fit with the monitor. The current scale value is ignored.
//
// On desktops, Ebitengine uses 'windowed' fullscreen mode, which doesn't change
// your monitor's resolution.
//
// On browsers, triggering fullscreen requires a user gesture otherwise SetFullscreen does nothing but leave an error message in console.
// This behaviour varies across browser implementations, your mileage may vary.
//
// SetFullscreen does nothing on mobiles.
//
// SetFullscreen does nothing on macOS when the window is fullscreened natively by the macOS desktop
// instead of SetFullscreen(true).
//
// SetFullscreen is concurrent-safe.
func SetFullscreen(fullscreen bool) {
	ui.Get().SetFullscreen(fullscreen)
}

// IsFocused returns a boolean value indicating whether
// the game is in focus or in the foreground.
//
// IsFocused will only return true if IsRunnableOnUnfocused is false.
//
// IsFocused is concurrent-safe.
func IsFocused() bool {
	return ui.Get().IsFocused()
}

// IsRunnableOnUnfocused returns a boolean value indicating whether
// the game runs even in background.
//
// IsRunnableOnUnfocused is concurrent-safe.
func IsRunnableOnUnfocused() bool {
	return ui.Get().IsRunnableOnUnfocused()
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
	ui.Get().SetRunnableOnUnfocused(runnableOnUnfocused)
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
	return ui.Get().DeviceScaleFactor()
}

// IsVsyncEnabled returns a boolean value indicating whether
// the game uses the display's vsync.
//
// Deprecated: as of v2.2. Use FPSMode instead.
func IsVsyncEnabled() bool {
	return ui.FPSMode() == ui.FPSModeVsyncOn
}

// SetVsyncEnabled sets a boolean value indicating whether
// the game uses the display's vsync.
//
// Deprecated: as of v2.2. Use SetFPSMode instead.
func SetVsyncEnabled(enabled bool) {
	if enabled {
		ui.SetFPSMode(ui.FPSModeVsyncOn)
	} else {
		ui.SetFPSMode(ui.FPSModeVsyncOffMaximum)
	}
}

// FPSModeType is a type of FPS modes.
type FPSModeType = ui.FPSModeType

const (
	// FPSModeVsyncOn indicates that the game tries to sync the display's refresh rate.
	// FPSModeVsyncOn is the default mode.
	FPSModeVsyncOn FPSModeType = ui.FPSModeVsyncOn

	// FPSModeVsyncOffMaximum indicates that the game doesn't sync with vsync, and
	// the game is updated whenever possible.
	//
	// Be careful that FPSModeVsyncOffMaximum might consume a lot of battery power.
	//
	// In FPSModeVsyncOffMaximum, the game's Draw is called almost without sleeping.
	// The game's Update is called based on the specified TPS.
	FPSModeVsyncOffMaximum FPSModeType = ui.FPSModeVsyncOffMaximum

	// FPSModeVsyncOffMinimum indicates that the game doesn't sync with vsync, and
	// the game is updated only when necessary.
	//
	// FPSModeVsyncOffMinimum is useful for relatively static applications to save battery power.
	//
	// In FPSModeVsyncOffMinimum, the game's Update and Draw are called only when
	// 1) new inputting except for gamepads is detected, or 2) ScheduleFrame is called.
	// In FPSModeVsyncOffMinimum, TPS is SyncWithFPS no matter what TPS is specified at SetTPS.
	FPSModeVsyncOffMinimum FPSModeType = ui.FPSModeVsyncOffMinimum
)

// FPSMode returns the current FPS mode.
//
// FPSMode is concurrent-safe.
func FPSMode() FPSModeType {
	return ui.FPSMode()
}

// SetFPSMode sets the FPS mode.
// The default FPS mode is FPSModeVsyncOn.
//
// SetFPSMode is concurrent-safe.
func SetFPSMode(mode FPSModeType) {
	ui.SetFPSMode(mode)
}

// ScheduleFrame schedules a next frame when the current FPS mode is FPSModeVsyncOffMinimum.
//
// ScheduleFrame is concurrent-safe.
func ScheduleFrame() {
	ui.Get().ScheduleFrame()
}

// TPS returns the current maximum TPS.
//
// TPS is concurrent-safe.
func TPS() int {
	return clock.TPS()
}

// MaxTPS returns the current maximum TPS.
//
// Deprecated: as of v2.4. Use TPS instead.
func MaxTPS() int {
	return TPS()
}

// ActualTPS returns the current TPS (ticks per second),
// that represents how many Update function is called in a second.
//
// This value is for measurement and/or debug, and your game logic should not rely on this value.
//
// ActualTPS is concurrent-safe.
func ActualTPS() float64 {
	return clock.ActualTPS()
}

// CurrentTPS returns the current TPS (ticks per second),
// that represents how many Update function is called in a second.
//
// Deprecated: as of v2.4. Use ActualTPS instead.
func CurrentTPS() float64 {
	return ActualTPS()
}

// SyncWithFPS is a special TPS value that means TPS syncs with FPS.
const SyncWithFPS = clock.SyncWithFPS

// UncappedTPS is a special TPS value that means TPS syncs with FPS.
//
// Deprecated: as of v2.2. Use SyncWithFPS instead.
const UncappedTPS = SyncWithFPS

// SetTPS sets the maximum TPS (ticks per second),
// that represents how many updating function is called per second.
// The initial value is 60.
//
// If tps is SyncWithFPS, TPS is uncapped and the game is updated per frame.
// If tps is negative but not SyncWithFPS, SetTPS panics.
//
// SetTPS is concurrent-safe.
func SetTPS(tps int) {
	clock.SetTPS(tps)
}

// SetMaxTPS sets the maximum TPS (ticks per second),
// that represents how many updating function is called per second.
//
// Deprecated: as of v2.4. Use SetTPS instead.
func SetMaxTPS(tps int) {
	SetTPS(tps)
}

// IsScreenTransparent reports whether the window is transparent.
//
// IsScreenTransparent is concurrent-safe.
func IsScreenTransparent() bool {
	return ui.Get().IsScreenTransparent()
}

// SetScreenTransparent sets the state if the window is transparent.
//
// SetScreenTransparent panics if SetScreenTransparent is called after the main loop.
//
// SetScreenTransparent does nothing on mobiles.
//
// SetScreenTransparent is concurrent-safe.
func SetScreenTransparent(transparent bool) {
	ui.Get().SetScreenTransparent(transparent)
}

// SetInitFocused sets whether the application is focused on show.
// The default value is true, i.e., the application is focused.
// Note that the application does not proceed if this is not focused by default.
// This behavior can be changed by SetRunnableOnUnfocused.
//
// SetInitFocused does nothing on mobile.
//
// SetInitFocused panics if this is called after the main loop.
//
// SetInitFocused is cuncurrent-safe.
func SetInitFocused(focused bool) {
	ui.Get().SetInitFocused(focused)
}
