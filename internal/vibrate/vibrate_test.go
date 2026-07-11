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

package vibrate_test

import (
	"slices"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2/internal/vibrate"
)

func TestRecording(t *testing.T) {
	// A device has no vibration platform in tests, so Vibrate only records once recording is enabled.
	vibrate.EnableRecording()

	// Nothing requested yet: the drain is empty.
	if got := vibrate.AppendPendingVibrations(nil); len(got) != 0 {
		t.Errorf("AppendPendingVibrations with no request = %v; want none", got)
	}

	vibrate.Vibrate(500*time.Millisecond, 0.25)

	got := vibrate.AppendPendingVibrations(nil)
	want := []vibrate.Vibration{
		{Duration: 500 * time.Millisecond, Magnitude: 0.25},
	}
	if !slices.Equal(got, want) {
		t.Errorf("AppendPendingVibrations = %v; want %v", got, want)
	}

	// A vibration is reported once: a second drain is empty.
	if got := vibrate.AppendPendingVibrations(nil); len(got) != 0 {
		t.Errorf("second AppendPendingVibrations = %v; want none (already drained)", got)
	}

	// A later request within one drain interval replaces an earlier one: a device has a single vibration
	// state, so the last write wins.
	vibrate.Vibrate(time.Second, 1)
	vibrate.Vibrate(0, 0)
	got = vibrate.AppendPendingVibrations(nil)
	want = []vibrate.Vibration{
		{Duration: 0, Magnitude: 0},
	}
	if !slices.Equal(got, want) {
		t.Errorf("after two requests, AppendPendingVibrations = %v; want %v (last write wins)", got, want)
	}
}
