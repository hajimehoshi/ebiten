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
// There are three types of clocks internally:
//
// System clock:
//   A clock offered by the OS.
//
// Audio clock:
//   An audio clock that is used in the higher priority over the system clock.
//   An audio clock might not exist when the audio is not used.
//
// Game clock:
//   A clock representing the actual game progress.
//   A game clock is basically updated based on the number of frames.
//   A game clock is adjusted by the audio clock when needed.
package clock

import (
	"time"

	"github.com/hajimehoshi/ebiten/internal/sync"
)

const FPS = 60

var (
	audioTimeInFrames     int64
	lastAudioTimeInFrames int64

	frames   int64
	gameTime int64

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

// ProceedAudioTimer increments the audio time by a frame.
func ProceedAudioTimer() {
	m.Lock()
	audioTimeInFrames++
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

	// Initialize gameTime if needed.
	if gameTime == 0 {
		gameTime = n
	}

	t := n - gameTime
	if t < 0 {
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
		syncWithSystemClock = true
	} else {
		// Use system clock when the audio clock is not updated yet.
		// As the audio clock can be updated discountinuously, the system clock is still needed.

		if t > 5*int64(time.Second)/FPS {
			// The previous time is too old.
			// Let's force to sync the game time with the system clock.
			syncWithSystemClock = true
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
	if syncWithSystemClock {
		gameTime = n
	} else {
		gameTime += int64(count) * int64(time.Second) / FPS
	}

	updateFPS(n)

	return count
}
