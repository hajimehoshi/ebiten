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

var (
	lastNow int64

	// lastSystemTime is the last system time in the previous Update.
	// lastSystemTime indicates the logical time in the game, so this can be bigger than the curren time.
	lastSystemTime int64

	currentFPS  float64
	currentTPS  float64
	lastTPS     int64
	tpsCalcErr  int64
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

func CurrentFPS() float64 {
	m.Lock()
	v := currentFPS
	m.Unlock()
	return v
}

func CurrentTPS() float64 {
	m.Lock()
	v := currentTPS
	m.Unlock()
	return v
}

func min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

// calcTPSFactor calculates the TPS that is used for the timer and the factor for the count.
// If tps is under the baseTPS, use tps as newTPS. The factor is 1. The timer precision should be enough.
// If not, use baseTPS as newTPS and factor that can be more than 1.
func calcTPSFactor(tps, baseTPS int64) (newTPS int64, factor int) {
	if tps <= baseTPS {
		return tps, 1
	}

	if lastTPS != tps {
		tpsCalcErr = 0
	}
	lastTPS = tps

	factor = int(tps / baseTPS)
	tpsCalcErr += tps - baseTPS*int64(factor)

	factor += int(tpsCalcErr / baseTPS)
	tpsCalcErr %= baseTPS

	return baseTPS, factor
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

	// When TPS is big (e.g. 300), the timer precision is no longer reliable.
	// Multiply the factor later instead (#1444).
	var tpsFactor int
	tps, tpsFactor = calcTPSFactor(tps, 60) // TODO: 60 should be the current display's FPS.

	if diff > int64(time.Second)*5/tps {
		// The previous time is too old.
		// Let's force to sync the game time with the system clock.
		syncWithSystemClock = true
	} else {
		count = int(diff * tps / int64(time.Second))
	}

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

	return count * tpsFactor
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
	currentFPS = float64(fpsCount) * float64(time.Second) / float64(now-lastUpdated)
	currentTPS = float64(tpsCount) * float64(time.Second) / float64(now-lastUpdated)
	lastUpdated = now
	fpsCount = 0
	tpsCount = 0
}

const UncappedTPS = -1

// Update updates the inner clock state and returns an integer value
// indicating how many times the game should update based on given tps.
// tps represents TPS (ticks per second).
// If tps is UncappedTPS, Update always returns 1.
// If tps <= 0 and not UncappedTPS, Update always returns 0.
//
// Update is expected to be called per frame.
func Update(tps int) int {
	m.Lock()
	defer m.Unlock()

	n := now()
	if lastNow > n {
		// This ensures that now() must be monotonic (#875).
		panic("clock: lastNow must be older than n")
	}
	lastNow = n

	c := 0
	if tps == UncappedTPS {
		c = 1
	} else if tps > 0 {
		c = calcCountFromTPS(int64(tps), n)
	}
	updateFPSAndTPS(n, c)

	return c
}
