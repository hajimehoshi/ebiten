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

func TestHandleEventsSkipsKeyEventsBufferedBeforeRecovery(t *testing.T) {
	g := gamepad.NewNativeGamepadForTest(gamepad.BTN_SOUTH)
	restoreDeviceState := func() error {
		return nil
	}

	events := []gamepad.InputEventForTest{
		{
			Typ:  unix.EV_SYN,
			Code: gamepad.SYN_DROPPED,
		},
		{
			Typ:  unix.EV_SYN,
			Code: gamepad.SYN_REPORT,
		},
		{
			Typ:   unix.EV_KEY,
			Code:  gamepad.BTN_SOUTH,
			Value: 1,
		},
		{
			Typ:  unix.EV_SYN,
			Code: gamepad.SYN_REPORT,
		},
	}
	if err := g.HandleEventsForTest(events, restoreDeviceState); err != nil {
		t.Fatalf("HandleEventsForTest failed: %v", err)
	}
	if g.IsButtonPressedForTest(0) {
		t.Errorf("button 0: got: pressed, want: not pressed (the snapshot flushed the queued release, so the press already in this batch is stale and must be skipped)")
	}

	events = []gamepad.InputEventForTest{
		{
			Typ:   unix.EV_KEY,
			Code:  gamepad.BTN_SOUTH,
			Value: 1,
		},
		{
			Typ:  unix.EV_SYN,
			Code: gamepad.SYN_REPORT,
		},
	}
	if err := g.HandleEventsForTest(events, restoreDeviceState); err != nil {
		t.Fatalf("HandleEventsForTest for the later batch failed: %v", err)
	}
	if !g.IsButtonPressedForTest(0) {
		t.Errorf("button 0: got: not pressed, want: pressed (a key event in a later batch is newer than the snapshot and must not be skipped)")
	}
}
