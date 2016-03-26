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

type runContext struct {
	fps             float64
	newScreenWidth  int
	newScreenHeight int
	newScreenScale  int
	isRunningSlowly bool
}

var currentRunContext = &runContext{}

func (c *runContext) CurrentFPS() float64 {
	if c == nil {
		// TODO: Should panic here?
		return 0
	}
	return c.fps
}

func (c *runContext) IsRunningSlowly() bool {
	if c == nil {
		// TODO: Should panic here?
		return false
	}
	return c.isRunningSlowly
}

func (c *runContext) updateScreenSize(g *graphicsContext) error {
	if c.newScreenWidth == 0 && c.newScreenHeight == 0 && c.newScreenScale == 0 {
		return nil
	}
	changed := false
	if 0 < c.newScreenWidth || 0 < c.newScreenHeight {
		c := ui.CurrentUI().SetScreenSize(c.newScreenWidth, c.newScreenHeight)
		changed = changed || c
	}
	if 0 < c.newScreenScale {
		c := ui.CurrentUI().SetScreenScale(c.newScreenScale)
		changed = changed || c
	}
	if changed {
		w, h := c.newScreenWidth, c.newScreenHeight
		if err := g.setSize(w, h, ui.CurrentUI().ActualScreenScale()); err != nil {
			return err
		}
	}
	c.newScreenWidth = 0
	c.newScreenHeight = 0
	c.newScreenScale = 0
	return nil
}

func (c *runContext) Run(f func(*Image) error, width, height, scale int, title string) error {
	if err := ui.CurrentUI().Start(width, height, scale, title); err != nil {
		return err
	}
	defer ui.CurrentUI().Terminate()

	glContext.Check()
	graphicsContext, err := newGraphicsContext(width, height, ui.CurrentUI().ActualScreenScale())
	if err != nil {
		return err
	}

	frames := 0
	now := ui.Now()
	beforeForUpdate := now
	beforeForFPS := now
	for {
		// TODO: setSize should be called after swapping buffers?
		if err := c.updateScreenSize(graphicsContext); err != nil {
			return err
		}
		if err := ui.CurrentUI().DoEvents(); err != nil {
			return err
		}
		if ui.CurrentUI().IsClosed() {
			return nil
		}
		now := ui.Now()
		// If beforeForUpdate is too old, we assume that screen is not shown.
		c.isRunningSlowly = false
		if int64(5*time.Second/FPS) < now-beforeForUpdate {
			beforeForUpdate = now
		} else {
			t := float64(now-beforeForUpdate) * FPS / float64(time.Second)
			c.isRunningSlowly = t >= 2.5
			for i := 0; i < int(t); i++ {
				if err := ui.CurrentUI().DoEvents(); err != nil {
					return err
				}
				if ui.CurrentUI().IsClosed() {
					return nil
				}
				if err := graphicsContext.update(f); err != nil {
					return err
				}
			}
			beforeForUpdate += int64(t) * int64(time.Second/FPS)
			ui.CurrentUI().SwapBuffers()
		}

		// Calc the current FPS.
		now = ui.Now()
		frames++
		if time.Second <= time.Duration(now-beforeForFPS) {
			c.fps = float64(frames) * float64(time.Second) / float64(now-beforeForFPS)
			beforeForFPS = now
			frames = 0
		}
	}
}

func (c *runContext) SetScreenSize(width, height int) {
	if c == nil {
		panic("ebiten: SetScreenSize must be called during Run")
	}
	if width <= 0 || height <= 0 {
		panic("ebiten: width and height must be positive")
	}
	c.newScreenWidth = width
	c.newScreenHeight = height
}

func (c *runContext) SetScreenScale(scale int) {
	if c == nil {
		panic("ebiten: SetScreenScale must be called during Run")
	}
	if scale <= 0 {
		panic("ebiten: scale must be positive")
	}
	c.newScreenScale = scale
}

// FPS represents how many times game updating happens in a second.
const FPS = 60

// CurrentFPS returns the current number of frames per second of rendering.
//
// This value represents how many times rendering happens in 1/60 second and
// NOT how many times logical game updating (a passed function to Run) happens.
// Note that logical game updating is assured to happen 60 times in a second
// as long as the screen is active.
func CurrentFPS() float64 {
	return currentRunContext.CurrentFPS()
}

// IsRunningSlowly returns true if the game is running too slowly to keep 60 FPS of rendering.
// The game screen is not updated when IsRunningSlowly is true.
// It is recommended to skip heavy processing, especially drawing, when IsRunningSlowly is true.
func IsRunningSlowly() bool {
	return currentRunContext.IsRunningSlowly()
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
func Run(f func(*Image) error, width, height, scale int, title string) error {
	currentRunContext = &runContext{}
	defer func() {
		currentRunContext = nil
	}()
	return currentRunContext.Run(f, width, height, scale, title)
}

// SetScreenSize changes the (logical) size of the screen.
// This doesn't affect the current scale of the screen.
func SetScreenSize(width, height int) {
	currentRunContext.SetScreenSize(width, height)
}

// SetScreenSize changes the scale of the screen.
func SetScreenScale(scale int) {
	currentRunContext.SetScreenScale(scale)
}

// ScreenScale returns the current screen scale.
func ScreenScale() int {
	return ui.CurrentUI().ScreenScale()
}
