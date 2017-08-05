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

	"github.com/hajimehoshi/ebiten/internal/clock"
	"github.com/hajimehoshi/ebiten/internal/sync"
)

const FPS = 60

func CurrentFPS() float64 {
	if theRunContext == nil {
		return 0
	}
	return theRunContext.getCurrentFPS()
}

type runContext struct {
	currentFPS     float64
	runningSlowly  bool
	frames         int64
	framesForFPS   int64
	lastUpdated    int64
	lastFPSUpdated int64
	lastClockFrame int64
	ping           func()
	m              sync.RWMutex
}

var (
	theRunContext *runContext
	contextInitCh = make(chan struct{})
)

func (c *runContext) getCurrentFPS() float64 {
	c.m.RLock()
	v := c.currentFPS
	c.m.RUnlock()
	return v
}

func (c *runContext) updateFPS(fps float64) {
	c.m.Lock()
	c.currentFPS = fps
	c.m.Unlock()
}

func Start() error {
	// TODO: Need lock here?
	if theRunContext != nil {
		return errors.New("loop: The game is already running")
	}
	theRunContext = &runContext{}

	n := now()
	theRunContext.lastUpdated = n
	theRunContext.lastFPSUpdated = n

	close(contextInitCh)
	return nil
}

func End() {
	theRunContext = nil
}

func (c *runContext) updateCount(now int64) int {
	count := 0
	sync := false

	t := now - c.lastUpdated
	if t < 0 {
		return 0
	}

	if clock.IsValid() {
		if c.lastClockFrame != clock.Now() {
			sync = true
			f := clock.Now()
			if c.frames < f {
				count = int(f - c.frames)
			}
			c.lastClockFrame = f
		}
	} else {
		if t > 5*int64(time.Second)/int64(FPS) {
			// The previous time is too old. Let's assume that the window was unfocused.
			count = 0
			sync = true
		} else {
			count = int(t * int64(FPS) / int64(time.Second))
		}
	}

	// Stabilize FPS.
	if count == 0 && (int64(time.Second)/int64(FPS)/2) < t {
		count = 1
	}
	if count == 2 && (int64(time.Second)/int64(FPS)*3/2) > t {
		count = 1
	}

	if count > 3 {
		count = 3
	}

	if sync {
		c.lastUpdated = now
	} else {
		c.lastUpdated += int64(count) * int64(time.Second) / int64(FPS)
	}

	c.frames += int64(count)
	return count
}

func RegisterPing(ping func()) {
	<-contextInitCh
	theRunContext.registerPing(ping)
}

func (c *runContext) registerPing(ping func()) {
	c.m.Lock()
	c.ping = ping
	c.m.Unlock()
}

type Updater interface {
	Update(updateCount int) error
}

func Update(u Updater) error {
	<-contextInitCh
	return theRunContext.update(u)
}

func (c *runContext) update(u Updater) error {
	n := now()

	c.m.Lock()
	if c.ping != nil {
		c.ping()
	}
	c.m.Unlock()

	count := c.updateCount(n)
	if err := u.Update(count); err != nil {
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
