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

package gamepad_test

import (
	"math"
	"slices"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2/internal/gamepad"
	"github.com/hajimehoshi/ebiten/v2/internal/gamepaddb"
)

func gamepadIDs() []gamepad.ID {
	return gamepad.AppendGamepadIDs(nil)
}

// updateVirtualGamepads drives Update with the snapshot (a non-nil slice keeps the subsystem virtual)
// and fails the test if Update errors.
func updateVirtualGamepads(t *testing.T, states []gamepad.VirtualGamepadState) {
	t.Helper()
	if err := gamepad.Update(0, states); err != nil {
		t.Fatal(err)
	}
}

func TestVirtualGamepadConnectAndDisconnect(t *testing.T) {
	updateVirtualGamepads(t, []gamepad.VirtualGamepadState{
		{ID: 0, SDLID: "id0", Name: "Pad 0"},
		{ID: 2, SDLID: "id2", Name: "Pad 2"},
	})
	defer updateVirtualGamepads(t, []gamepad.VirtualGamepadState{})

	if got, want := gamepadIDs(), []gamepad.ID{0, 2}; !slices.Equal(got, want) {
		t.Errorf("gamepad IDs = %v; want %v", got, want)
	}
	if g := gamepad.Get(1); g != nil {
		t.Errorf("Get(1) = %v; want nil (the snapshot skips ID 1)", g)
	}
	if g := gamepad.Get(2); g == nil {
		t.Fatal("Get(2) = nil; want a gamepad")
	} else if got, want := g.Name(), "Pad 2"; got != want {
		t.Errorf("Get(2).Name() = %q; want %q", got, want)
	}

	// Drop ID 0; ID 2 stays.
	updateVirtualGamepads(t, []gamepad.VirtualGamepadState{
		{ID: 2, SDLID: "id2", Name: "Pad 2"},
	})
	if got, want := gamepadIDs(), []gamepad.ID{2}; !slices.Equal(got, want) {
		t.Errorf("after dropping ID 0, gamepad IDs = %v; want %v", got, want)
	}

	// The empty snapshot disconnects everything.
	updateVirtualGamepads(t, []gamepad.VirtualGamepadState{})
	if got := gamepadIDs(); len(got) != 0 {
		t.Errorf("after the empty snapshot, gamepad IDs = %v; want none", got)
	}
}

func TestVirtualGamepadRawState(t *testing.T) {
	updateVirtualGamepads(t, []gamepad.VirtualGamepadState{
		{
			ID:      0,
			SDLID:   "raw",
			Name:    "Raw Pad",
			Axes:    []float64{0.5, -0.25},
			Buttons: []bool{true, false, true},
		},
	})
	defer updateVirtualGamepads(t, []gamepad.VirtualGamepadState{})

	g := gamepad.Get(0)
	if g == nil {
		t.Fatal("Get(0) = nil; want a gamepad")
	}
	if got, want := g.SDLID(), "raw"; got != want {
		t.Errorf("SDLID() = %q; want %q", got, want)
	}
	if got, want := g.AxisCount(), 2; got != want {
		t.Errorf("AxisCount() = %d; want %d", got, want)
	}
	if got, want := g.Axis(1), -0.25; got != want {
		t.Errorf("Axis(1) = %v; want %v", got, want)
	}
	if got, want := g.ButtonCount(), 3; got != want {
		t.Errorf("ButtonCount() = %d; want %d", got, want)
	}
	if got, want := g.HatCount(), 0; got != want {
		t.Errorf("HatCount() = %d; want %d (hats are folded into buttons)", got, want)
	}
	for b, want := range []bool{true, false, true} {
		if got := g.Button(b); got != want {
			t.Errorf("Button(%d) = %v; want %v", b, got, want)
		}
	}
}

