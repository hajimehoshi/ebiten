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
	audio "github.com/hajimehoshi/ebiten/exp/audio/internal"
	"github.com/hajimehoshi/ebiten/internal/ui"
	"time"
)

var runContext = &struct {
	running         bool
	fps             float64
	newScreenWidth  int
	newScreenHeight int
	newScreenScale  int
}{}

// CurrentFPS returns the current number of frames per second.
func CurrentFPS() float64 {
	return runContext.fps
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

	actualScale, err := ui.Start(width, height, scale, title)
	if err != nil {
		return err
	}
	defer ui.Terminate()

	graphicsContext, err := newGraphicsContext(width, height, actualScale)
	if err != nil {
		return err
	}

	frames := 0
	gameTime := ui.Now()
	before := ui.Now()
	for {
		if 0 < runContext.newScreenWidth || 0 < runContext.newScreenHeight || 0 < runContext.newScreenScale {
			changed := false
			actualScale := 0
			if 0 < runContext.newScreenWidth || 0 < runContext.newScreenHeight {
				c, a := ui.SetScreenSize(runContext.newScreenWidth, runContext.newScreenHeight)
				changed = changed || c
				actualScale = a
			}
			if 0 < runContext.newScreenScale {
				c, a := ui.SetScreenScale(runContext.newScreenScale)
				changed = changed || c
				// actualScale of SetScreenState is more reliable than one of SetScreenSize
				actualScale = a
			}
			if changed {
				w, h := runContext.newScreenWidth, runContext.newScreenHeight
				if err := graphicsContext.setSize(w, h, actualScale); err != nil {
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
		// If gameTime is too old, we assume that screen is not shown.
		if int64(5*time.Second/60) < now-gameTime {
			gameTime = now
		}
		for gameTime < now {
			gameTime += int64(time.Second / 60)

			if err := graphicsContext.preUpdate(); err != nil {
				return err
			}
			if err := f(graphicsContext.screen); err != nil {
				return err
			}
			audio.Tick()
		}
		if err := graphicsContext.postUpdate(); err != nil {
			return err
		}

		ui.SwapBuffers()

		// Calc the current FPS.
		now = ui.Now()
		frames++
		if time.Second <= time.Duration(now-before) {
			runContext.fps = float64(frames) * float64(time.Second) / float64(now-before)
			before = now
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
