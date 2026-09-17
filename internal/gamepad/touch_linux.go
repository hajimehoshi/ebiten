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
	"fmt"
	"unsafe"

	"golang.org/x/sys/unix"
)

// evdevKind is what an event node is, decided from its capabilities and properties.
type evdevKind int

const (
	// evdevKindOther is a node that is neither: a keyboard, a mouse, or the motion sensors of a
	// controller, which the kernel registers as an accelerometer device of its own.
	evdevKindOther evdevKind = iota

	// evdevKindGamepad is a node with absolute axes that is polled for buttons, axes, and hats.
	evdevKindGamepad

	// evdevKindTouchSurface is a multitouch device without gamepad buttons: the touchpad of a
	// controller, which the kernel registers as a node of its own next to the gamepad node.
	evdevKindTouchSurface
)

// classifyEvdev decides what an event node is from its event, key, absolute axis, and property bits.
func classifyEvdev(evBits, keyBits, absBits, propBits []byte) evdevKind {
	if !isBitSet(evBits, unix.EV_ABS) {
		return evdevKindOther
	}
	if isBitSet(propBits, _INPUT_PROP_ACCELEROMETER) {
		return evdevKindOther
	}
	if isBitSet(absBits, _ABS_MT_SLOT) && isBitSet(absBits, _ABS_MT_POSITION_X) &&
		isBitSet(absBits, _ABS_MT_POSITION_Y) && isBitSet(absBits, _ABS_MT_TRACKING_ID) {
		hasGamepadButton := false
		for code := _BTN_JOYSTICK; code < _BTN_DIGI; code++ {
			if isBitSet(keyBits, code) {
				hasGamepadButton = true
				break
			}
		}
		for code := _BTN_DPAD_UP; code <= _BTN_DPAD_RIGHT; code++ {
			if isBitSet(keyBits, code) {
				hasGamepadButton = true
				break
			}
		}
		if !hasGamepadButton {
			return evdevKindTouchSurface
		}
	}
	return evdevKindGamepad
}

// maxTouchSlots bounds the slots read from a touch surface node, whatever it reports.
const maxTouchSlots = 16

// touchNode is the opened touch surface node of a controller. The kernel delivers its contacts
// with the multitouch protocol: a slot event selects the slot the following events apply to, a
// tracking id of -1 ends a contact and any other value identifies one, and each contact's
// position arrives as its own events.
type touchNode struct {
	path    string
	fdPlus1 int

	// uniq and id identify the controller the node belongs to; the gamepad node of the same
	// controller carries the same values.
	uniq string
	id   input_id

	xInfo, yInfo input_absinfo
	slots        []touchContact
	current      int
	dropped      bool
}

// newTouchNode reads the slot layout and the current contacts of an opened touch surface node.
func newTouchNode(fd int, path, uniq string, id input_id) (*touchNode, error) {
	t := &touchNode{
		path:    path,
		fdPlus1: fd + 1,
		uniq:    uniq,
		id:      id,
	}
	var slotInfo input_absinfo
	if err := ioctl(fd, uint(_EVIOCGABS(_ABS_MT_SLOT)), unsafe.Pointer(&slotInfo)); err != nil {
		return nil, fmt.Errorf("gamepad: ioctl for the slot abs failed: %w", err)
	}
	if err := ioctl(fd, uint(_EVIOCGABS(_ABS_MT_POSITION_X)), unsafe.Pointer(&t.xInfo)); err != nil {
		return nil, fmt.Errorf("gamepad: ioctl for the touch x abs failed: %w", err)
	}
	if err := ioctl(fd, uint(_EVIOCGABS(_ABS_MT_POSITION_Y)), unsafe.Pointer(&t.yInfo)); err != nil {
		return nil, fmt.Errorf("gamepad: ioctl for the touch y abs failed: %w", err)
	}
	slotCount := int(slotInfo.maximum) + 1
	if slotCount < 1 {
		return nil, fmt.Errorf("gamepad: touch surface has no slots")
	}
	t.slots = make([]touchContact, min(slotCount, maxTouchSlots))
	if err := t.pollSlots(); err != nil {
		return nil, err
	}
	return t, nil
}

