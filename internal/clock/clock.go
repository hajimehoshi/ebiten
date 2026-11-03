// Copyright 2017 The Ebiten Authors
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

// Package clock manages game timers.
package clock

import (
	"sync"
	"sync/atomic"
	"time"
)

const (
	DefaultTPS  = 60
	SyncWithFPS = -1
)

const (
	// maxCatchUpTicks is the maximum number of ticks that one frame can run to catch up with the real time.
	maxCatchUpTicks = 5

	// minCatchUpTime is the lower limit of the real time that one frame can catch up with.
	// This limit matters when TPS is so big that maxCatchUpTicks ticks are shorter than one frame (#1444).
	minCatchUpTime = 5 * time.Second / 60
)

// Clock determines how many ticks should run in each frame, so that ticks run at the specified TPS
// as closely as the frame rate allows.
type Clock struct {
	tps atomic.Int64

	// unconsumedTime is the real time that has passed but that ticks have not consumed yet.
	// This is negative when a tick has run a little earlier than its exact timing.
	unconsumedTime int64

	// lastNow is the time given to the last UpdateFrame call.
	lastNow int64

	// frameUpdated reports whether UpdateFrame has been called.
	frameUpdated bool

	// smoothedFrameInterval is the real time between recent frames.
	smoothedFrameInterval int64

	// frameIntervals is the number of frame intervals that smoothedFrameInterval is based on, up to 8.
	frameIntervals int64

	// smoothedFrameIntervalDeviation is the smoothed deviation of frame intervals from smoothedFrameInterval.
	smoothedFrameIntervalDeviation int64

	// skippableFrameIntervals is the number of irregular frame intervals that can still be skipped in a row.
	skippableFrameIntervals int64

	// frameIntervalSkipped reports whether a frame interval has been skipped since rounding last ran a tick.
	frameIntervalSkipped bool

	// framesSinceTick is the number of frames since the last frame that ran a tick.
	framesSinceTick int64

	actualFPS   float64
	actualTPS   float64
	lastUpdated int64
	fpsCount    int
	tpsCount    int

	mu sync.Mutex
}

// NewClock returns a new Clock. now is the current time, in the same time base as UpdateFrame's argument.
func NewClock(now int64) *Clock {
	c := &Clock{
		lastNow:     now,
		lastUpdated: now,
	}
	c.tps.Store(DefaultTPS)
	return c
}

func (c *Clock) ActualFPS() float64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.actualFPS
}

func (c *Clock) ActualTPS() float64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.actualTPS
}

// SetTPS sets the number of ticks per second. 0 stops ticks.
//
// If tps is negative but not SyncWithFPS, SetTPS panics.
func (c *Clock) SetTPS(tps int) {
	if tps < 0 && tps != SyncWithFPS {
		panic("clock: tps must be >= 0 or SyncWithFPS")
	}
	c.tps.Store(int64(tps))
}

func (c *Clock) TPS() int {
	return int(c.tps.Load())
}

// UpdateFrame updates the inner clock state and returns an integer value
// indicating how many times the game should update based on the current TPS.
//
// If TPS is SyncWithFPS, UpdateFrame always returns 1.
// If TPS is 0, UpdateFrame always returns 0.
// The real time before the first call doesn't count toward ticks.
//
// UpdateFrame is expected to be called once per frame with a monotonic now.
func (c *Clock) UpdateFrame(now int64) int {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.lastNow > now {
		// This ensures that now must be monotonic (#875).
		panic("clock: now must be monotonic")
	}
	delta := now - c.lastNow
	c.lastNow = now
	// The first delta is the time since NewClock, which is neither a frame interval nor the game's time.
	if c.frameUpdated {
		c.updateSmoothedFrameInterval(delta)
	} else {
		delta = 0
	}
	c.frameUpdated = true

	var count int
	switch tps := c.tps.Load(); {
	case tps == SyncWithFPS:
		count = 1
	case tps > 0:
		count = c.consumeTicks(tps, delta)
	default:
		// Ticks are stopped. Don't keep the elapsed time so that restarting them doesn't run a burst of ticks.
		c.unconsumedTime = 0
	}

	if count > 0 {
		c.framesSinceTick = 0
	} else {
		c.framesSinceTick++
	}

	c.updateActualFPSAndTPS(now, count)

	return count
}

// updateSmoothedFrameInterval updates the smoothed frame interval with the real time between the last
// two frames.
func (c *Clock) updateSmoothedFrameInterval(interval int64) {
	s := c.smoothedFrameInterval
	deviation := abs(interval - s)

	if c.frameIntervals < 8 {
		// The first frame intervals can be irregular, such as while the display queue fills up. Until the
		// average is based on enough of them, start it over at an interval far from it.
		if interval > 2*s || 2*interval < s {
			c.frameIntervals = 0
		}
	} else {
		// After that, skip up to 8 irregular frame intervals in a row, such as a long frame and the short
		// frames catching up after it, so that they don't move the average across the ratios judged in
		// consumeTicks. An irregular one is off by more than both a quarter of the average and 6 times the
		// deviation, which a jitter hardly is, or is shorter than half the average right after another.
		irregular := deviation > max(s/4, 6*c.smoothedFrameIntervalDeviation) ||
			(c.skippableFrameIntervals < 8 && 2*interval < s)
		if !irregular {
			c.skippableFrameIntervals = 8
		} else if c.skippableFrameIntervals > 0 {
			c.skippableFrameIntervals--
			c.frameIntervalSkipped = true
			return
		}
	}

	// Average the intervals so far, and then take an exponential moving average.
	c.frameIntervals = min(c.frameIntervals+1, 8)
	c.smoothedFrameInterval += (interval - s) / c.frameIntervals
	c.smoothedFrameIntervalDeviation += (deviation - c.smoothedFrameIntervalDeviation) / c.frameIntervals
}

