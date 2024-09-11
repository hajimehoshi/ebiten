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
	"time"
)

const (
	DefaultTPS  = 60
	SyncWithFPS = -1
)

var (
	// tps represents TPS (ticks per second).
	tps = DefaultTPS

	lastNow int64

	// lastSystemTime is the last system time in the previous UpdateFrame.
	// lastSystemTime indicates the logical time in the game, so this can be bigger than the current time.
	lastSystemTime int64

	actualFPS   float64
	actualTPS   float64
	prevTPS     int64
	lastUpdated int64
	fpsCount    = 0
	tpsCount    = 0

	m sync.Mutex
)

func init() {
	n := now()
	lastNow = n
	lastSystemTime = n
	lastUpdated = n
}

func ActualFPS() float64 {
	m.Lock()
	defer m.Unlock()
	return actualFPS
}

func ActualTPS() float64 {
	m.Lock()
	defer m.Unlock()
	return actualTPS
}

func calcCountFromTPS(tps int64, now int64) int {
	if tps == 0 {
		return 0
	}
	if tps < 0 {
		panic("clock: tps must >= 0")
	}

	diff := now - lastSystemTime
	if diff < 0 {
		return 0
	}

	count := 0
	syncWithSystemClock := false

	// Detect whether the previous time is too old.
	// Use either 5 ticks or 5/60 sec in the case when TPS is too big like 300 (#1444).
	if diff > max(int64(time.Second)*5/tps, int64(time.Second)*5/60) || prevTPS != tps {
		// The previous time is too old.
		// Let's force to sync the game time with the system clock.
		syncWithSystemClock = true
	} else {
		count = int(diff * tps / int64(time.Second))
	}
	prevTPS = tps

	// Stabilize the count.
	// Without this adjustment, count can be unstable like 0, 2, 0, 2, ...
	// TODO: Brush up this logic so that this will work with any FPS. Now this works only when FPS = TPS.
	if count == 0 && (int64(time.Second)/tps/2) < diff {
		count = 1
	}
	if count == 2 && (int64(time.Second)/tps*3/2) > diff {
		count = 1
	}

	if syncWithSystemClock {
		lastSystemTime = now
	} else {
		lastSystemTime += int64(count) * int64(time.Second) / tps
	}

	return count
}

func updateFPSAndTPS(now int64, count int) {
	fpsCount++
	tpsCount += count
	if now < lastUpdated {
		panic("clock: lastUpdated must be older than now")
	}
	if time.Second > time.Duration(now-lastUpdated) {
		return
	}
	actualFPS = float64(fpsCount) * float64(time.Second) / float64(now-lastUpdated)
	actualTPS = float64(tpsCount) * float64(time.Second) / float64(now-lastUpdated)
	lastUpdated = now
	fpsCount = 0
	tpsCount = 0
}

// UpdateFrame updates the inner clock state and returns an integer value
// indicating how many times the game should update based on the current tps.
//
// If tps is SyncWithFPS, UpdateFrame always returns 1.
// If tps <= 0 and not SyncWithFPS, UpdateFrame always returns 0.
//
// UpdateFrame is expected to be called once per frame.
func UpdateFrame() int {
	m.Lock()
	defer m.Unlock()

	n := now()
	if lastNow > n {
		// This ensures that now() must be monotonic (#875).
		panic("clock: lastNow must be older than n")
	}
	lastNow = n

	c := 0
	if tps == SyncWithFPS {
		c = 1
	} else if tps > 0 {
		c = calcCountFromTPS(int64(tps), n)
	}
	updateFPSAndTPS(n, c)

	return c
}

func SetTPS(newTPS int) {
	m.Lock()
	defer m.Unlock()
	tps = newTPS
}

func TPS() int {
	m.Lock()
	defer m.Unlock()
	return tps
}
