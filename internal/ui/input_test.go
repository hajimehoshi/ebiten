// Copyright 2025 The Ebitengine Authors
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

package ui_test

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2/internal/ui"
)

func TestMultipleKeyReleases(t *testing.T) {
	const baseTick = 100
	var inputState ui.InputState

	// Press 'A'.
	inputState.SetKeyPressed(ui.KeyA, ui.NewInputTimeFromTick(baseTick))
	if got, want := inputState.IsKeyPressed(ui.KeyA, baseTick), true; got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
	if got, want := inputState.IsKeyJustPressed(ui.KeyA, baseTick), true; got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
	if got, want := inputState.IsKeyJustReleased(ui.KeyA, baseTick), false; got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}

	// Check again in the next tick
	if got, want := inputState.IsKeyPressed(ui.KeyA, baseTick+1), true; got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
	if got, want := inputState.IsKeyJustPressed(ui.KeyA, baseTick+1), false; got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
	if got, want := inputState.IsKeyJustReleased(ui.KeyA, baseTick+1), false; got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}

	// Release 'A'.
	inputState.SetKeyReleased(ui.KeyA, ui.NewInputTimeFromTick(baseTick+2))
	if got, want := inputState.IsKeyPressed(ui.KeyA, baseTick+2), false; got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
	if got, want := inputState.IsKeyJustPressed(ui.KeyA, baseTick+2), false; got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
	if got, want := inputState.IsKeyJustReleased(ui.KeyA, baseTick+2), true; got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}

	// Release 'A' again in the next tick. This should be no-op (#3326).
	inputState.SetKeyReleased(ui.KeyA, ui.NewInputTimeFromTick(baseTick+3))
	if got, want := inputState.IsKeyPressed(ui.KeyA, baseTick+3), false; got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
	if got, want := inputState.IsKeyJustPressed(ui.KeyA, baseTick+3), false; got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
	if got, want := inputState.IsKeyJustReleased(ui.KeyA, baseTick+3), false; got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
}

func TestMultipleMouseButtonReleases(t *testing.T) {
	const baseTick = 100
	var inputState ui.InputState

	// Press a button.
	inputState.SetMouseButtonPressed(ui.MouseButton0, ui.NewInputTimeFromTick(baseTick))
	if got, want := inputState.IsMouseButtonPressed(ui.MouseButton0, baseTick), true; got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
	if got, want := inputState.IsMouseButtonJustPressed(ui.MouseButton0, baseTick), true; got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
	if got, want := inputState.IsMouseButtonJustReleased(ui.MouseButton0, baseTick), false; got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}

	// Check again in the next tick.
	if got, want := inputState.IsMouseButtonPressed(ui.MouseButton0, baseTick+1), true; got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
	if got, want := inputState.IsMouseButtonJustPressed(ui.MouseButton0, baseTick+1), false; got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
	if got, want := inputState.IsMouseButtonJustReleased(ui.MouseButton0, baseTick+1), false; got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}

	// Release the button.
	inputState.SetMouseButtonReleased(ui.MouseButton0, ui.NewInputTimeFromTick(baseTick+2))
	if got, want := inputState.IsMouseButtonPressed(ui.MouseButton0, baseTick+2), false; got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
	if got, want := inputState.IsMouseButtonJustPressed(ui.MouseButton0, baseTick+2), false; got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
	if got, want := inputState.IsMouseButtonJustReleased(ui.MouseButton0, baseTick+2), true; got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}

	// Release the button again in the next tick. This should be no-op (#3326).
	inputState.SetMouseButtonReleased(ui.MouseButton0, ui.NewInputTimeFromTick(baseTick+3))
	if got, want := inputState.IsMouseButtonPressed(ui.MouseButton0, baseTick+3), false; got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
	if got, want := inputState.IsMouseButtonJustPressed(ui.MouseButton0, baseTick+3), false; got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
	if got, want := inputState.IsMouseButtonJustReleased(ui.MouseButton0, baseTick+3), false; got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
}

// TestModifierKeyReleasedInTick tests that a modifier key released in the current tick is still
// reported as pressed, so that a chord whose events are delivered in one tick is not lost (#3497).
func TestModifierKeyReleasedInTick(t *testing.T) {
	const baseTick = 100
	var inputState ui.InputState

	inputState.SetKeyPressed(ui.KeyShiftLeft, ui.NewInputTimeFromTick(baseTick))
	inputState.SetKeyReleased(ui.KeyShiftLeft, ui.NewInputTimeFromTick(baseTick+2))

	if got, want := inputState.IsKeyPressed(ui.KeyShiftLeft, baseTick+2), true; got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
	if got, want := inputState.IsKeyJustReleased(ui.KeyShiftLeft, baseTick+2), true; got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
	// The press duration must agree with IsKeyPressed.
	if got, want := inputState.KeyPressDuration(ui.KeyShiftLeft, baseTick+2), int64(3); got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}

	// The virtual modifier key follows its variants.
	if got, want := inputState.IsKeyPressed(ui.KeyShift, baseTick+2), true; got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}

	// The key is no longer pressed in the next tick.
	if got, want := inputState.IsKeyPressed(ui.KeyShiftLeft, baseTick+3), false; got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
	if got, want := inputState.IsKeyPressed(ui.KeyShift, baseTick+3), false; got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
	if got, want := inputState.KeyPressDuration(ui.KeyShiftLeft, baseTick+3), int64(0); got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
}

