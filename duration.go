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

package ebiten

import (
	"time"

	"github.com/hajimehoshi/ebiten/v2/internal/clock"
)

// Duration represents elapsed time in units of 1/90,000 second.
type Duration int64

const (
	DurationMillisecond Duration = 90
	DurationSecond      Duration = Duration(clock.Second)
	DurationMinute               = 60 * DurationSecond
	DurationHour                 = 60 * DurationMinute
)

// DurationFromTime converts d to Duration, truncating toward zero.
func DurationFromTime(d time.Duration) Duration {
	return Duration(clock.DurationFromTime(d))
}

// TimeDuration converts d to time.Duration, truncating toward zero.
// Results outside time.Duration's range are clamped to its minimum or maximum.
func (d Duration) TimeDuration() time.Duration {
	return clock.Duration(d).TimeDuration()
}

// Seconds returns the duration in seconds.
func (d Duration) Seconds() float64 {
	return float64(d) / float64(DurationSecond)
}

// DurationTime returns the elapsed logical game time, which is zero before and during the first Update.
// With positive TPS, time advances by 1/TPS second per subsequent update.
// With [SyncWithFPS], time advances by elapsed real time, excluding engine-controlled pauses.
// Time is frozen at TPS zero and while the game is suspended.
// The value changes only at the start of an update and is rounded down to whole Duration units.
// Changing TPS preserves elapsed time but discards any fraction smaller than one unit.
//
// DurationTime is concurrent-safe.
func DurationTime() Duration {
	return Duration(clock.DurationTime())
}
