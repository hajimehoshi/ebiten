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

	"github.com/hajimehoshi/ebiten/internal/loop"
	"github.com/hajimehoshi/ebiten/internal/ui"
)

// FPS represents how many times game updating happens in a second (60).
const FPS = loop.FPS

// CurrentFPS returns the current number of frames per second of rendering.
//
// This function is concurrent-safe.
//
// This value represents how many times rendering happens in 1/60 second and
// NOT how many times logical game updating (a passed function to Run) happens.
// Note that logical game updating is assured to happen 60 times in a second
// as long as the screen is active.
func CurrentFPS() float64 {
	return loop.CurrentFPS()
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
// It is recommended to skip heavy processing, especially drawing, when IsRunningSlowly is true.
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

func (u *updater) Update() error {
	return loop.Update(u.g)
}

func (u *updater) Invalidate() {
	u.g.Invalidate()
}

// Run runs the game.
// f is a function which is called at every frame.
// The argument (*Image) is the render target that represents the screen.
//
// Run must be called from the main thread.
// Note that ebiten bounds the main goroutine to the main OS thread by runtime.LockOSThread.
//
// The given function f is guaranteed to be called 60 times a second
// even if a rendering frame is skipped.
// f is not called when the screen is not shown.
//
// The given scale is ignored on fullscreen mode.
//
// Run returns error when 1) OpenGL error happens, or 2) f returns error.
// In the case of 2), Run returns the same error.
//
// The size unit is device-independent pixel.
func Run(f func(*Image) error, width, height int, scale float64, title string) error {
	ch := make(chan error)
	go func() {
		defer close(ch)

		g := newGraphicsContext(f)
		theGraphicsContext.Store(g)
		if err := loop.Start(); err != nil {
			ch <- err
			return
		}
		if err := run(width, height, scale, title, g); err != nil {
			ch <- err
			return
		}
		loop.End()
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
//
// The size unit is device-independent pixel.
func RunWithoutMainLoop(f func(*Image) error, width, height int, scale float64, title string) <-chan error {
	ch := make(chan error)
	go func() {
		defer close(ch)

		g := newGraphicsContext(f)
		theGraphicsContext.Store(g)
		if err := loop.Start(); err != nil {
			ch <- err
			return
		}
		if err := run(width, height, scale, title, g); err != nil {
			ch <- err
			return
		}
		loop.End()
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

// SetCursorVisibility changes the state of cursor visiblity.
//
// This function is concurrent-safe.
func SetCursorVisibility(visible bool) {
	ui.SetCursorVisibility(visible)
}

// IsScreen returns a boolean value indicating whether
// the current mode is fullscreen or not.
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
// SetFullscreen doesn't work on mobiles.
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
// SetRunnableInBackground doesn't work on mobiles so far.
//
// This function is concurrent-safe.
func SetRunnableInBackground(runnableInBackground bool) {
	ui.SetRunnableInBackground(runnableInBackground)
}
