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
	"time"

	"github.com/hajimehoshi/ebiten/internal/ui"
)

var runContext = &struct {
	running         bool
	fps             float64
	newScreenWidth  int
	newScreenHeight int
	newScreenScale  int
	isRunningSlowly bool
}{}

// CurrentFPS returns the current number of frames per second.
func CurrentFPS() float64 {
	return runContext.fps
}

// IsRunningSlowly returns true if the game is running too slowly to keep 60 FPS.
// The game screen is not updated when IsRunningSlowly is true.
// It is recommended to skip heavy processing, especially drawing, when IsRunningSlowly is true.
func IsRunningSlowly() bool {
	return runContext.isRunningSlowly
}

// Run runs the game.
// f is a function which is called at every frame.
// The argument (*Image) is the render target that represents the screen.
//
// This function must be called from the main thread.
//
// The given function f is guaranteed to be called 60 times a second
// even if a rendering frame is skipped.
// f is not called when the screen is not shown.
func Run(f func(*Image) error, width, height, scale int, title string) error {
	runContext.running = true
	defer func() {
		runContext.running = false
	}()

	if err := ui.Start(width, height, scale, title); err != nil {
		return err
	}
	defer ui.Terminate()

	glContext.Check()
	graphicsContext, err := newGraphicsContext(width, height, ui.ActualScale())
	if err != nil {
		return err
	}

	frames := 0
	now := ui.Now()
	beforeForUpdate := now
	beforeForFPS := now
	for {
		// TODO: setSize should be called after swapping buffers?
		if 0 < runContext.newScreenWidth || 0 < runContext.newScreenHeight || 0 < runContext.newScreenScale {
			changed := false
			if 0 < runContext.newScreenWidth || 0 < runContext.newScreenHeight {
				c := ui.SetScreenSize(runContext.newScreenWidth, runContext.newScreenHeight)
				changed = changed || c
			}
			if 0 < runContext.newScreenScale {
				c := ui.SetScreenScale(runContext.newScreenScale)
				changed = changed || c
			}
			if changed {
				w, h := runContext.newScreenWidth, runContext.newScreenHeight
				if err := graphicsContext.setSize(w, h, ui.ActualScale()); err != nil {
					return err
				}
			}
		}
		runContext.newScreenWidth = 0
		runContext.newScreenHeight = 0
		runContext.newScreenScale = 0

		if err := ui.DoEvents(); err != nil {
			return err
		}
		if ui.IsClosed() {
			return nil
		}
		now := ui.Now()
		// If beforeForUpdate is too old, we assume that screen is not shown.
		runContext.isRunningSlowly = false
		if int64(5*time.Second/60) < now-beforeForUpdate {
			beforeForUpdate = now
		} else {
			c := int((now - beforeForUpdate) * 60 / int64(time.Second))
			runContext.isRunningSlowly = c >= 2
			for i := 0; i < c; i++ {
				if err := ui.DoEvents(); err != nil {
					return err
				}
				if ui.IsClosed() {
					return nil
				}
				if err := graphicsContext.update(f); err != nil {
					return err
				}
			}
			beforeForUpdate += int64(c) * int64(time.Second/60)
			ui.SwapBuffers()
		}

		// Calc the current FPS.
		now = ui.Now()
		frames++
		if time.Second <= time.Duration(now-beforeForFPS) {
			runContext.fps = float64(frames) * float64(time.Second) / float64(now-beforeForFPS)
			beforeForFPS = now
			frames = 0
		}
	}
}

// SetScreenSize changes the (logical) size of the screen.
// This doesn't affect the current scale of the screen.
func SetScreenSize(width, height int) {
	if !runContext.running {
		panic("SetScreenSize must be called during Run")
	}
	if width <= 0 || height <= 0 {
		panic("width and height must be positive")
	}
	runContext.newScreenWidth = width
	runContext.newScreenHeight = height
}

// SetScreenSize changes the scale of the screen.
func SetScreenScale(scale int) {
	if !runContext.running {
		panic("SetScreenScale must be called during Run")
	}
	if scale <= 0 {
		panic("scale must be positive")
	}
	runContext.newScreenScale = scale
}
