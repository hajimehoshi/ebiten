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

package gamepad

import (
	"testing"
	"unsafe"

	"golang.org/x/sys/unix"
)

func encodeInputEvents(t *testing.T, events []input_event) []byte {
	t.Helper()
	const (
		eventSize   = unsafe.Sizeof(input_event{})
		offsetTyp   = int(unsafe.Offsetof(input_event{}.typ))
		offsetCode  = int(unsafe.Offsetof(input_event{}.code))
		offsetValue = int(unsafe.Offsetof(input_event{}.value))
	)
	buf := make([]byte, 0, len(events)*int(eventSize))
	for _, e := range events {
		b := make([]byte, eventSize)
		b[offsetTyp] = byte(e.typ)
		b[offsetTyp+1] = byte(e.typ >> 8)
		b[offsetCode] = byte(e.code)
		b[offsetCode+1] = byte(e.code >> 8)
		b[offsetValue] = byte(e.value)
		b[offsetValue+1] = byte(e.value >> 8)
		b[offsetValue+2] = byte(e.value >> 16)
		b[offsetValue+3] = byte(e.value >> 24)
		buf = append(buf, b...)
	}
	return buf
}

func newTestGamepadImpl() *nativeGamepadImpl {
	g := &nativeGamepadImpl{}
	for i := range g.keyMap {
		g.keyMap[i] = -1
	}
	for i := range g.absMap {
		g.absMap[i] = -1
	}
	// Map BTN_A to the button 0 and ABS_X to the axis 0.
	g.keyMap[_BTN_A-_BTN_MISC] = 0
	g.absMap[_ABS_X] = 0
	g.absInfo[_ABS_X] = input_absinfo{minimum: -1, maximum: 1}
	return g
}

func TestHandleEvents(t *testing.T) {
	g := newTestGamepadImpl()
	buf := encodeInputEvents(t, []input_event{
		{typ: unix.EV_KEY, code: _BTN_A, value: 1},
		{typ: unix.EV_SYN, code: _SYN_REPORT},
		{typ: unix.EV_ABS, code: _ABS_X, value: 1},
		{typ: unix.EV_SYN, code: _SYN_REPORT},
	})
	if err := g.handleEvents(buf, nil); err != nil {
		t.Fatalf("handleEvents failed: %v", err)
	}
	if !g.buttons[0] {
		t.Errorf("buttons[0]: got: false, want: true")
	}
	if got, want := g.axes[0], 1.0; got != want {
		t.Errorf("axes[0]: got: %g, want: %g", got, want)
	}
}

// TestHandleEventsSynDroppedStaleKeyEvent tests that key events buffered after
// the SYN_REPORT that ends a SYN_DROPPED are not applied on top of the
// restored device state. The kernel flushes the queued key events when the key
// state is queried, so applying an already-buffered press would leave the
// button stuck until the next transition.
func TestHandleEventsSynDroppedStaleKeyEvent(t *testing.T) {
	g := newTestGamepadImpl()
	// Simulate the kernel state: the button is released. This is what
	// pollKeyState (EVIOCGKEY) would report, and the queued release is
	// flushed at the same time.
	restoreDeviceState := func() error {
		g.buttons[0] = false
		return nil
	}

	// The press was queued before the release that the ioctl flushed, so it
	// is stale once the snapshot is taken.
	buf := encodeInputEvents(t, []input_event{
		{typ: unix.EV_SYN, code: _SYN_DROPPED},
		{typ: unix.EV_SYN, code: _SYN_REPORT},
		{typ: unix.EV_KEY, code: _BTN_A, value: 1},
		{typ: unix.EV_SYN, code: _SYN_REPORT},
	})
	if err := g.handleEvents(buf, restoreDeviceState); err != nil {
		t.Fatalf("handleEvents failed: %v", err)
	}
	if g.dropped {
		t.Errorf("dropped: got: true, want: false")
	}
	if g.buttons[0] {
		t.Errorf("buttons[0]: got: true, want: false (the stale key event must be skipped)")
	}
}

// TestHandleEventsSynDroppedEventsAfterBatch tests that key events read in a
// later batch are applied again after a SYN_DROPPED recovery. Only the events
// of the batch that contained the recovery are stale.
func TestHandleEventsSynDroppedEventsAfterBatch(t *testing.T) {
	g := newTestGamepadImpl()
	restoreCalled := 0
	restoreDeviceState := func() error {
		restoreCalled++
		return nil
	}

	dropped := encodeInputEvents(t, []input_event{
		{typ: unix.EV_SYN, code: _SYN_DROPPED},
		{typ: unix.EV_KEY, code: _BTN_A, value: 1},
		{typ: unix.EV_SYN, code: _SYN_REPORT},
	})
	if err := g.handleEvents(dropped, restoreDeviceState); err != nil {
		t.Fatalf("handleEvents for the dropped batch failed: %v", err)
	}
	if restoreCalled != 1 {
		t.Errorf("restoreDeviceState calls: got: %d, want: %d", restoreCalled, 1)
	}
	// The event between SYN_DROPPED and SYN_REPORT must be ignored.
	if g.buttons[0] {
		t.Errorf("buttons[0]: got: true, want: false")
	}

	next := encodeInputEvents(t, []input_event{
		{typ: unix.EV_KEY, code: _BTN_A, value: 1},
		{typ: unix.EV_SYN, code: _SYN_REPORT},
	})
	if err := g.handleEvents(next, restoreDeviceState); err != nil {
		t.Fatalf("handleEvents for the next batch failed: %v", err)
	}
	if !g.buttons[0] {
		t.Errorf("buttons[0]: got: false, want: true (a later batch must not be skipped)")
	}
	if restoreCalled != 1 {
		t.Errorf("restoreDeviceState calls: got: %d, want: %d", restoreCalled, 1)
	}
}

// TestHandleEventsSynDroppedWithoutReport tests that events are ignored while
// the device is in the dropped state until the next SYN_REPORT arrives.
func TestHandleEventsSynDroppedWithoutReport(t *testing.T) {
	g := newTestGamepadImpl()
	restoreDeviceState := func() error {
		t.Error("restoreDeviceState must not be called without SYN_REPORT")
		return nil
	}

	buf := encodeInputEvents(t, []input_event{
		{typ: unix.EV_SYN, code: _SYN_DROPPED},
		{typ: unix.EV_KEY, code: _BTN_A, value: 1},
		{typ: unix.EV_ABS, code: _ABS_X, value: 1},
	})
	if err := g.handleEvents(buf, restoreDeviceState); err != nil {
		t.Fatalf("handleEvents failed: %v", err)
	}
	if !g.dropped {
		t.Errorf("dropped: got: false, want: true")
	}
	if g.buttons[0] {
		t.Errorf("buttons[0]: got: true, want: false")
	}
	if got, want := g.axes[0], 0.0; got != want {
		t.Errorf("axes[0]: got: %g, want: %g", got, want)
	}
}
