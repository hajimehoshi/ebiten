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

package loop

import (
	"errors"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/internal/ui"
)

func CurrentFPS() float64 {
	return currentRunContext.getCurrentFPS()
}

func IsRunning() bool {
	return currentRunContext.isRunning()
}

func IsRunningSlowly() bool {
	return currentRunContext.isRunningSlowly()
}

type runContext struct {
	running        bool
	fps            int
	currentFPS     float64
	runningSlowly  bool
	frames         int
	lastUpdated    int64
	lastFPSUpdated int64
	m              sync.RWMutex
}

var currentRunContext *runContext

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

func (c *runContext) getCurrentFPS() float64 {
	c.m.RLock()
	defer c.m.RUnlock()
	if !c.running {
		// TODO: Should panic here?
		return 0
	}
	return c.currentFPS
}

func (c *runContext) updateFPS(fps float64) {
	c.m.Lock()
	defer c.m.Unlock()
	c.currentFPS = fps
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

type GraphicsContext interface {
	SetSize(width, height int, scale float64) error
	UpdateAndDraw() error
	Draw() error
}

func Run(g GraphicsContext, width, height int, scale float64, title string, fps int) error {
	if currentRunContext != nil {
		return errors.New("loop: The game is already running")
	}
	currentRunContext = &runContext{
		fps: fps,
	}
	currentRunContext.startRunning()
	defer currentRunContext.endRunning()

	if err := ui.CurrentUI().Start(width, height, scale, title); err != nil {
		return err
	}
	// TODO: Use the error value
	defer ui.CurrentUI().Terminate()

	n := now()
	currentRunContext.lastUpdated = n
	currentRunContext.lastFPSUpdated = n
	for {
		e, err := ui.CurrentUI().Update()
		if err != nil {
			return err
		}
		switch e := e.(type) {
		case ui.ScreenSizeEvent:
			if err := g.SetSize(e.Width, e.Height, e.ActualScale); err != nil {
				return err
			}
		case ui.CloseEvent:
			return nil
		case ui.RenderEvent:
			if err := currentRunContext.render(g); err != nil {
				return err
			}
			e.Done <- struct{}{}
		default:
			panic("not reach")
		}
	}
}

func (c *runContext) render(g GraphicsContext) error {
	fps := c.fps
	n := now()
	defer func() {
		// Calc the current FPS.
		if time.Second > time.Duration(n-c.lastFPSUpdated) {
			return
		}
		currentFPS := float64(c.frames) * float64(time.Second) / float64(n-c.lastFPSUpdated)
		c.updateFPS(currentFPS)
		c.lastFPSUpdated = n
		c.frames = 0
	}()

	// If lastUpdated is too old, we assume that screen is not shown.
	if 5*int64(time.Second)/int64(fps) < n-c.lastUpdated {
		c.setRunningSlowly(false)
		c.lastUpdated = n
		return nil
	}

	// Note that generally t is a little different from 1/60[sec].
	t := n - c.lastUpdated
	tt := int(t * int64(fps) / int64(time.Second))

	// As t is not accurate 1/60[sec], errors are accumulated.
	// To make the FPS stable, set tt 1 if t is a little less than 1/60[sec].
	if tt == 0 && (int64(time.Second)/int64(fps)-int64(5*time.Millisecond)) < t {
		tt = 1
	}
	if 1 <= tt {
		for i := 0; i < tt; i++ {
			slow := i < tt-1
			c.setRunningSlowly(slow)
			if err := g.UpdateAndDraw(); err != nil {
				return err
			}
		}
	} else {
		if err := g.Draw(); err != nil {
			return err
		}
	}
	if err := ui.CurrentUI().SwapBuffers(); err != nil {
		return err
	}
	c.lastUpdated += int64(tt) * int64(time.Second) / int64(fps)
	c.frames++
	return nil
}