func TestVirtualGamepadStandardLayout(t *testing.T) {
	updateVirtualGamepads(t, []gamepad.VirtualGamepadState{
		{
			ID:    0,
			SDLID: "standard",
			Name:  "Standard Pad",
			StandardAxes: map[gamepaddb.StandardAxis]float64{
				gamepaddb.StandardAxisLeftStickHorizontal: 0.5,
			},
			StandardButtons: map[gamepaddb.StandardButton]gamepad.VirtualStandardGamepadButton{
				gamepaddb.StandardButtonRightBottom: {Pressed: true, Value: 1},
				gamepaddb.StandardButtonLeftTop:     {Pressed: false, Value: 0.5},
			},
		},
	})
	defer updateVirtualGamepads(t, []gamepad.VirtualGamepadState{})

	g := gamepad.Get(0)
	if g == nil {
		t.Fatal("Get(0) = nil; want a gamepad")
	}
	if !g.IsStandardLayoutAvailable() {
		t.Error("IsStandardLayoutAvailable() = false; want true")
	}

	if !g.IsStandardButtonAvailable(gamepaddb.StandardButtonRightBottom) {
		t.Error("IsStandardButtonAvailable(RightBottom) = false; want true")
	}
	if g.IsStandardButtonAvailable(gamepaddb.StandardButtonRightTop) {
		t.Error("IsStandardButtonAvailable(RightTop) = true; want false (it was not forwarded)")
	}
	if !g.IsStandardButtonPressed(gamepaddb.StandardButtonRightBottom) {
		t.Error("IsStandardButtonPressed(RightBottom) = false; want true")
	}
	if got, want := g.StandardButtonValue(gamepaddb.StandardButtonLeftTop), 0.5; got != want {
		t.Errorf("StandardButtonValue(LeftTop) = %v; want %v", got, want)
	}

	if !g.IsStandardAxisAvailable(gamepaddb.StandardAxisLeftStickHorizontal) {
		t.Error("IsStandardAxisAvailable(LeftStickHorizontal) = false; want true")
	}
	if g.IsStandardAxisAvailable(gamepaddb.StandardAxisRightStickVertical) {
		t.Error("IsStandardAxisAvailable(RightStickVertical) = true; want false (it was not forwarded)")
	}
	if got, want := g.StandardAxisValue(gamepaddb.StandardAxisLeftStickHorizontal), 0.5; math.Abs(got-want) > 1e-9 {
		t.Errorf("StandardAxisValue(LeftStickHorizontal) = %v; want %v", got, want)
	}
}

// TestVirtualGamepadStandardLayoutBypassesDB proves a virtual gamepad serves the forwarded standard
// layout even when its SDL ID has a gamepaddb mapping, so the host's mapping wins over the database.
func TestVirtualGamepadStandardLayoutBypassesDB(t *testing.T) {
	// A platform-less mapping registers on any OS. It maps the standard RightBottom button to raw
	// button 0.
	const sdlID = "00000000000000000000000000009099"
	if err := gamepaddb.Update([]byte(sdlID + ",DB Pad,a:b0,leftx:a0,\n")); err != nil {
		t.Fatal(err)
	}
	if !gamepaddb.HasStandardLayoutMapping(sdlID) {
		t.Fatalf("gamepaddb did not register the test mapping for %q", sdlID)
	}

	// Raw button 0 is not pressed, so the database mapping would report RightBottom as not pressed.
	// The forwarded standard layout says it IS pressed; the forwarded value must win.
	updateVirtualGamepads(t, []gamepad.VirtualGamepadState{
		{
			ID:      0,
			SDLID:   sdlID,
			Name:    "DB Pad",
			Buttons: []bool{false},
			StandardButtons: map[gamepaddb.StandardButton]gamepad.VirtualStandardGamepadButton{
				gamepaddb.StandardButtonRightBottom: {Pressed: true, Value: 1},
			},
		},
	})
	defer updateVirtualGamepads(t, []gamepad.VirtualGamepadState{})

	g := gamepad.Get(0)
	if g == nil {
		t.Fatal("Get(0) = nil; want a gamepad")
	}
	if got, want := g.SDLID(), sdlID; got != want {
		t.Errorf("SDLID() = %q; want %q (the real SDL ID is preserved)", got, want)
	}
	if !g.IsStandardButtonPressed(gamepaddb.StandardButtonRightBottom) {
		t.Error("IsStandardButtonPressed(RightBottom) = false; want true (the forwarded layout must win over gamepaddb)")
	}
}

