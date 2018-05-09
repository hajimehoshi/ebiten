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
//
// There are two types of clocks internally:
//
// System clock:
//   A clock offered by the OS.
//
// Audio clock:
//   An audio clock that is used in the higher priority over the system clock.
//   An audio clock might not exist when the audio is not used.
package clock

import (
	"sync"
	"time"
)

const FPS = 60

var (
	frames                int64
	audioTimeInFrames     int64
	lastAudioTimeInFrames int64

	// lastSystemTime is the last system time in the previous Update.
	lastSystemTime int64

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

// ProceedAudioTimer increments the audio time by the given number of frames.
func ProceedAudioTimer(num int64) {
	m.Lock()
	audioTimeInFrames += num
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
// indicating how many game frames the game should update.
func Update() int {
	m.Lock()
	defer m.Unlock()

	n := now()

	if ping != nil {
		ping()
	}

	// Initialize lastSystemTime if needed.
	if lastSystemTime == 0 {
		lastSystemTime = n
	}

	diff := n - lastSystemTime
	if diff < 0 {
		return 0
	}

	count := 0
	syncWithSystemClock := false

	if audioTimeInFrames > 0 && lastAudioTimeInFrames != audioTimeInFrames {
		// If the audio clock is updated, use this.
		if frames < audioTimeInFrames {
			count = int(audioTimeInFrames - frames)
		}
		lastAudioTimeInFrames = audioTimeInFrames

		// Now the current lastSystemTime value is not meaningful,
		// force to sync lastSystemTime with the system timer.
		syncWithSystemClock = true
	} else {
		// Use system clock when the audio clock is not updated yet.
		// As the audio clock can be updated discountinuously,
		// the system clock is still needed.

		if diff > 5*int64(time.Second)/FPS {
			// The previous time is too old.
			// Let's force to sync the game time with the system clock.
			syncWithSystemClock = true
		} else {
			count = int(diff * FPS / int64(time.Second))
		}
	}

	// Stabilize FPS.
	// Without this adjustment, count can be unstable like 0, 2, 0, 2, ...
	if count == 0 && (int64(time.Second)/FPS/2) < diff {
		count = 1
	}
	if count == 2 && (int64(time.Second)/FPS*3/2) > diff {
		count = 1
	}

	frames += int64(count)
	if syncWithSystemClock {
		lastSystemTime = n
	} else {
		lastSystemTime += int64(count) * int64(time.Second) / FPS
	}

	updateFPS(n)

	return count
}
