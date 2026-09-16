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

package gamepad

import (
	"github.com/ebitengine/purego/objc"
)

// gcTouchSlotMax is the number of fingers the GameController framework tracks on a touch surface:
// the DualShock 4 and DualSense profiles have a primary and a secondary finger element.
const gcTouchSlotMax = 2

// gcTouchSlotKind is where a finger's element was found in a controller's physical input profile.
type gcTouchSlotKind uint8

const (
	gcTouchSlotNone gcTouchSlotKind = iota

	// gcTouchSlotTouchpad is a GCControllerTouchpad from the profile's touchpads. It carries a touch
	// state along with the finger's position.
	gcTouchSlotTouchpad

	// gcTouchSlotDPad is a GCControllerDirectionPad from the profile's dpads. It carries only the
	// finger's position, which is the origin while no finger is on the surface. The DualShock 4 and
	// DualSense expose their fingers this way; a finger cannot rest exactly on the origin, as their
	// touch grids have an even number of columns.
	gcTouchSlotDPad
)

// gcTouchStateUp is the GCTouchState value of GCControllerTouchpad.touchState with no finger on the
// surface; the other values are 1 while a finger has just touched down and 2 while it moves.
const gcTouchStateUp = 0

// gcTouchState is one finger's report read from its element. x and y are as the framework reports
// them, with positive y pointing up.
type gcTouchState struct {
	active bool
	x, y   float32
}

// gcTouchSlotKey returns the profile key of a finger's element, or 0 if its symbol did not load.
func gcTouchSlotKey(slot int) objc.ID {
	switch slot {
	case 0:
		return gcInputDualShockTouchpadOne
	case 1:
		return gcInputDualShockTouchpadTwo
	}
	return 0
}

// gcTouchSlotCount returns the number of fingers a controller's touch surface tracks: the leading
// slots that have an element.
func gcTouchSlotCount(kinds [gcTouchSlotMax]gcTouchSlotKind) int {
	n := 0
	for n < len(kinds) && kinds[n] != gcTouchSlotNone {
		n++
	}
	return n
}

// discoverGCTouchSlots finds the finger elements of a controller's touch surface in its physical
// input profile, preferring a touchpad element over a direction pad for the same finger.
func discoverGCTouchSlots(profile objc.ID) [gcTouchSlotMax]gcTouchSlotKind {
	var kinds [gcTouchSlotMax]gcTouchSlotKind

	var touchpads, dpads objc.ID
	if profile.Send(sel_respondsToSelector, sel_touchpads) != 0 {
		touchpads = profile.Send(sel_touchpads)
	}
	if profile.Send(sel_respondsToSelector, sel_dpads) != 0 {
		dpads = profile.Send(sel_dpads)
	}

	for slot := range kinds {
		key := gcTouchSlotKey(slot)
		if key == 0 {
			continue
		}
		if touchpads != 0 && touchpads.Send(sel_objectForKeyedSubscript, key) != 0 {
			kinds[slot] = gcTouchSlotTouchpad
		} else if dpads != 0 && dpads.Send(sel_objectForKeyedSubscript, key) != 0 {
			kinds[slot] = gcTouchSlotDPad
		}
	}
	return kinds
}

// readGCTouchSlot reads one finger's element from a controller's physical input profile.
func readGCTouchSlot(profile objc.ID, slot int, kind gcTouchSlotKind) gcTouchState {
	var st gcTouchState
	key := gcTouchSlotKey(slot)

	switch kind {
	case gcTouchSlotTouchpad:
		touchpad := profile.Send(sel_touchpads).Send(sel_objectForKeyedSubscript, key)
		if touchpad == 0 {
			return st
		}
		st.active = int(touchpad.Send(sel_touchState)) != gcTouchStateUp
		surface := touchpad.Send(sel_touchSurface)
		st.x = getAxisValue(surface.Send(sel_xAxis))
		st.y = getAxisValue(surface.Send(sel_yAxis))

	case gcTouchSlotDPad:
		dpad := profile.Send(sel_dpads).Send(sel_objectForKeyedSubscript, key)
		if dpad == 0 {
			return st
		}
		st.x = getAxisValue(dpad.Send(sel_xAxis))
		st.y = getAxisValue(dpad.Send(sel_yAxis))
		st.active = st.x != 0 || st.y != 0
	}
	return st
}

func (g *nativeGamepadGC) touchSurfaceCount() int {
	if len(g.touches) == 0 {
		return 0
	}
	return 1
}

func (g *nativeGamepadGC) touchSlotCount(surface int) int {
	if surface != 0 {
		return 0
	}
	return len(g.touches)
}

func (g *nativeGamepadGC) touchContactAt(surface, slot int) touchContact {
	if surface != 0 || slot < 0 || slot >= len(g.touches) {
		return touchContact{}
	}
	return g.touches[slot]
}
