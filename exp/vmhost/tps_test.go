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

// The guest keeps the default TPS, drops it to 30 at tick 3, and switches to SyncWithFPS at tick 5 (see
// testdata/tps). The host asserts RequestedTPS reflects the guest's default before any tick and tracks
// each change afterward.

package vmhost_test

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestRequestedTPSForwarding(t *testing.T) {
	guest := startGuest(t, "./testdata/tps", activateByEnv, "unix")

	// AdvanceTicks requires a screen even though only the requested TPS is observed.
	scale := ebiten.Monitor().DeviceScaleFactor()
	if err := guest.SetOutsideScreen(ebiten.NewImage(int(64*scale), int(64*scale))); err != nil {
		t.Fatal(err)
	}

	// Before the guest reports anything, the host assumes the standard default.
	if tps := guest.RequestedTPS(); tps != 60 {
		t.Errorf("RequestedTPS() before any tick = %d; want 60", tps)
	}

	tests := []struct {
		tick int
		want int
	}{
		{tick: 1, want: 60},
		{tick: 2, want: 60},
		{tick: 3, want: 30},
		{tick: 4, want: 30},
		{tick: 5, want: ebiten.SyncWithFPS},
	}
	for _, tt := range tests {
		guest.AdvanceTicks(1)
		if !guest.WaitTicks() {
			t.Fatalf("waiting for tick %d failed: %v", tt.tick, guest.Err())
		}
		if got := guest.ProcessedTicks(); got != tt.tick {
			t.Errorf("ProcessedTicks() after tick %d = %d; want %d", tt.tick, got, tt.tick)
		}
		if tps := guest.RequestedTPS(); tps != tt.want {
			t.Errorf("RequestedTPS() after tick %d = %d; want %d", tt.tick, tps, tt.want)
		}
	}
}
