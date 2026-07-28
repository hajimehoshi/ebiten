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

func TestUnfocusedSleepDuration(t *testing.T) {
	const wait = time.Second / 60
	tests := []struct {
		name          string
		elapsed       time.Duration
		sleepOverhead time.Duration
		want          time.Duration
	}{
		{
			name: "zero",
			want: wait,
		},
		{
			name:    "partial",
			elapsed: time.Millisecond,
			want:    wait - time.Millisecond,
		},
		{
			name:          "sleep overhead",
			elapsed:       time.Millisecond,
			sleepOverhead: 2 * time.Millisecond,
			want:          wait - 3*time.Millisecond,
		},
		{
			name:    "at wait",
			elapsed: wait,
		},
		{
			name:    "over wait",
			elapsed: wait + time.Millisecond,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := unfocusedSleepDuration(tt.elapsed, tt.sleepOverhead); got != tt.want {
				t.Errorf("unfocusedSleepDuration(%v, %v) = %v, want %v", tt.elapsed, tt.sleepOverhead, got, tt.want)
			}
		})
	}
}
