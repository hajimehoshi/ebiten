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

var (
	m        sync.Mutex
	tick     int64
	lastTick int64
	frames   int64
)

func Inc() {
	m.Lock()
	tick++
	m.Unlock()
}

func Frames(timeDuration time.Duration, fps int) (int, bool) {
	m.Lock()
	defer m.Unlock()

	count := 0

	sync := false
	if tick > 0 && lastTick != tick {
		if frames < tick {
			count = int(tick - frames)
		}
		lastTick = tick
		sync = true
	} else {
		// Use system clock when
		// 1) Inc() is not called, or
		// 2) tick is not updated yet.
		// As tick can be updated discountinuously, use system clock supplementarily.

		if int64(timeDuration) > 5*int64(time.Second)/int64(fps) {
			// The previous time is too old.
			// Let's force to sync the logical frame with the OS clock (or tick).
			return 0, true
		}
		count = int(int64(timeDuration) * int64(fps) / int64(time.Second))
	}

	// Stabilize FPS.
	if count == 0 && (int64(time.Second)/int64(fps)/2) < int64(timeDuration) {
		count = 1
	}
	if count == 2 && (int64(time.Second)/int64(fps)*3/2) > int64(timeDuration) {
		count = 1
	}
	if count > 3 {
		count = 3
	}

	frames += int64(count)
	return count, sync
}