func (t *touchNode) close() {
	if t.fdPlus1 == 0 {
		return
	}
	_ = unix.Close(t.fdPlus1 - 1)
	t.fdPlus1 = 0
}

// pollSlots replaces the contacts with the node's current slot state.
func (t *touchNode) pollSlots() error {
	// struct input_mt_request_layout { __u32 code; __s32 values[num_slots]; }
	buf := make([]int32, 1+len(t.slots))
	size := uint(len(buf)) * uint(unsafe.Sizeof(buf[0]))
	for _, code := range []int{_ABS_MT_TRACKING_ID, _ABS_MT_POSITION_X, _ABS_MT_POSITION_Y} {
		buf[0] = int32(code)
		if err := ioctl(t.fdPlus1-1, _EVIOCGMTSLOTS(size), unsafe.Pointer(&buf[0])); err != nil {
			return fmt.Errorf("gamepad: ioctl for the touch slots failed: %w", err)
		}
		for slot, value := range buf[1:] {
			t.handleSlotValue(slot, code, value)
		}
	}
	return nil
}

// update applies the node's pending events to its contacts.
func (t *touchNode) update() error {
	for {
		e, ok, err := readInputEvent(t.fdPlus1 - 1)
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}

		if e.typ == unix.EV_SYN && e.code == _SYN_DROPPED {
			t.dropped = true
		}
		if t.dropped {
			// Ignore events through the next SYN_REPORT, then restore the slot state.
			if e.typ == unix.EV_SYN && e.code == _SYN_REPORT {
				if err := t.pollSlots(); err != nil {
					return err
				}
				t.dropped = false
			}
			continue
		}

		if e.typ == unix.EV_ABS {
			t.handleAbsEvent(int(e.code), e.value)
		}
	}
}

// handleAbsEvent applies one absolute axis event: a slot event selects the current slot, and the
// other multitouch events update it. Events for a slot the node did not report are ignored.
func (t *touchNode) handleAbsEvent(code int, value int32) {
	if code == _ABS_MT_SLOT {
		t.current = int(value)
		return
	}
	if t.current < 0 || t.current >= len(t.slots) {
		return
	}
	t.handleSlotValue(t.current, code, value)
}

func (t *touchNode) handleSlotValue(slot, code int, value int32) {
	c := &t.slots[slot]
	switch code {
	case _ABS_MT_TRACKING_ID:
		c.active = value >= 0
		c.id = int(value)
	case _ABS_MT_POSITION_X:
		c.x = normalizeAbs(value, t.xInfo)
	case _ABS_MT_POSITION_Y:
		// The kernel's y is 0 at the top, as the touch API has -1 at the top.
		c.y = normalizeAbs(value, t.yInfo)
	}
}

// normalizeAbs maps an absolute axis value to -1..1 over the axis's range.
func normalizeAbs(value int32, info input_absinfo) float64 {
	v := float64(value)
	if r := float64(info.maximum) - float64(info.minimum); r != 0 {
		v = (v - float64(info.minimum)) / r
		v = v*2 - 1
	}
	return v
}

func (g *nativeGamepadImpl) touchSurfaceCount() int {
	if g.touch == nil {
		return 0
	}
	return 1
}

func (g *nativeGamepadImpl) touchSlotCount(surface int) int {
	if surface != 0 || g.touch == nil {
		return 0
	}
	return len(g.touch.slots)
}

func (g *nativeGamepadImpl) touchContactAt(surface, slot int) touchContact {
	if surface != 0 || g.touch == nil || slot < 0 || slot >= len(g.touch.slots) {
		return touchContact{}
	}
	return g.touch.slots[slot]
}