// TestModifierKeyPressedAndReleasedInTick tests a modifier key whose press and release are both
// delivered in one tick.
func TestModifierKeyPressedAndReleasedInTick(t *testing.T) {
	const baseTick = 100
	var inputState ui.InputState

	inputState.SetKeyPressed(ui.KeyMetaLeft, ui.NewInputTimeFromTick(baseTick)+1)
	inputState.SetKeyReleased(ui.KeyMetaLeft, ui.NewInputTimeFromTick(baseTick)+2)

	if got, want := inputState.IsKeyPressed(ui.KeyMetaLeft, baseTick), true; got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
	if got, want := inputState.KeyPressDuration(ui.KeyMetaLeft, baseTick), int64(1); got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
	if got, want := inputState.IsKeyPressed(ui.KeyMetaLeft, baseTick+1), false; got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
}

// TestModifierKeyVariants tests that releasing one variant of a modifier key does not affect the other.
func TestModifierKeyVariants(t *testing.T) {
	const baseTick = 100
	var inputState ui.InputState

	inputState.SetKeyPressed(ui.KeyControlLeft, ui.NewInputTimeFromTick(baseTick))
	inputState.SetKeyPressed(ui.KeyControlRight, ui.NewInputTimeFromTick(baseTick))
	inputState.SetKeyReleased(ui.KeyControlLeft, ui.NewInputTimeFromTick(baseTick+1))

	if got, want := inputState.IsKeyPressed(ui.KeyControlRight, baseTick+2), true; got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
	if got, want := inputState.IsKeyPressed(ui.KeyControl, baseTick+2), true; got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}

	inputState.SetKeyReleased(ui.KeyControlRight, ui.NewInputTimeFromTick(baseTick+3))
	if got, want := inputState.IsKeyPressed(ui.KeyControl, baseTick+3), true; got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
	if got, want := inputState.IsKeyPressed(ui.KeyControl, baseTick+4), false; got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
}

// TestNonModifierKeyReleasedInTick tests that a key that is not a modifier key keeps the existing
// behavior: the release edge ends the press for the whole tick.
func TestNonModifierKeyReleasedInTick(t *testing.T) {
	const baseTick = 100
	var inputState ui.InputState

	inputState.SetKeyPressed(ui.KeyA, ui.NewInputTimeFromTick(baseTick))
	inputState.SetKeyReleased(ui.KeyA, ui.NewInputTimeFromTick(baseTick+2))

	if got, want := inputState.IsKeyPressed(ui.KeyA, baseTick+2), false; got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
	if got, want := inputState.KeyPressDuration(ui.KeyA, baseTick+2), int64(0); got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
}

// TestKeyPressedAndReleasedInSameTick tests that a key pressed and released within one tick — as the
// browser delivers it when the press and the release both land between two input reads — is still
// observed by the read that stamps it (#3317).
func TestKeyPressedAndReleasedInSameTick(t *testing.T) {
	const baseTick = 100
	var inputState ui.InputState

	// Both events are stamped with the tick that will read them, as the browsers do by folding the
	// queued events in at read time with the current input time.
	inputState.SetKeyPressed(ui.KeyA, ui.NewInputTimeFromTick(baseTick)+1)
	inputState.SetKeyReleased(ui.KeyA, ui.NewInputTimeFromTick(baseTick)+2)

	if got, want := inputState.IsKeyJustPressed(ui.KeyA, baseTick), true; got != want {
		t.Errorf("just pressed: got: %v, want: %v", got, want)
	}
	if got, want := inputState.IsKeyPressed(ui.KeyA, baseTick), true; got != want {
		t.Errorf("pressed: got: %v, want: %v", got, want)
	}

	// The press is gone once the tick has passed.
	if got, want := inputState.IsKeyPressed(ui.KeyA, baseTick+1), false; got != want {
		t.Errorf("pressed in the next tick: got: %v, want: %v", got, want)
	}
	if got, want := inputState.IsKeyJustPressed(ui.KeyA, baseTick+1), false; got != want {
		t.Errorf("just pressed in the next tick: got: %v, want: %v", got, want)
	}
}

// TestLockKeyStates tests that a lock key state a platform does not report falls back to Caps Lock
// off and Num Lock on.
func TestLockKeyStates(t *testing.T) {
	testCases := []struct {
		state ui.LockKeyState
		caps  bool
		num   bool
	}{
		{ui.LockKeyStateUnknown, false, true},
		{ui.LockKeyStateOn, true, true},
		{ui.LockKeyStateOff, false, false},
	}

	for _, tc := range testCases {
		var inputState ui.InputState
		inputState.CapsLock = tc.state
		inputState.NumLock = tc.state

		if got, want := inputState.IsCapsLockOn(), tc.caps; got != want {
			t.Errorf("state: %d, got: %v, want: %v", tc.state, got, want)
		}
		if got, want := inputState.IsNumLockOn(), tc.num; got != want {
			t.Errorf("state: %d, got: %v, want: %v", tc.state, got, want)
		}
	}
}
