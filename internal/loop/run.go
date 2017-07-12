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
	"time"

	"github.com/hajimehoshi/ebiten/audio"
	"github.com/hajimehoshi/ebiten/internal/sync"
	"github.com/hajimehoshi/ebiten/internal/ui"
)

func CurrentFPS() float64 {
	return currentRunContext.getCurrentFPS()
}

type runContext struct {
	running        bool
	fps            int
	currentFPS     float64
	runningSlowly  bool
	frames         int64
	framesForFPS   int64
	lastUpdated    int64
	lastFPSUpdated int64
	lastAudioFrame int64
	m              sync.RWMutex
}

var currentRunContext *runContext

func (c *runContext) startRunning() {
	c.m.Lock()
	defer c.m.Unlock()
	c.running = true
}

func (c *runContext) isRunning() bool {
	c.m.RLock()
	defer c.m.RUnlock()
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

type GraphicsContext interface {
	SetSize(width, height int, scale float64)
	UpdateAndDraw(updateCount int) error
	Invalidate()
}

type loopGraphicsContext struct {
	runContext      *runContext
	graphicsContext GraphicsContext
}

func (g *loopGraphicsContext) SetSize(width, height int, scale float64) {
	g.graphicsContext.SetSize(width, height, scale)
}

func (g *loopGraphicsContext) Update() error {
	return g.runContext.render(g.graphicsContext)
}

func (g *loopGraphicsContext) Invalidate() {
	g.graphicsContext.Invalidate()
}

func Run(g GraphicsContext, width, height int, scale float64, title string, fps int) (err error) {
	if currentRunContext != nil {
		return errors.New("loop: The game is already running")
	}
	currentRunContext = &runContext{
		fps: fps,
	}
	currentRunContext.startRunning()
	defer currentRunContext.endRunning()

	n := now()
	currentRunContext.lastUpdated = n
	currentRunContext.lastFPSUpdated = n

	lg := &loopGraphicsContext{currentRunContext, g}
	if err := ui.Run(width, height, scale, title, lg); err != nil {
		if _, ok := err.(*ui.RegularTermination); ok {
			return nil
		}
		return err
	}
	return nil
}

func (c *runContext) updateCount(now int64) int {
	count := 0
	sync := false

	t := now - c.lastUpdated
	if t < 0 {
		return 0
	}

	if audio.CurrentContext() != nil && c.lastAudioFrame != audio.CurrentContext().Frame() {
		sync = true
		f := audio.CurrentContext().Frame()
		if c.frames < f {
			count = int(f - c.frames)
		}
		c.lastAudioFrame = f
	} else {
		count = int(t * int64(c.fps) / int64(time.Second))
	}

	// Stabilize FPS.
	if count == 0 && (int64(time.Second)/int64(c.fps)/2) < t {
		count = 1
	}
	if count == 2 && (int64(time.Second)/int64(c.fps)*3/2) > t {
		count = 1
	}

	if count > 3 {
		count = 3
	}

	if sync {
		c.lastUpdated = now
	} else {
		c.lastUpdated += int64(count) * int64(time.Second) / int64(c.fps)
	}

	c.frames += int64(count)
	return count
}

func (c *runContext) render(g GraphicsContext) error {
	n := now()

	if audio.CurrentContext() != nil {
		audio.CurrentContext().Ping()
	}

	count := c.updateCount(n)
	if err := g.UpdateAndDraw(count); err != nil {
		return err
	}
	c.framesForFPS++

	// Calc the current FPS.
	if time.Second > time.Duration(n-c.lastFPSUpdated) {
		return nil
	}
	currentFPS := float64(c.framesForFPS) * float64(time.Second) / float64(n-c.lastFPSUpdated)
	c.updateFPS(currentFPS)
	c.lastFPSUpdated = n
	c.framesForFPS = 0

	return nil
}
