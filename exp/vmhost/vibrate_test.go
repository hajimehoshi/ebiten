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

// End-to-end gamepad vibration forwarding: the host injects a gamepad, the guest's game requests a
// vibration during Update, and the host receives it through the OnGamepadVibration handler, proving the
// guest→host channel round-trips the gamepad ID, magnitudes, duration, and tick.

package vmhost_test

import (
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/exp/vmhost"
)

func TestGamepadVibrationForwarding(t *testing.T) {
	// The handler runs on this goroutine, during AdvanceTicks and WaitTicks, so the collected vibrations
	// need no lock.
	var got []vmhost.GamepadVibration

	guest := startGuestWithOptions(t, "./testdata/vibrate", activateByEnv, "unix", &vmhost.NewGuestSessionOptions{
		OnGamepadVibration: func(v vmhost.GamepadVibration) {
			got = append(got, v)
		},
	})

	// Advancing a tick requires a screen even though only the vibration is observed.
	scale := ebiten.Monitor().DeviceScaleFactor()
	outsideScreen := ebiten.NewImage(int(32*scale), int(32*scale))
	if err := guest.SetOutsideScreen(outsideScreen); err != nil {
		t.Fatal(err)
	}

	// The guest vibrates gamepad 0 during Update, so it must be connected before the tick. The injected
	// state is queued before the tick, so the guest's Update observes it.
	guest.UpdateGamepads([]vmhost.GamepadState{
		{ID: 0, SDLID: "00000000000000000000000076543210", Name: "VM Test Controller"},
	})

	guest.AdvanceTicks(1)
	if !guest.WaitTicks() {
		t.Fatalf("waiting for the tick failed: %v", guest.Err())
	}

	// WaitTicks delivers the tick's vibration to the handler before returning.
	if len(got) != 1 {
		t.Fatalf("got %d vibrations; want 1: %+v", len(got), got)
	}
	want := vmhost.GamepadVibration{
		GamepadID:       0,
		Duration:        500 * time.Millisecond,
		StrongMagnitude: 0.25,
		WeakMagnitude:   0.75,
		// The vibration was requested during the first tick, whose ebiten.Tick() is 0.
		StartTick: 0,
	}
	if got[0] != want {
		t.Errorf("vibration = %+v; want %+v", got[0], want)
	}

	// Every vibration is delivered — advancing several ticks does not coalesce them — each stamped with
	// the tick that produced it, so a host can tell how much guest time has elapsed since each request.
	guest.AdvanceTicks(3)
	if !guest.WaitTicks() {
		t.Fatalf("waiting for the batched ticks failed: %v", guest.Err())
	}
	if len(got) != 4 {
		t.Fatalf("got %d vibrations after 4 ticks; want 4: %+v", len(got), got)
	}
	for i, v := range got {
		// Four ticks have run, one vibration each; their ebiten.Tick() values are 0 through 3.
		if v.StartTick != i {
			t.Errorf("vibration %d has StartTick %d; want %d", i, v.StartTick, i)
		}
	}
}
