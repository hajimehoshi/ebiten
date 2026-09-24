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
// axis codes, and property bits.
func ClassifyEvdevForTest(evs, keys, abs, props []int) EvdevKind {
	return classifyEvdev(
		bitmapForTest(unix.EV_CNT, evs),
		bitmapForTest(_KEY_CNT, keys),
		bitmapForTest(_ABS_CNT, abs),
		bitmapForTest(_INPUT_PROP_CNT, props))
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
