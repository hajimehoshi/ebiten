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

// TouchID identifies one touch on a gamepad's touch surface. It is unique among the gamepad's
// touches and stays the same while the finger is held; a lifted finger retires its ID and a new
// touch gets a new one. 0 is never a valid ID.
type TouchID int

// touchContact is a backend's report of one touch slot on one touch surface.
type touchContact struct {
	// active reports that a finger is on the slot.
	active bool

	// id tells consecutive contacts in the same slot apart when the device reports such a value: a
	// finger lifted and put back between two updates changes it while active stays true. It is
	// compared only for equality. A backend that cannot tell contacts apart reports 0.
	id int

	// x and y are the contact's position, each in 0..1, with (0, 0) at the top left of the surface
	// and (1, 1) at its bottom right.
	x, y float64
}

// nativeGamepadTouch is implemented by a backend whose gamepads can have touch surfaces, like the
// touchpad of a DualShock 4 or DualSense. Each surface has a fixed number of slots, one per finger
// the device can track at once. A backend whose gamepads have no touch surface need not implement
// it.
type nativeGamepadTouch interface {
	touchSurfaceCount() int
	touchSlotCount(surface int) int
	touchContactAt(surface, slot int) touchContact
}

// touchSlotTracker numbers the contacts of a surface's slots for a backend whose device does not
// number them but that observes each slot's transitions as they happen, such as through callbacks.
// A report that puts a finger on an empty slot begins a new contact and gives it the next number,
// so a finger lifted and put back between two updates is told apart from one held the whole time.
// The tracker does no locking; a backend whose reports arrive on another thread guards it.
type touchSlotTracker struct {
	slots    []touchContact
	contacts int
}

func newTouchSlotTracker(slotCount int) touchSlotTracker {
	return touchSlotTracker{
		slots: make([]touchContact, slotCount),
	}
}

// report records the slot's state as the backend observed it. It ignores a slot the tracker does
// not have.
func (t *touchSlotTracker) report(slot int, active bool, x, y float64) {
	if slot < 0 || slot >= len(t.slots) {
		return
	}
	c := &t.slots[slot]
	if active && !c.active {
		t.contacts++
		c.id = t.contacts
	}
	c.active = active
	c.x, c.y = x, y
}

// contactAt returns the slot's contact as last reported. A slot the tracker does not have is
// empty.
func (t *touchSlotTracker) contactAt(slot int) touchContact {
	if slot < 0 || slot >= len(t.slots) {
		return touchContact{}
	}
	return t.slots[slot]
}

// touch is the public view of one touch slot: its ID while a contact is active, or 0 when the slot
// is empty, and the contact's last report.
type touch struct {
	id      TouchID
	contact touchContact
}

// updateTouches derives the public touch state from the native slots after a native update,
// assigning an ID to each contact that started since the last update and retiring the IDs of the
// contacts that ended.
//
// updateTouches must be called with g.m held.
func (g *Gamepad) updateTouches() {
	n, ok := g.native.(nativeGamepadTouch)
	if !ok {
		g.touches = nil
		return
	}
	// A backend whose device reports its touch surfaces only after a change that affects other
	// applications reading the device makes the change only for a game that uses the touch API.
	// enableTouch is called at every update once the game has used the touch API on the gamepad, and
	// must do nothing once the touch surfaces are enabled.
	if e, ok := n.(interface{ enableTouch() }); ok && g.touchUsed {
		e.enableTouch()
	}

	surfaceCount := n.touchSurfaceCount()
	if len(g.touches) != surfaceCount {
		g.touches = make([][]touch, surfaceCount)
	}
	for s := range surfaceCount {
		slotCount := n.touchSlotCount(s)
		if len(g.touches[s]) != slotCount {
			g.touches[s] = make([]touch, slotCount)
		}
		for i := range slotCount {
			c := n.touchContactAt(s, i)
			t := &g.touches[s][i]
			if !c.active {
				t.id = 0
				continue
			}
			if t.id == 0 || t.contact.id != c.id {
				g.lastTouchID++
				t.id = g.lastTouchID
			}
			t.contact = c
		}
	}
}

// TouchSurfaceCount returns the number of the gamepad's touch surfaces.
//
// TouchSurfaceCount is concurrent-safe.
func (g *Gamepad) TouchSurfaceCount() int {
	g.m.Lock()
	defer g.m.Unlock()

	g.touchUsed = true
	return len(g.touches)
}

// AppendTouchIDs appends the IDs of the touches on the surface to ids and returns the extended
// slice. It appends nothing for a surface the gamepad does not have.
//
// AppendTouchIDs is concurrent-safe.
func (g *Gamepad) AppendTouchIDs(surface int, ids []TouchID) []TouchID {
	g.m.Lock()
	defer g.m.Unlock()

	g.touchUsed = true
	if surface < 0 || surface >= len(g.touches) {
		return ids
	}
	for _, t := range g.touches[surface] {
		if t.id != 0 {
			ids = append(ids, t.id)
		}
	}
	return ids
}

// TouchPosition returns the position of the touch, each coordinate in 0..1 with (0, 0) at the top
// left of its surface and (1, 1) at its bottom right, or (0, 0) if the gamepad has no such touch.
//
// TouchPosition is concurrent-safe.
func (g *Gamepad) TouchPosition(id TouchID) (x, y float64) {
	g.m.Lock()
	defer g.m.Unlock()

	g.touchUsed = true
	if id == 0 {
		return 0, 0
	}
	for _, surface := range g.touches {
		for _, t := range surface {
			if t.id == id {
				return t.contact.x, t.contact.y
			}
		}
	}
	return 0, 0
}
