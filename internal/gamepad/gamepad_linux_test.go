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

	"github.com/hajimehoshi/ebiten/v2/internal/gamepad"
)

// reportUnexpectedRestore returns a restore callback for tests that do not
// expect a recovery from a SYN_DROPPED event.
func reportUnexpectedRestore(t *testing.T) func() error {
	return func() error {
		t.Error("the device state must not be restored without SYN_REPORT after SYN_DROPPED")
		return nil
	}
}

func TestHandleEvents(t *testing.T) {
	gp := gamepad.NewTestGamepad()
	events := []gamepad.InputEvent{
		{Typ: gamepad.EVKey, Code: gamepad.BtnA, Value: 1},
		{Typ: gamepad.EVSyn, Code: gamepad.SynReport},
		{Typ: gamepad.EVAbs, Code: gamepad.AbsX, Value: 1},
		{Typ: gamepad.EVSyn, Code: gamepad.SynReport},
	}
	if err := gp.HandleEvents(events, reportUnexpectedRestore(t)); err != nil {
		t.Fatalf("HandleEvents failed: %v", err)
	}
	if !gp.IsButtonPressed(0) {
		t.Errorf("button 0: got: not pressed, want: pressed")
	}
	if got, want := gp.AxisValue(0), 1.0; got != want {
		t.Errorf("axis 0: got: %g, want: %g", got, want)
	}
}

// TestHandleEventsSynDroppedStaleKeyEvent tests that key events buffered after
// the SYN_REPORT that ends a SYN_DROPPED are not applied on top of the
// restored device state. The kernel flushes the queued key events when the key
// state is queried, so applying an already-buffered press would leave the
// button stuck until the next transition.
func TestHandleEventsSynDroppedStaleKeyEvent(t *testing.T) {
	gp := gamepad.NewTestGamepad()
	restoreCalled := 0
	restoreDeviceState := func() error {
		restoreCalled++
		return nil
	}

	// The press was queued before the release that the key-state snapshot
	// flushed, so it is stale once the snapshot is taken.
	events := []gamepad.InputEvent{
		{Typ: gamepad.EVSyn, Code: gamepad.SynDropped},
		{Typ: gamepad.EVSyn, Code: gamepad.SynReport},
		{Typ: gamepad.EVKey, Code: gamepad.BtnA, Value: 1},
		{Typ: gamepad.EVSyn, Code: gamepad.SynReport},
	}
	if err := gp.HandleEvents(events, restoreDeviceState); err != nil {
		t.Fatalf("HandleEvents failed: %v", err)
	}
	if restoreCalled != 1 {
		t.Errorf("the number of restoreDeviceState calls: got: %d, want: %d", restoreCalled, 1)
	}
	if gp.IsButtonPressed(0) {
		t.Errorf("button 0: got: pressed, want: not pressed (the stale key event must be skipped)")
	}

	// A key event in a later batch is not stale and must be applied.
	events = []gamepad.InputEvent{
		{Typ: gamepad.EVKey, Code: gamepad.BtnA, Value: 1},
		{Typ: gamepad.EVSyn, Code: gamepad.SynReport},
	}
	if err := gp.HandleEvents(events, restoreDeviceState); err != nil {
		t.Fatalf("HandleEvents for the later batch failed: %v", err)
	}
	if !gp.IsButtonPressed(0) {
		t.Errorf("button 0: got: not pressed, want: pressed (a later batch must not be skipped)")
	}
}

// TestHandleEventsSynDroppedUntilReport tests that events are ignored while
// the device is in the dropped state, across batches, until the next
// SYN_REPORT restores the device state.
func TestHandleEventsSynDroppedUntilReport(t *testing.T) {
	gp := gamepad.NewTestGamepad()
	restoreCalled := 0
	restoreDeviceState := func() error {
		restoreCalled++
		return nil
	}

	// The batch ends without SYN_REPORT.
	events := []gamepad.InputEvent{
		{Typ: gamepad.EVSyn, Code: gamepad.SynDropped},
		{Typ: gamepad.EVKey, Code: gamepad.BtnA, Value: 1},
		{Typ: gamepad.EVAbs, Code: gamepad.AbsX, Value: 1},
	}
	if err := gp.HandleEvents(events, restoreDeviceState); err != nil {
		t.Fatalf("HandleEvents for the first batch failed: %v", err)
	}
	if restoreCalled != 0 {
		t.Errorf("the number of restoreDeviceState calls: got: %d, want: %d", restoreCalled, 0)
	}
	if gp.IsButtonPressed(0) {
		t.Errorf("button 0: got: pressed, want: not pressed (events must be ignored while dropped)")
	}
	if got, want := gp.AxisValue(0), 0.0; got != want {
		t.Errorf("axis 0: got: %g, want: %g (events must be ignored while dropped)", got, want)
	}

	// The dropped state lasts until the SYN_REPORT of this batch. The events
	// before the report are ignored, and the key events after the report are
	// stale.
	events = []gamepad.InputEvent{
		{Typ: gamepad.EVKey, Code: gamepad.BtnA, Value: 1},
		{Typ: gamepad.EVSyn, Code: gamepad.SynReport},
		{Typ: gamepad.EVKey, Code: gamepad.BtnA, Value: 1},
	}
	if err := gp.HandleEvents(events, restoreDeviceState); err != nil {
		t.Fatalf("HandleEvents for the second batch failed: %v", err)
	}
	if restoreCalled != 1 {
		t.Errorf("the number of restoreDeviceState calls: got: %d, want: %d", restoreCalled, 1)
	}
	if gp.IsButtonPressed(0) {
		t.Errorf("button 0: got: pressed, want: not pressed")
	}
	if got, want := gp.AxisValue(0), 0.0; got != want {
		t.Errorf("axis 0: got: %g, want: %g", got, want)
	}

	// The next batch is processed normally.
	events = []gamepad.InputEvent{
		{Typ: gamepad.EVKey, Code: gamepad.BtnA, Value: 1},
		{Typ: gamepad.EVAbs, Code: gamepad.AbsX, Value: 1},
		{Typ: gamepad.EVSyn, Code: gamepad.SynReport},
	}
	if err := gp.HandleEvents(events, restoreDeviceState); err != nil {
		t.Fatalf("HandleEvents for the third batch failed: %v", err)
	}
	if restoreCalled != 1 {
		t.Errorf("the number of restoreDeviceState calls: got: %d, want: %d", restoreCalled, 1)
	}
	if !gp.IsButtonPressed(0) {
		t.Errorf("button 0: got: not pressed, want: pressed")
	}
	if got, want := gp.AxisValue(0), 1.0; got != want {
		t.Errorf("axis 0: got: %g, want: %g", got, want)
	}
}
