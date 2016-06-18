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
	"github.com/hajimehoshi/ebiten/internal/loop"
	"github.com/hajimehoshi/ebiten/internal/ui"
)

// FPS represents how many times game updating happens in a second.
const FPS = 60

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

// IsRunningSlowly returns true if the game is running too slowly to keep 60 FPS of rendering.
// The game screen is not updated when IsRunningSlowly is true.
// It is recommended to skip heavy processing, especially drawing, when IsRunningSlowly is true.
//
// This function is concurrent-safe.
func IsRunningSlowly() bool {
	return loop.IsRunningSlowly()
}

// Run runs the game.
// f is a function which is called at every frame.
// The argument (*Image) is the render target that represents the screen.
//
// This function must be called from the main thread.
// Note that ebiten bounds the main goroutine to the main OS thread by runtime.LockOSThread.
//
// The given function f is guaranteed to be called 60 times a second
// even if a rendering frame is skipped.
// f is not called when the screen is not shown.
func Run(f func(*Image) error, width, height int, scale float64, title string) error {
	ch := make(chan error)
	go func() {
		g := newGraphicsContext(f)
		ch <- loop.Run(g, width, height, scale, title, FPS)
	}()
	ui.Main()
	return <-ch
}

// RunWithoutMainLoop runs the game, but don't call the loop on the main (UI) thread.
// Different from Run, this function returns immediately.
//
// Typically, Ebiten users don't have to call this directly.
// Instead, functions in github.com/hajimehoshi/ebiten/mobile module call this.
func RunWithoutMainLoop(f func(*Image) error, width, height int, scale float64, title string) <-chan error {
	ch := make(chan error)
	go func() {
		g := newGraphicsContext(f)
		if err := loop.Run(g, width, height, scale, title, FPS); err != nil {
			ch <- err
		}
		close(ch)
	}()
	return ch
}

// SetScreenSize changes the (logical) size of the screen.
// This doesn't affect the current scale of the screen.
//
// This function is concurrent-safe.
func SetScreenSize(width, height int) {
	if width <= 0 || height <= 0 {
		panic("ebiten: width and height must be positive")
	}
	ui.CurrentUI().SetScreenSize(width, height)
}

// SetScreenScale changes the scale of the screen.
//
// This function is concurrent-safe.
func SetScreenScale(scale float64) {
	if scale <= 0 {
		panic("ebiten: scale must be positive")
	}
	ui.CurrentUI().SetScreenScale(scale)
}

// ScreenScale returns the current screen scale.
//
// This function is concurrent-safe.
func ScreenScale() float64 {
	return ui.CurrentUI().ScreenScale()
}
