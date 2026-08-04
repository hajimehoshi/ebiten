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

//go:build !android && !ios && !js && !nintendosdk && !playstation5

package ui

import (
	"testing"
	"time"
)

func TestNextUnfocusedWake(t *testing.T) {
	const wait = time.Second / 60
	base := time.Unix(0, 0)

	tests := []struct {
		name     string
		lastWake time.Time
		now      time.Time
		want     time.Time
	}{
		{
			name:     "first frame schedules one interval ahead",
			lastWake: base,
			now:      base,
			want:     base.Add(wait),
		},
		{
			name:     "target is kept when the frame was quick",
			lastWake: base,
			now:      base.Add(wait / 4),
			want:     base.Add(wait),
		},
		{
			name:     "a small overshoot still advances by a whole interval",
			lastWake: base,
			now:      base.Add(wait - time.Nanosecond),
			want:     base.Add(wait),
		},
		{
			name:     "no catch-up burst after a long stall",
			lastWake: base,
			now:      base.Add(10 * wait),
			want:     base.Add(10 * wait),
		},
		{
			name:     "exactly on target does not sleep",
			lastWake: base,
			now:      base.Add(wait),
			want:     base.Add(wait),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := nextUnfocusedWake(tc.lastWake, tc.now); !got.Equal(tc.want) {
				t.Errorf("nextUnfocusedWake(%v, %v) = %v, want %v", tc.lastWake, tc.now, got, tc.want)
			}
		})
	}
}
