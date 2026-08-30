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

//go:build js && wasm

package ui

import (
	"testing"
)

// TestQueuedKeyEventsAreStampedAtDrain verifies that a key pressed and released while the game loop
// is not reading the input is folded into the input state with the tick that drains it, so the press
// is observed by the tick that reads it and not lost to a stale tick (#3317).
func TestQueuedKeyEventsAreStampedAtDrain(t *testing.T) {
	u := &UserInterface{}

	// A tick boundary passes while the browser delivers a full tap (down then up) of Key A.
	u.incrementTick()
	u.enqueueKeys(KeyA, -1, true)
	u.enqueueKeys(KeyA, -1, false)

	// The game loop reads: drain the queue into the input state and snapshot it, as readInputState does.
	var snap InputState
	u.inputMu.Lock()
	u.applyInputEvents()
	u.inputState.copyAndReset(&snap)
	u.inputMu.Unlock()

	now := u.Tick()
	if got, want := snap.IsKeyJustPressed(KeyA, now), true; got != want {
		t.Errorf("just pressed in the draining tick: got: %v, want: %v", got, want)
	}
	if got, want := snap.IsKeyPressed(KeyA, now), true; got != want {
		t.Errorf("pressed in the draining tick: got: %v, want: %v", got, want)
	}

	// In the next tick the tap is over: the press and release were both stamped with the previous tick.
	u.inputMu.Lock()
	pressed := u.inputState.IsKeyPressed(KeyA, u.Tick()+1)
	justPressed := u.inputState.IsKeyJustPressed(KeyA, u.Tick()+1)
	u.inputMu.Unlock()
	if pressed {
		t.Errorf("still pressed in the next tick")
	}
	if justPressed {
		t.Errorf("still just pressed in the next tick")
	}
}

// TestQueuedKeyEventsAreAppliedInOrder verifies that the queued events keep their order, so a release
// that precedes a later press of the same key does not cancel the press (#3317).
func TestQueuedKeyEventsAreAppliedInOrder(t *testing.T) {
	u := &UserInterface{}

	// A tap followed by a held press, all delivered between two reads. The release must not end the
	// press because the press comes after it.
	u.incrementTick()
	u.enqueueKeys(KeyA, -1, true)
	u.enqueueKeys(KeyA, -1, false)
	u.enqueueKeys(KeyA, -1, true)

	u.inputMu.Lock()
	u.applyInputEvents()
	var snap InputState
	u.inputState.copyAndReset(&snap)
	u.inputMu.Unlock()

	now := u.Tick()
	if got, want := snap.IsKeyPressed(KeyA, now), true; got != want {
		t.Errorf("held press after a release: got: %v, want: %v", got, want)
	}
	if got, want := snap.IsKeyJustPressed(KeyA, now), true; got != want {
		t.Errorf("last event was a press: got: %v, want: %v", got, want)
	}
}
