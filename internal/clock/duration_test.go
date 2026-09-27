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

package clock_test

import (
	"fmt"
	"math"
	"testing"
	"testing/synctest"
	"time"

	"github.com/hajimehoshi/ebiten/v2/internal/clock"
)

func TestDurationFixedTPS(t *testing.T) {
	for _, tps := range []int{1, 50, 60, 64, 100, 120, 144, 240, 90_001} {
		t.Run(fmt.Sprint(tps), func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				var c clock.DurationClock
				c.SetTPS(tps)
				if got := c.Time(); got != 0 {
					t.Errorf("before first update: got %d, want 0", got)
				}
				time.Sleep(time.Hour)
				c.Update()
				if got := c.Time(); got != 0 {
					t.Errorf("first update: got %d, want 0", got)
				}
				time.Sleep(time.Hour)
				for i := 1; i <= tps; i++ {
					// Fixed-rate time depends on updates, including multiple updates at the same instant.
					c.Update()
					want := clock.Duration(i) * clock.Second / clock.Duration(tps)
					if got := c.Time(); got != want {
						t.Errorf("update %d: got %d, want %d", i, got, want)
						break
					}
				}
			})
		})
	}
}

func TestDurationSyncWithFPS(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var c clock.DurationClock
		c.SetTPS(clock.SyncWithFPS)
		c.Update()
		var previous time.Duration
		for _, elapsed := range []time.Duration{time.Nanosecond, time.Microsecond, 11111 * time.Nanosecond, 11112 * time.Nanosecond, time.Second / 60, time.Second, 3 * time.Second} {
			time.Sleep(elapsed - previous)
			previous = elapsed
			c.Update()
			want := clock.DurationFromTime(elapsed)
			for range 2 {
				if got := c.Time(); got != want {
					t.Errorf("at %v: got %d, want %d", elapsed, got, want)
				}
			}
		}
	})
}

func TestDurationSyncWithFPSFraction(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var c clock.DurationClock
		c.SetTPS(clock.SyncWithFPS)
		c.Update()
		for range 1000 {
			time.Sleep(time.Microsecond)
			c.Update()
		}
		if got, want := c.Time(), clock.Second/1000; got != want {
			t.Errorf("after 1000 sub-unit intervals: got %d, want %d", got, want)
		}
	})
}

func TestDurationTPSChanges(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var c clock.DurationClock
		c.SetTPS(60)
		c.Update()
		c.Update()
		c.SetTPS(120)
		if got, want := c.Time(), clock.Second/60; got != want {
			t.Errorf("SetTPS changed published time: got %d, want %d", got, want)
		}
		c.Update()
		time.Sleep(time.Hour)
		c.SetTPS(clock.SyncWithFPS)
		time.Sleep(100 * time.Millisecond)
		c.Update()
		if got, want := c.Time(), clock.Second/60+clock.Second/120+clock.Second/10; got != want {
			t.Errorf("after entering SyncWithFPS: got %d, want %d", got, want)
		}
		// Capture elapsed time up to the transition without publishing in the middle of an update.
		time.Sleep(100 * time.Millisecond)
		c.SetTPS(60)
		if got, want := c.Time(), clock.Second/60+clock.Second/120+clock.Second/10; got != want {
			t.Errorf("SetTPS changed published time: got %d, want %d", got, want)
		}
		time.Sleep(time.Hour)
		c.Update()
		if got, want := c.Time(), 2*(clock.Second/60)+clock.Second/120+clock.Second/5; got != want {
			t.Errorf("after leaving SyncWithFPS: got %d, want %d", got, want)
		}
	})
}

func TestDurationPause(t *testing.T) {
	for _, tps := range []int{60, clock.SyncWithFPS} {
		for _, suspend := range []bool{false, true} {
			t.Run(fmt.Sprintf("TPS=%d/suspend=%t", tps, suspend), func(t *testing.T) {
				synctest.Test(t, func(t *testing.T) {
					var c clock.DurationClock
					c.SetTPS(tps)
					c.Update()
					time.Sleep(time.Second)
					c.Update()
					before := c.Time()
					pause := func(paused bool) {
						if suspend {
							c.SetSuspended(paused)
						} else if paused {
							c.SetTPS(0)
						} else {
							c.SetTPS(tps)
						}
					}
					pause(true)
					// Forced updates while paused must not advance time.
					time.Sleep(4 * time.Second)
					c.Update()
					time.Sleep(time.Second)
					pause(true)
					if got := c.Time(); got != before {
						t.Errorf("paused: got %d, want %d", got, before)
					}
					time.Sleep(4 * time.Second)
					pause(false)
					if got := c.Time(); got != before {
						t.Errorf("resume changed published time: got %d, want %d", got, before)
					}
					time.Sleep(time.Second)
					c.Update()
					want := before + clock.Second
					if tps > 0 {
						want = before + clock.Second/clock.Duration(tps)
					}
					if got := c.Time(); got != want {
						t.Errorf("resumed: got %d, want %d", got, want)
					}
				})
			})
		}
	}
}

func TestDurationPauseBetweenUpdates(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var c clock.DurationClock
		c.SetTPS(clock.SyncWithFPS)
		c.Update()
		time.Sleep(100 * time.Millisecond)
		c.SetSuspended(true)
		time.Sleep(time.Second)
		c.Update()
		if got := c.Time(); got != 0 {
			t.Errorf("forced update while suspended: got %d, want 0", got)
		}
		time.Sleep(time.Second)
		c.SetTPS(0)
		time.Sleep(time.Second)
		c.SetSuspended(false)
		time.Sleep(time.Second)
		c.SetTPS(clock.SyncWithFPS)
		time.Sleep(100 * time.Millisecond)
		c.Update()
		if got, want := c.Time(), clock.Second/5; got != want {
			t.Errorf("active time before and after overlapping pauses: got %d, want %d", got, want)
		}
	})
}

func TestDurationFractionOnTPSChange(t *testing.T) {
	var c clock.DurationClock
	c.SetTPS(64)
	c.Update()
	c.Update()
	c.SetTPS(64)
	for range 3 {
		c.Update()
	}
	if got, want := c.Time(), clock.Second/16; got != want {
		t.Errorf("unchanged TPS lost fractional carry: got %d, want %d", got, want)
	}
	c.Update()
	before := c.Time()
	c.SetTPS(128)
	c.Update()
	c.Update()
	c.Update()
	if got, want := c.Time(), before+3*clock.Second/128; got != want {
		t.Errorf("changed TPS retained old fraction: got %d, want %d", got, want)
	}
}

func TestDurationLargeTPS(t *testing.T) {
	var c clock.DurationClock
	c.SetTPS(math.MaxInt)
	c.Update()
	for range 10 {
		c.Update()
	}
	if got := c.Time(); got != 0 {
		t.Errorf("large TPS: got %d, want 0", got)
	}
}

func TestDurationZeroTPSWithoutUpdates(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var c clock.DurationClock
		c.SetTPS(clock.SyncWithFPS)
		c.Update()
		time.Sleep(100 * time.Millisecond)
		c.SetTPS(0)
		time.Sleep(time.Hour)
		c.SetTPS(clock.SyncWithFPS)
		time.Sleep(100 * time.Millisecond)
		c.Update()
		if got, want := c.Time(), clock.Second/5; got != want {
			t.Errorf("pause without updates: got %d, want %d", got, want)
		}
	})
}
