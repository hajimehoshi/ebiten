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
	tps      atomic.Int64
	duration DurationClock

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
	c.duration.SetTPS(DefaultTPS)
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
	c.mu.Lock()
	defer c.mu.Unlock()

	c.duration.SetTPS(tps)
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
	// The first delta is the time since NewClock, which is not a frame interval.
	if c.frameUpdated {
		c.updateSmoothedFrameInterval(delta)
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
	// The first frame intervals can be irregular, for example while frames don't wait for the display
	// until its queue is full, or while the first frames do heavy work. Until the average is based on
	// enough intervals, start it over at an interval far from it, so that the irregular intervals don't
	// keep the average off for long.
	if c.frameIntervals < 8 && (interval > 2*c.smoothedFrameInterval || 2*interval < c.smoothedFrameInterval) {
		c.frameIntervals = 0
	}

	// Average the intervals so far, and then the recent intervals with exponentially decreasing weights.
	c.frameIntervals = min(c.frameIntervals+1, 8)
	c.smoothedFrameInterval += (interval - c.smoothedFrameInterval) / c.frameIntervals
}

// consumeTicks returns how many ticks the elapsed real time affords, and subtracts them from the
// unconsumed time.
func (c *Clock) consumeTicks(tps int64, delta int64) int {
	// Avoid a zero tick, which an unrealistically big TPS would cause.
	tick := max(int64(time.Second)/tps, 1)

	c.unconsumedTime += delta

	// Limit the time that one frame can catch up with. Otherwise, a slow frame would run so many ticks
	// that the next frame gets even slower, and the frame rate would never recover. With this limit,
	// the game time gets behind the real time instead: the game runs in slow motion, which is
	// recoverable, rather than spiraling down.
	limit := max(maxCatchUpTicks*tick, int64(minCatchUpTime))
	c.unconsumedTime = min(c.unconsumedTime, limit)

	// When frames keep coming at a multiple of a tick, every frame sees the ticks at almost the same
	// phase. If the phase is around a boundary of the rounding below, a jitter in the frame timing
	// flips the count back and forth like 0, 2, 0, 2, ... until the phase drifts away from the boundary,
	// which can take seconds. To avoid this, run the multiple of ticks as long as the game time stays
	// within a tick of the real time. Once the game time goes out of this range, the rounding below
	// brings it back to around the middle of the range, far from both ends.
	//
	// Whether frames keep coming at the multiple is judged by the smoothed frame interval, as a single
	// frame interval is as noisy as the jitter. For the same reason, the judgement waits until the
	// smoothed frame interval is based on a few frame intervals. The judgement has a tight tolerance, as
	// switching between this and the rounding below back and forth with a jitter also makes the count
	// unstable.
	judgeable := c.frameIntervals >= 3
	n := (delta + tick/2) / tick
	if judgeable && n > 0 && n*tick <= limit && abs(c.smoothedFrameInterval-n*tick) < tick/16 && abs(c.unconsumedTime-n*tick) < tick {
		c.unconsumedTime -= n * tick
		return int(n)
	}

	// When frames keep coming k times per tick, every k-th frame sees the ticks at almost the same
	// phase. If the phase is around a boundary of the rounding below, a jitter in the frame timing
	// moves the tick between adjacent frames like 1, 0, 0, 1, 1, 0, ... To avoid this, run a tick on
	// every k-th frame as long as the game time stays within half a tick and half a frame of the real
	// time. Once the game time goes out of this range, the rounding below moves the tick by a frame,
	// which brings the game time back to around the middle of the range.
	//
	// The ratio is judged in the same way as above, but only up to 4 frames per tick, as the error of
	// the smoothed frame interval is multiplied by k.
	if s := c.smoothedFrameInterval; judgeable && s > 0 {
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

	// Round the count to the nearest tick. The rounded-up part is kept as a negative unconsumed time,
	// so that rounding never accumulates into a drift from the real time.
	t := c.unconsumedTime + tick/2
	if t <= 0 {
		return 0
	}
	count := t / tick
	c.unconsumedTime -= count * tick
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

// SinkTick advances the game time by one tick, so that the following UpdateFrame calls
// return correspondingly fewer ticks. It accounts for a tick that was run outside of UpdateFrame's
// own counting (such as a tick forced on window resize), keeping the total number of ticks
// consistent with the target TPS over time.
//
// now is the current time, in the same time base as UpdateFrame's argument.
//
// The game time is never advanced past the current real time, so that a long continuous resize does
// not bank an unbounded number of future ticks and stall the game after the resize ends.
//
// SinkTick counts the tick in ActualTPS regardless of the target TPS.
// It advances the game time only when a positive TPS is set.
func (c *Clock) SinkTick(now int64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.tpsCount++
	c.framesSinceTick = 0

	tps := c.tps.Load()
	if tps <= 0 {
		return
	}

	// Consume the real time that has passed since the last frame, so that the sunk tick counts even
	// when UpdateFrame is not called in between.
	c.unconsumedTime = max(c.unconsumedTime+(now-c.lastNow)-int64(time.Second)/tps, 0)
	c.lastNow = now
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
