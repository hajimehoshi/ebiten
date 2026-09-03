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
	// This limit matters when TPS is so big that maxCatchUpTicks ticks are shorter than minCatchUpTime (#1444).
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

	c.updateActualFPSAndTPS(now, count)

	return count
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
	c.unconsumedTime = min(c.unconsumedTime, max(maxCatchUpTicks*tick, int64(minCatchUpTime)))

	// Round the count to the nearest tick rather than truncating it. With truncation, a small jitter in
	// the frame timing crosses a tick boundary back and forth, and the count becomes unstable like
	// 0, 2, 0, 2, ... whenever the frame interval is close to a multiple of a tick.
	//
	// The rounded-up part is kept as a negative unconsumed time, so that rounding never accumulates
	// into a drift from the real time.
	t := c.unconsumedTime + tick/2
	if t <= 0 {
		return 0
	}
	count := t / tick
	c.unconsumedTime -= count * tick
	return int(count)
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
// SinkTick has no effect unless a positive TPS is set.
func (c *Clock) SinkTick(now int64) {
	c.mu.Lock()
	defer c.mu.Unlock()

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
