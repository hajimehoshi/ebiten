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
	lastUpdated    int64
	lastFPSUpdated int64
	m              sync.RWMutex

	lastAudioFrame     int64
	lastAudioFrameTime int64
	deltaTime          int64
	targetDeltaTime    int64
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

	n := currentRunContext.adjustedNowWithAudio()
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

func (c *runContext) adjustedNowWithAudio() int64 {
	n := now()
	if audio.CurrentContext() == nil {
		return n
	}
	if c.lastAudioFrameTime == 0 {
		c.lastAudioFrameTime = n
	}
	if f := audio.CurrentContext().Frame(); c.lastAudioFrame != f {
		c.targetDeltaTime += (f-c.lastAudioFrame)*int64(time.Second)/audio.FPS - (n - c.lastAudioFrameTime)
		c.lastAudioFrame = f
		c.lastAudioFrameTime = n
	}
	switch {
	case c.deltaTime > c.targetDeltaTime:
		c.deltaTime -= int64(time.Millisecond)
		if c.deltaTime < c.targetDeltaTime {
			c.deltaTime = c.targetDeltaTime
		}
	case c.deltaTime < c.targetDeltaTime:
		c.deltaTime += int64(time.Millisecond)
		if c.deltaTime > c.targetDeltaTime {
			c.deltaTime = c.targetDeltaTime
		}
	}
	return n + c.deltaTime
}

func (c *runContext) render(g GraphicsContext) error {
	fps := c.fps
	clockN := now()
	n := c.adjustedNowWithAudio()
	defer func() {
		// Calc the current FPS.
		if time.Second > time.Duration(clockN-c.lastFPSUpdated) {
			return
		}
		currentFPS := float64(c.frames) * float64(time.Second) / float64(clockN-c.lastFPSUpdated)
		c.updateFPS(currentFPS)
		c.lastFPSUpdated = clockN
		c.frames = 0
	}()

	// If lastUpdated is too old, we assume that screen is not shown.
	if 10*int64(time.Second)/int64(fps) < n-c.lastUpdated {
		c.lastUpdated = n
		return nil
	}
	if audio.CurrentContext() != nil {
		audio.CurrentContext().Ping()
	}

	// Note that generally t is a little different from 1/60[sec].
	t := n - c.lastUpdated
	if t < 0 {
		return nil
	}
	tt := int(t * int64(fps) / int64(time.Second))

	// As t is not accurate 1/60[sec], errors are accumulated.
	// To make the FPS stable, set tt 1 if t is a little less than 1/60[sec].
	if tt == 0 && (int64(time.Second)/int64(fps)-int64(5*time.Millisecond)) < t {
		tt = 1
	}
	if err := g.UpdateAndDraw(tt); err != nil {
		return err
	}
	c.lastUpdated += int64(tt) * int64(time.Second) / int64(fps)
	c.frames++
	return nil
}
