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
	"errors"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/internal/ui"
)

type runContext struct {
	isRunning       bool
	fps             float64
	newScreenWidth  int
	newScreenHeight int
	newScreenScale  int
	isRunningSlowly bool
	m               sync.RWMutex
}

var currentRunContext runContext

func (c *runContext) startRunning() {
	c.m.Lock()
	defer c.m.Unlock()
	c.isRunning = true
}

func (c *runContext) endRunning() {
	c.m.Lock()
	defer c.m.Unlock()
	c.isRunning = false
}

func (c *runContext) FPS() float64 {
	c.m.RLock()
	defer c.m.RUnlock()
	if !c.isRunning {
		// TODO: Should panic here?
		return 0
	}
	return c.fps
}

func (c *runContext) updateFPS(fps float64) {
	c.m.Lock()
	defer c.m.Unlock()
	c.fps = fps
}

func (c *runContext) IsRunningSlowly() bool {
	c.m.RLock()
	defer c.m.RUnlock()
	if !c.isRunning {
		// TODO: Should panic here?
		return false
	}
	return c.isRunningSlowly
}

func (c *runContext) setRunningSlowly(isRunningSlowly bool) {
	c.m.Lock()
	defer c.m.Unlock()
	c.isRunningSlowly = isRunningSlowly
}

func (c *runContext) updateScreenSize(g *graphicsContext) error {
	c.m.Lock()
	defer c.m.Unlock()
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

func (c *runContext) SetScreenSize(width, height int) error {
	c.m.Lock()
	defer c.m.Unlock()
	if !c.isRunning {
		return errors.New("ebiten: SetScreenSize must be called during Run")
	}
	if width <= 0 || height <= 0 {
		return errors.New("ebiten: width and height must be positive")
	}
	c.newScreenWidth = width
	c.newScreenHeight = height
	return nil
}

func (c *runContext) SetScreenScale(scale int) error {
	c.m.Lock()
	defer c.m.Unlock()
	if !c.isRunning {
		return errors.New("ebiten: SetScreenScale must be called during Run")
	}
	if scale <= 0 {
		return errors.New("ebiten: scale must be positive")
	}
	c.newScreenScale = scale
	return nil
}

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
	return currentRunContext.FPS()
}

// IsRunningSlowly returns true if the game is running too slowly to keep 60 FPS of rendering.
// The game screen is not updated when IsRunningSlowly is true.
// It is recommended to skip heavy processing, especially drawing, when IsRunningSlowly is true.
//
// This function is concurrent-safe.
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
	ch := make(chan error)
	go func() {
		ch <- run(f, width, height, scale, title)
	}()
	ui.Main()
	return <-ch
}

func run(f func(*Image) error, width, height, scale int, title string) error {
	currentRunContext.startRunning()
	defer currentRunContext.endRunning()

	if err := ui.CurrentUI().Start(width, height, scale, title); err != nil {
		return err
	}
	defer ui.CurrentUI().Terminate()

	graphicsContext, err := newGraphicsContext(width, height, ui.CurrentUI().ActualScreenScale())
	if err != nil {
		return err
	}

	frames := 0
	n := ui.Now()
	beforeForUpdate := n
	beforeForFPS := n
	for {
		// TODO: setSize should be called after swapping buffers?
		if err := currentRunContext.updateScreenSize(graphicsContext); err != nil {
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
		if int64(5*time.Second/FPS) < now-beforeForUpdate {
			currentRunContext.setRunningSlowly(false)
			beforeForUpdate = now
		} else {
			// Note that generally t is a little different from 1/60[sec].
			t := now - beforeForUpdate
			currentRunContext.setRunningSlowly(t*FPS >= int64(time.Second*5/2))
			tt := int(t * FPS / int64(time.Second))
			// As t is not accurate 1/60[sec], errors are accumulated.
			// To make the FPS stable, set tt 1 if t is a little less than 1/60[sec].
			if tt == 0 && (int64(time.Second)/FPS-int64(5*time.Millisecond)) < t {
				tt = 1
			}
			for i := 0; i < tt; i++ {
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
			ui.CurrentUI().SwapBuffers()
			beforeForUpdate += int64(tt) * int64(time.Second) / FPS
			frames++
		}

		// Calc the current FPS.
		if time.Second <= time.Duration(now-beforeForFPS) {
			currentRunContext.updateFPS(float64(frames) * float64(time.Second) / float64(now-beforeForFPS))
			beforeForFPS = now
			frames = 0
		}
	}
}

// SetScreenSize changes the (logical) size of the screen.
// This doesn't affect the current scale of the screen.
//
// This function is concurrent-safe.
func SetScreenSize(width, height int) {
	if err := currentRunContext.SetScreenSize(width, height); err != nil {
		panic(err)
	}
}

// SetScreenScale changes the scale of the screen.
//
// This function is concurrent-safe.
func SetScreenScale(scale int) {
	if err := currentRunContext.SetScreenScale(scale); err != nil {
		panic(err)
	}
}

// ScreenScale returns the current screen scale.
//
// This function is concurrent-safe.
func ScreenScale() int {
	return ui.CurrentUI().ScreenScale()
}
