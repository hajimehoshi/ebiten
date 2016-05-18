// Copyright 2016 Hajime Hoshi
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

package ui

import (
	"errors"
	"sync"
	"time"
)

const FPS = 60

func CurrentFPS() float64 {
	return currentRunContext.currentFPS()
}

func IsRunning() bool {
	return currentRunContext.isRunning()
}

func IsRunningSlowly() bool {
	return currentRunContext.isRunningSlowly()
}

func SetScreenSize(width, height int) error {
	return currentRunContext.setScreenSize(width, height)
}

func SetScreenScale(scale int) error {
	return currentRunContext.setScreenScale(scale)
}

type runContext struct {
	running         bool
	fps             float64
	newScreenWidth  int
	newScreenHeight int
	newScreenScale  int
	runningSlowly   bool
	m               sync.RWMutex
}

var currentRunContext runContext

func (c *runContext) startRunning() {
	c.m.Lock()
	defer c.m.Unlock()
	c.running = true
}

func (c *runContext) isRunning() bool {
	c.m.Lock()
	defer c.m.Unlock()
	return c.running
}

func (c *runContext) endRunning() {
	c.m.Lock()
	defer c.m.Unlock()
	c.running = false
}

func (c *runContext) currentFPS() float64 {
	c.m.RLock()
	defer c.m.RUnlock()
	if !c.running {
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

func (c *runContext) isRunningSlowly() bool {
	c.m.RLock()
	defer c.m.RUnlock()
	if !c.running {
		// TODO: Should panic here?
		return false
	}
	return c.runningSlowly
}

func (c *runContext) setRunningSlowly(isRunningSlowly bool) {
	c.m.Lock()
	defer c.m.Unlock()
	c.runningSlowly = isRunningSlowly
}

func (c *runContext) updateScreenSize(g GraphicsContext) error {
	c.m.Lock()
	defer c.m.Unlock()
	if c.newScreenWidth == 0 && c.newScreenHeight == 0 && c.newScreenScale == 0 {
		return nil
	}
	changed := false
	if 0 < c.newScreenWidth || 0 < c.newScreenHeight {
		c := CurrentUI().SetScreenSize(c.newScreenWidth, c.newScreenHeight)
		changed = changed || c
	}
	if 0 < c.newScreenScale {
		c := CurrentUI().SetScreenScale(c.newScreenScale)
		changed = changed || c
	}
	if changed {
		w, h := c.newScreenWidth, c.newScreenHeight
		if err := g.SetSize(w, h, CurrentUI().ActualScreenScale()); err != nil {
			return err
		}
	}
	c.newScreenWidth = 0
	c.newScreenHeight = 0
	c.newScreenScale = 0
	return nil
}

func (c *runContext) setScreenSize(width, height int) error {
	c.m.Lock()
	defer c.m.Unlock()
	if !c.running {
		return errors.New("ebiten: SetScreenSize must be called during Run")
	}
	if width <= 0 || height <= 0 {
		return errors.New("ebiten: width and height must be positive")
	}
	c.newScreenWidth = width
	c.newScreenHeight = height
	return nil
}

func (c *runContext) setScreenScale(scale int) error {
	c.m.Lock()
	defer c.m.Unlock()
	if !c.running {
		return errors.New("ebiten: SetScreenScale must be called during Run")
	}
	if scale <= 0 {
		return errors.New("ebiten: scale must be positive")
	}
	c.newScreenScale = scale
	return nil
}

type GraphicsContext interface {
	SetSize(width, height, scale int) error
	Update() error
}

func Run(g GraphicsContext, width, height, scale int, title string) error {
	currentRunContext.startRunning()
	defer currentRunContext.endRunning()

	if err := CurrentUI().Start(width, height, scale, title); err != nil {
		return err
	}
	defer CurrentUI().Terminate()

	if err := g.SetSize(width, height, CurrentUI().ActualScreenScale()); err != nil {
		return err
	}

	frames := 0
	n := Now()
	beforeForUpdate := n
	beforeForFPS := n
	for {
		if err := currentRunContext.updateScreenSize(g); err != nil {
			return err
		}
		e, err := CurrentUI().Update()
		if err != nil {
			return err
		}
		switch e.(type) {
		case CloseEvent:
			return nil
		case RenderEvent:
			now := Now()
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
					if err := g.Update(); err != nil {
						return err
					}
				}
				CurrentUI().SwapBuffers()
				beforeForUpdate += int64(tt) * int64(time.Second) / FPS
				frames++
			}

			// Calc the current FPS.
			if time.Second <= time.Duration(now-beforeForFPS) {
				currentRunContext.updateFPS(float64(frames) * float64(time.Second) / float64(now-beforeForFPS))
				beforeForFPS = now
				frames = 0
			}
		default:
			panic("not reach")
		}
	}
}
