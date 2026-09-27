// Copyright 2026 The Ebitengine Authors
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

package clock

import (
	"math"
	"sync"
	"sync/atomic"
	"time"
)

// Duration represents elapsed time in units of 1/90,000 second.
type Duration int64

const Second Duration = 90_000

// DurationFromTime converts d to Duration, truncating toward zero.
func DurationFromTime(d time.Duration) Duration {
	return Duration(d/time.Second)*Second + Duration(d%time.Second)*Second/Duration(time.Second)
}

// TimeDuration converts d to time.Duration, truncating toward zero and saturating on overflow.
func (d Duration) TimeDuration() time.Duration {
	const maxSeconds = Duration(math.MaxInt64 / int64(time.Second))
	const minSeconds = Duration(math.MinInt64 / int64(time.Second))
	seconds := d / Second
	if seconds > maxSeconds {
		return time.Duration(math.MaxInt64)
	}
	if seconds < minSeconds {
		return time.Duration(math.MinInt64)
	}
	whole := time.Duration(seconds) * time.Second
	fraction := time.Duration(d%Second) * time.Second / time.Duration(Second)
	if fraction > 0 && whole > time.Duration(math.MaxInt64)-fraction {
		return time.Duration(math.MaxInt64)
	}
	if fraction < 0 && whole < time.Duration(math.MinInt64)-fraction {
		return time.Duration(math.MinInt64)
	}
	return whole + fraction
}

// DurationClock tracks logical elapsed time. Its zero value is paused at TPS zero.
type DurationClock struct {
	published atomic.Int64

	started    bool
	suspended  bool
	tps        int
	elapsed    Duration
	fraction   int64
	lastSample time.Time
	mu         sync.Mutex
}

// Time returns the time published by the latest Update.
func (c *DurationClock) Time() Duration {
	return Duration(c.published.Load())
}

// SetTPS sets the rate for subsequent updates.
func (c *DurationClock) SetTPS(tps int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.tps == tps {
		return
	}
	if c.tps == SyncWithFPS || tps == SyncWithFPS {
		c.advanceMonotonic()
	}
	c.tps = tps
	c.fraction = 0
}

// SetSuspended pauses or resumes logical time.
func (c *DurationClock) SetSuspended(suspended bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.suspended == suspended {
		return
	}
	if c.tps == SyncWithFPS {
		c.advanceMonotonic()
	}
	c.suspended = suspended
}

// Update publishes the logical time for the next game update.
func (c *DurationClock) Update() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.started {
		if c.tps == SyncWithFPS && !c.suspended {
			c.advanceMonotonic()
		}
		c.started = true
		return
	}
	if c.suspended || c.tps == 0 {
		return
	}
	if c.tps == SyncWithFPS {
		c.advanceMonotonic()
	} else if c.tps > 0 {
		tps := int64(c.tps)
		c.elapsed += Second / Duration(tps)
		// Carry the remainder without overflowing at very large TPS values.
		remainder := int64(Second) % tps
		if c.fraction >= tps-remainder {
			c.elapsed++
			c.fraction -= tps - remainder
		} else {
			c.fraction += remainder
		}
	}
	c.published.Store(int64(c.elapsed))
}

func (c *DurationClock) advanceMonotonic() {
	now := time.Now()
	if c.started && !c.suspended && c.tps == SyncWithFPS {
		// time.Time preserves the monotonic clock reading. Fractional units carry
		// across samples, so sampling more frequently does not lose elapsed time.
		delta := now.Sub(c.lastSample)
		c.elapsed += Duration(delta/time.Second) * Second
		numerator := int64(delta%time.Second)*int64(Second) + c.fraction
		c.elapsed += Duration(numerator / int64(time.Second))
		c.fraction = numerator % int64(time.Second)
	}
	c.lastSample = now
}

func DurationTime() Duration {
	return theClock.duration.Time()
}

func UpdateDurationTime() {
	theClock.duration.Update()
}

func SetDurationSuspended(suspended bool) {
	theClock.duration.SetSuspended(suspended)
}