// consumeTicks returns how many ticks the elapsed real time affords, and subtracts them from the
// unconsumed time.
func (c *Clock) consumeTicks(tps int64, delta int64) int {
	// Avoid a zero tick, which an unrealistically big TPS would cause.
	tick := max(int64(time.Second)/tps, 1)

	c.unconsumedTime += delta

	// Limit the time that one frame can catch up with, so that a slow frame doesn't run so many ticks that
	// the frame rate never recovers. The game runs in slow motion instead.
	limit := max(maxCatchUpTicks*tick, int64(minCatchUpTime))
	c.unconsumedTime = min(c.unconsumedTime, limit)

	// When frames come at a multiple of a tick, a jitter around a rounding boundary flips the count like
	// 0, 2, 0, 2. To avoid this, run the multiple of ticks while the game time stays within a tick of the
	// real time, and otherwise let the rounding below bring it back to the middle. The ratio is judged by
	// the smoothed frame interval once it is based on a few frame intervals, with a tight tolerance so that
	// a jitter doesn't switch between this and the rounding.
	judgeable := c.frameIntervals >= 3
	n := (delta + tick/2) / tick
	if judgeable && n > 0 && n*tick <= limit && abs(c.smoothedFrameInterval-n*tick) < tick/16 && abs(c.unconsumedTime-n*tick) < tick {
		c.unconsumedTime -= n * tick
		return int(n)
	}

	// Likewise, when frames come k times per tick, a jitter moves the tick between adjacent frames like
	// 1, 0, 0, 1, 1, 0. Run a tick every k-th frame while the game time stays within half a tick and half a
	// frame of the real time. k is up to 4, as the error of the smoothed frame interval is multiplied by k.
	// A skipped frame interval, such as a dropped frame, can shift the phase, so this waits until the
	// rounding runs a tick.
	if s := c.smoothedFrameInterval; judgeable && !c.frameIntervalSkipped && s > 0 {
		k := (tick + s/2) / s
		if k >= 2 && k <= 4 && abs(k*s-tick) < tick/16 {
			var count int64
			if c.framesSinceTick >= k-1 {
				count = 1
			}
			if abs(c.unconsumedTime-count*tick) < tick/2+tick/(2*k) {
				c.unconsumedTime -= count * tick
				return int(count)
			}
		}
	}

	// Round to the nearest tick. The rounded-up part is kept as a negative unconsumed time to avoid a drift.
	t := c.unconsumedTime + tick/2
	if t <= 0 {
		return 0
	}
	count := t / tick
	c.unconsumedTime -= count * tick
	if count > 0 {
		c.frameIntervalSkipped = false
	}
	return int(count)
}

func abs(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}

func (c *Clock) updateActualFPSAndTPS(now int64, count int) {
	c.fpsCount++
	c.tpsCount += count
	if time.Second > time.Duration(now-c.lastUpdated) {
		return
	}
	c.actualFPS = float64(c.fpsCount) * float64(time.Second) / float64(now-c.lastUpdated)
	c.actualTPS = float64(c.tpsCount) * float64(time.Second) / float64(now-c.lastUpdated)
	c.lastUpdated = now
	c.fpsCount = 0
	c.tpsCount = 0
}

// SinkTick advances the game time by one tick run outside of UpdateFrame, such as one forced on window
// resize, so that the following UpdateFrame calls return correspondingly fewer ticks. The game time is
// never advanced past the real time. now is the current time, in the same time base as UpdateFrame's
// argument.
//
// SinkTick has no effect unless a positive TPS is set.
func (c *Clock) SinkTick(now int64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	tps := c.tps.Load()
	if tps <= 0 {
		return
	}

	// Consume the real time since the last frame too, as UpdateFrame might not be called in between.
	c.unconsumedTime = max(c.unconsumedTime+(now-c.lastNow)-int64(time.Second)/tps, 0)
	c.lastNow = now
	c.framesSinceTick = 0
}

var theClock = NewClock(now())

func ActualFPS() float64 {
	return theClock.ActualFPS()
}

func ActualTPS() float64 {
	return theClock.ActualTPS()
}

func SetTPS(tps int) {
	theClock.SetTPS(tps)
}

func TPS() int {
	return theClock.TPS()
}

// UpdateFrame updates the inner clock state and returns an integer value
// indicating how many times the game should update based on the current TPS.
//
// See [Clock.UpdateFrame].
func UpdateFrame() int {
	return theClock.UpdateFrame(now())
}

// SinkTick advances the game time by one tick.
//
// See [Clock.SinkTick].
func SinkTick() {
	theClock.SinkTick(now())
}
