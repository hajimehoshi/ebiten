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

package clock

import (
	"time"

	"github.com/hajimehoshi/ebiten/internal/sync"
)

const FPS = 60

var (
	primaryTime     int64
	lastPrimaryTime int64
	frames          int64
	logicalTime     int64

	currentFPS     float64
	lastFPSUpdated int64
	framesForFPS   int64

	ping func()

	m sync.Mutex
)

func CurrentFPS() float64 {
	m.Lock()
	v := currentFPS
	m.Unlock()
	return v
}

func RegisterPing(pingFunc func()) {
	m.Lock()
	ping = pingFunc
	m.Unlock()
}

// ProceedPrimaryTimer increments the primary time by a frame.
func ProceedPrimaryTimer() {
	m.Lock()
	primaryTime++
	m.Unlock()
}

func updateFPS(now int64) {
	if lastFPSUpdated == 0 {
		lastFPSUpdated = now
	}
	framesForFPS++
	if time.Second > time.Duration(now-lastFPSUpdated) {
		return
	}
	currentFPS = float64(framesForFPS) * float64(time.Second) / float64(now-lastFPSUpdated)
	lastFPSUpdated = now
	framesForFPS = 0
}

// Update updates the inner clock state and returns an integer value
// indicating how many logical frames the game should update.
func Update(now int64) int {
	m.Lock()
	defer m.Unlock()

	if ping != nil {
		ping()
	}

	// Initialize logicalTime if needed.
	if logicalTime == 0 {
		logicalTime = now
	}

	t := now - logicalTime
	if t < 0 {
		return 0
	}

	count := 0

	// There are two time lines: one is the logical time and the other is the system clock.
	//
	// Usually logical time is updated based on the number of frames, but
	// when sync is true, the logical time is forced to sync with the system clock.
	sync := false

	if primaryTime > 0 && lastPrimaryTime != primaryTime {
		// If the primary time is updated, use this.
		if frames < primaryTime {
			count = int(primaryTime - frames)
		}
		lastPrimaryTime = primaryTime
		sync = true
	} else {
		// Use system clock when
		// 1) Inc() is not called, or
		// 2) the primary time is not updated yet.
		// As the primary time can be updated discountinuously,
		// the system clock is still needed.

		if t > 5*int64(time.Second)/FPS {
			// The previous time is too old.
			// Let's force to sync the logical time with the OS clock.
			sync = true
		} else {
			count = int(t * FPS / int64(time.Second))
		}
	}

	// Stabilize FPS.
	if count == 0 && (int64(time.Second)/FPS/2) < t {
		count = 1
	}
	if count == 2 && (int64(time.Second)/FPS*3/2) > t {
		count = 1
	}
	if count > 3 {
		count = 3
	}

	frames += int64(count)
	if sync {
		logicalTime = now
	} else {
		logicalTime += int64(count) * int64(time.Second) / FPS
	}

	updateFPS(now)

	return count
}
