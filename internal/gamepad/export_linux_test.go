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

// Event node capability codes, for building the bitmaps classification reads.
const (
	ABS_X              = _ABS_X
	ABS_Y              = _ABS_Y
	ABS_RZ             = _ABS_RZ
	ABS_HAT0X          = _ABS_HAT0X
	ABS_HAT0Y          = _ABS_HAT0Y
	ABS_MT_SLOT        = _ABS_MT_SLOT
	ABS_MT_POSITION_X  = _ABS_MT_POSITION_X
	ABS_MT_POSITION_Y  = _ABS_MT_POSITION_Y
	ABS_MT_TRACKING_ID = _ABS_MT_TRACKING_ID

	BTN_LEFT           = 0x110
	BTN_SOUTH          = _BTN_A
	BTN_TOOL_FINGER    = 0x145
	BTN_TOUCH          = 0x14a
	BTN_TOOL_DOUBLETAP = 0x14d

	INPUT_PROP_ACCELEROMETER = _INPUT_PROP_ACCELEROMETER
)

type EvdevKind = evdevKind

const (
	EvdevKindOther        = evdevKindOther
	EvdevKindGamepad      = evdevKindGamepad
	EvdevKindTouchSurface = evdevKindTouchSurface
)

// bitmapForTest returns a bitmap of the given size with the given bits set.
func bitmapForTest(count int, bits []int) []byte {
	b := make([]byte, (count+7)/8)
	for _, bit := range bits {
		b[bit/8] |= 1 << (bit % 8)
	}
	return b
}

// ClassifyEvdevForTest classifies a node from the lists of its event types, key codes, absolute
// axis codes, and property bits. Reading the property bits fails with propsErr if it is not nil.
// propsRead reports whether the property bits were read.
func ClassifyEvdevForTest(evs, keys, abs, props []int, propsErr error) (kind EvdevKind, propsRead bool, err error) {
	kind, err = classifyEvdev(
		bitmapForTest(unix.EV_CNT, evs),
		bitmapForTest(_KEY_CNT, keys),
		bitmapForTest(_ABS_CNT, abs),
		func() ([]byte, error) {
			propsRead = true
			if propsErr != nil {
				return nil, propsErr
			}
			return bitmapForTest(_INPUT_PROP_CNT, props), nil
		})
	return kind, propsRead, err
}

type TouchNode = touchNode

// NewTouchNodeForTest returns a touch surface node with the given number of slots and position
// ranges from 0 to the maxima, with no node behind it: events are fed with HandleAbsEventForTest.
func NewTouchNodeForTest(slots int, xMax, yMax int32) *TouchNode {
	return &touchNode{
		xInfo: input_absinfo{maximum: xMax},
		yInfo: input_absinfo{maximum: yMax},
		slots: make([]touchContact, slots),
	}
}

func (t *touchNode) HandleAbsEventForTest(code int, value int32) {
	t.handleAbsEvent(code, value)
}

// NewLinuxGamepadForTest returns a gamepad on the evdev backend with no node behind it and the
// given touch surface, which may be nil.
func NewLinuxGamepadForTest(t *TouchNode) *Gamepad {
	return &Gamepad{native: &nativeGamepadImpl{touch: t}}
}
