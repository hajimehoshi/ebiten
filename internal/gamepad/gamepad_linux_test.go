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

//go:build !android && !nintendosdk && !playstation5

package gamepad_test

import (
	"testing"

	"golang.org/x/sys/unix"

	"github.com/hajimehoshi/ebiten/v2/internal/gamepad"
)

// TestHandleEventsSkipsKeyEventsBufferedBeforeRecovery tests that the key events left in the batch
// that ends a SYN_DROPPED recovery are not applied on top of the restored state, and that a later
// batch is applied normally. The kernel flushes the queued key events when it returns the key
// state, so an already-buffered press is older than the snapshot and would leave the button stuck.
func TestHandleEventsSkipsKeyEventsBufferedBeforeRecovery(t *testing.T) {
	g := gamepad.NewNativeGamepadForTest(gamepad.BTN_SOUTH)
	// The snapshot the recovery takes reports the button released, which is the state the kernel
	// flushed the queued press with.
	restoreDeviceState := func() error {
		return nil
	}

	events := []gamepad.InputEventForTest{
		{Typ: unix.EV_SYN, Code: gamepad.SYN_DROPPED},
		{Typ: unix.EV_SYN, Code: gamepad.SYN_REPORT},
		{Typ: unix.EV_KEY, Code: gamepad.BTN_SOUTH, Value: 1},
		{Typ: unix.EV_SYN, Code: gamepad.SYN_REPORT},
	}
	if err := g.HandleEventsForTest(events, restoreDeviceState); err != nil {
		t.Fatalf("HandleEventsForTest failed: %v", err)
	}
	if g.IsButtonPressedForTest(0) {
		t.Errorf("button 0: got: pressed, want: not pressed (a key event buffered before the key state snapshot must be skipped)")
	}

	// A key event in a later batch is newer than the snapshot and must be applied.
	events = []gamepad.InputEventForTest{
		{Typ: unix.EV_KEY, Code: gamepad.BTN_SOUTH, Value: 1},
		{Typ: unix.EV_SYN, Code: gamepad.SYN_REPORT},
	}
	if err := g.HandleEventsForTest(events, restoreDeviceState); err != nil {
		t.Fatalf("HandleEventsForTest for the later batch failed: %v", err)
	}
	if !g.IsButtonPressedForTest(0) {
		t.Errorf("button 0: got: not pressed, want: pressed (a later batch must not be skipped)")
	}
}
