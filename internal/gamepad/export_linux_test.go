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
	"unsafe"

	"golang.org/x/sys/unix"
)

const (
	FFEffectSize         = unsafe.Sizeof(ff_effect{})
	FFEffectUnionOffset  = unsafe.Offsetof(ff_effect{}.u)
	FFEffectRumbleOffset = unsafe.Offsetof(ff_effect{}.u) + unsafe.Offsetof(ff_effect_union{}.rumble)
)

// InputEvent is a kernel input event used to drive the event processing from
// tests. It corresponds to struct input_event in the Linux kernel.
type InputEvent struct {
	Typ   uint16
	Code  uint16
	Value int32
}

// The event kinds and codes tests need.
const (
	EVSyn      = unix.EV_SYN
	EVKey      = unix.EV_KEY
	EVAbs      = unix.EV_ABS
	SynReport  = _SYN_REPORT
	SynDropped = _SYN_DROPPED
	BtnA       = _BTN_A
	AbsX       = _ABS_X
)

// TestGamepad drives the event processing of a native gamepad implementation
// from tests without exposing its private state.
type TestGamepad struct {
	impl *nativeGamepadImpl
}

// NewTestGamepad returns a test gamepad with BTN_A mapped to the button 0 and
// ABS_X (-1..1) mapped to the axis 0.
func NewTestGamepad() *TestGamepad {
	g := &nativeGamepadImpl{}
	for i := range g.keyMap {
		g.keyMap[i] = -1
	}
	for i := range g.absMap {
		g.absMap[i] = -1
	}
	g.keyMap[_BTN_A-_BTN_MISC] = 0
	g.absMap[_ABS_X] = 0
	g.absInfo[_ABS_X] = input_absinfo{minimum: -1, maximum: 1}
	g.buttonCount_ = 1
	g.axisCount_ = 1
	return &TestGamepad{impl: g}
}

// HandleEvents processes the events with the same code path that handles one
// batch read from the device. restoreDeviceState corresponds to the
// production pollAbsState/pollKeyState calls, and is called when the recovery
// from a SYN_DROPPED event needs the device state.
func (gp *TestGamepad) HandleEvents(events []InputEvent, restoreDeviceState func() error) error {
	if len(events) == 0 {
		return nil
	}
	evs := make([]input_event, len(events))
	for i, e := range events {
		evs[i] = input_event{
			typ:   e.Typ,
			code:  e.Code,
			value: e.Value,
		}
	}
	buf := unsafe.Slice((*byte)(unsafe.Pointer(&evs[0])), len(evs)*int(unsafe.Sizeof(input_event{})))
	return gp.impl.handleEvents(buf, restoreDeviceState)
}

// IsButtonPressed reports whether the button with the given index is pressed.
func (gp *TestGamepad) IsButtonPressed(button int) bool {
	return gp.impl.isButtonPressed(button)
}

// AxisValue returns the value of the axis with the given index.
func (gp *TestGamepad) AxisValue(axis int) float64 {
	return gp.impl.axisValue(axis)
}