func TestVirtualGamepadVibration(t *testing.T) {
	updateVirtualGamepads(t, []gamepad.VirtualGamepadState{
		{ID: 0, SDLID: "id0", Name: "Pad 0"},
		{ID: 1, SDLID: "id1", Name: "Pad 1"},
	})
	defer updateVirtualGamepads(t, []gamepad.VirtualGamepadState{})

	// Nothing requested yet: the drain is empty.
	if got := gamepad.AppendVirtualGamepadVibrations(nil); len(got) != 0 {
		t.Errorf("AppendVirtualGamepadVibrations with no request = %v; want none", got)
	}

	g0 := gamepad.Get(0)
	if g0 == nil {
		t.Fatal("Get(0) = nil; want a gamepad")
	}
	g0.Vibrate(500*time.Millisecond, 0.25, 0.75)

	got := gamepad.AppendVirtualGamepadVibrations(nil)
	want := []gamepad.VirtualGamepadVibration{
		{ID: 0, Duration: 500 * time.Millisecond, StrongMagnitude: 0.25, WeakMagnitude: 0.75},
	}
	if !slices.Equal(got, want) {
		t.Errorf("AppendVirtualGamepadVibrations = %v; want %v", got, want)
	}

	// A vibration is reported once: a second drain is empty.
	if got := gamepad.AppendVirtualGamepadVibrations(nil); len(got) != 0 {
		t.Errorf("second AppendVirtualGamepadVibrations = %v; want none (already drained)", got)
	}

	// A later request within one drain interval replaces an earlier one: a device has a single rumble
	// state, so the last write wins.
	g0.Vibrate(time.Second, 1, 1)
	g0.Vibrate(0, 0, 0)
	got = gamepad.AppendVirtualGamepadVibrations(nil)
	want = []gamepad.VirtualGamepadVibration{
		{ID: 0, Duration: 0, StrongMagnitude: 0, WeakMagnitude: 0},
	}
	if !slices.Equal(got, want) {
		t.Errorf("after two requests, AppendVirtualGamepadVibrations = %v; want %v (last write wins)", got, want)
	}

	// Each gamepad is reported on its own ID, in ID order.
	gamepad.Get(1).Vibrate(time.Second, 0.5, 0.5)
	g0.Vibrate(2*time.Second, 0.1, 0.2)
	got = gamepad.AppendVirtualGamepadVibrations(nil)
	want = []gamepad.VirtualGamepadVibration{
		{ID: 0, Duration: 2 * time.Second, StrongMagnitude: 0.1, WeakMagnitude: 0.2},
		{ID: 1, Duration: time.Second, StrongMagnitude: 0.5, WeakMagnitude: 0.5},
	}
	if !slices.Equal(got, want) {
		t.Errorf("AppendVirtualGamepadVibrations for two gamepads = %v; want %v", got, want)
	}
}

func TestVirtualGamepadModeCannotChange(t *testing.T) {
	// A non-nil snapshot fixes the subsystem as virtual.
	updateVirtualGamepads(t, []gamepad.VirtualGamepadState{})

	// Switching to a nil snapshot (real devices) afterward must fail.
	if err := gamepad.Update(0, nil); err == nil {
		t.Error("Update(0, nil) after a virtual Update = nil error; want an error")
	}
}

func TestVirtualGamepadCopiesState(t *testing.T) {
	axes := []float64{0.5}
	buttons := []bool{true}
	standardAxes := map[gamepaddb.StandardAxis]float64{
		gamepaddb.StandardAxisLeftStickHorizontal: 0.5,
	}
	updateVirtualGamepads(t, []gamepad.VirtualGamepadState{
		{ID: 0, Axes: axes, Buttons: buttons, StandardAxes: standardAxes},
	})
	defer updateVirtualGamepads(t, []gamepad.VirtualGamepadState{})

	// Mutating the caller's buffers after Update must not change the gamepad's state.
	axes[0] = -1
	buttons[0] = false
	standardAxes[gamepaddb.StandardAxisLeftStickHorizontal] = -1

	g := gamepad.Get(0)
	if g == nil {
		t.Fatal("Get(0) = nil; want a gamepad")
	}
	if got, want := g.Axis(0), 0.5; got != want {
		t.Errorf("Axis(0) = %v; want %v (input mutation leaked)", got, want)
	}
	if !g.Button(0) {
		t.Error("Button(0) = false; want true (input mutation leaked)")
	}
	if got, want := g.StandardAxisValue(gamepaddb.StandardAxisLeftStickHorizontal), 0.5; got != want {
		t.Errorf("StandardAxisValue(LeftStickHorizontal) = %v; want %v (input mutation leaked)", got, want)
	}
}
