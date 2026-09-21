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

	"github.com/hajimehoshi/ebiten/v2/internal/cocoa"
)

// gcTouchSlotMax is the number of fingers the GameController framework tracks on a touch surface:
// the DualShock 4 and DualSense profiles have a primary and a secondary finger element.
const gcTouchSlotMax = 2

// gcTouchSlotKind is where a finger's element was found in a controller's physical input profile.
type gcTouchSlotKind uint8

const (
	gcTouchSlotNone gcTouchSlotKind = iota

	// gcTouchSlotTouchpad is a GCControllerTouchpad from the profile's touchpads. It carries a touch
	// state along with the finger's position, and reports a finger's touch down, movement, and
	// touch up through separate handlers.
	gcTouchSlotTouchpad

	// gcTouchSlotDPad is a GCControllerDirectionPad from the profile's dpads. It carries only the
	// finger's position, which is the origin while no finger is on the surface, and reports every
	// change of it through one handler. The DualShock 4 and DualSense expose their fingers this
	// way; a finger cannot rest exactly on the origin, as their touch grids have an even number of
	// columns.
	gcTouchSlotDPad
)

// gcTouchStateUp is the GCTouchState value of GCControllerTouchpad.touchState with no finger on the
// surface; the other values are 1 while a finger has just touched down and 2 while it moves.
const gcTouchStateUp = 0

// gcTouchState is one finger's report, read from its element or delivered to one of its handlers.
// x and y are as the framework reports them, with positive y pointing up.
type gcTouchState struct {
	active bool
	x, y   float32
}

// gcTouchHandler is one handler block set on a finger element, with the setter that clears it.
type gcTouchHandler struct {
	setter objc.SEL
	block  objc.Block
}

// gcTouchElement is a finger element with the handlers set on it. The element is retained.
type gcTouchElement struct {
	element  objc.ID
	handlers []gcTouchHandler
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

// gcTouchSlotElement returns one finger's element from a controller's physical input profile, or
// 0 if the profile no longer has it.
func gcTouchSlotElement(profile objc.ID, slot int, kind gcTouchSlotKind) objc.ID {
	key := gcTouchSlotKey(slot)
	switch kind {
	case gcTouchSlotTouchpad:
		return profile.Send(sel_touchpads).Send(sel_objectForKeyedSubscript, key)
	case gcTouchSlotDPad:
		return profile.Send(sel_dpads).Send(sel_objectForKeyedSubscript, key)
	}
	return 0
}

// readGCTouchSlot reads one finger's current state from its element.
func readGCTouchSlot(element objc.ID, kind gcTouchSlotKind) gcTouchState {
	var st gcTouchState

	switch kind {
	case gcTouchSlotTouchpad:
		// Without the surface there is no position to report, so the finger is not reported
		// either: a touch at the center of the surface is not what is happening.
		surface := element.Send(sel_touchSurface)
		if surface == 0 {
			return st
		}
		st.active = int(element.Send(sel_touchState)) != gcTouchStateUp
		st.x = getAxisValue(surface.Send(sel_xAxis))
		st.y = getAxisValue(surface.Send(sel_yAxis))

	case gcTouchSlotDPad:
		st.x = getAxisValue(element.Send(sel_xAxis))
		st.y = getAxisValue(element.Send(sel_yAxis))
		st.active = st.x != 0 || st.y != 0
	}
	return st
}

// startTouchTracking sets handlers on the finger elements of g's touch surface that report each
// finger's transitions to g's touch tracker as the framework observes them, and seeds the tracker
// with the fingers already on the surface. The elements' polled values alone cannot tell a finger
// lifted and put back between two updates from one held the whole time; the handlers see both
// transitions, so the tracker numbers the second finger as a new contact.
//
// The handlers run on the framework's handler queue until stopTouchTracking clears them.
func (g *nativeGamepadGC) startTouchTracking() {
	n := gcTouchSlotCount(g.touchSlots)
	g.touchTracker = newTouchSlotTracker(n)
	if n == 0 {
		return
	}

	// The profile getters return autoreleased objects, and the gamepad update does not run inside an
	// autorelease pool. The pool is safe here only because the update goroutine is locked to an OS
	// thread.
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	profile := objc.ID(g.controller).Send(sel_physicalInputProfile)
	if profile == 0 {
		return
	}

	for slot := range n {
		element := gcTouchSlotElement(profile, slot, g.touchSlots[slot])
		if element == 0 {
			continue
		}
		element.Send(sel_retain)
		e := gcTouchElement{element: element}

		switch g.touchSlots[slot] {
		case gcTouchSlotTouchpad:
			// The touchpad handlers share one signature; each reports the touch state its event
			// implies, ignoring the touchpad button's value and pressed state.
			touchpadHandler := func(active bool) objc.Block {
				return objc.NewBlock(func(_ objc.Block, _ objc.ID, x, y, _ float32, _ bool) {
					g.reportTouch(slot, gcTouchState{active: active, x: x, y: y})
				})
			}
			e.handlers = []gcTouchHandler{
				{setter: sel_setTouchDown, block: touchpadHandler(true)},
				{setter: sel_setTouchMoved, block: touchpadHandler(true)},
				{setter: sel_setTouchUp, block: touchpadHandler(false)},
			}

		case gcTouchSlotDPad:
			valueChanged := objc.NewBlock(func(_ objc.Block, _ objc.ID, x, y float32) {
				g.reportTouch(slot, gcTouchState{active: x != 0 || y != 0, x: x, y: y})
			})
			e.handlers = []gcTouchHandler{
				{setter: sel_setValueChangedHandler, block: valueChanged},
			}
		}

		for _, h := range e.handlers {
			element.Send(h.setter, h.block)
		}
		g.touchElements = append(g.touchElements, e)

		// A finger already on the surface changes nothing until it moves, so it is read once the
		// handlers are in place. A handler that ran first has the newer state and is kept.
		g.touchMu.Lock()
		if !g.touchTracker.contactAt(slot).active {
			g.recordTouch(slot, readGCTouchSlot(element, g.touchSlots[slot]))
		}
		g.touchMu.Unlock()
	}
}

// stopTouchTracking clears the handlers set by startTouchTracking and releases the elements.
func (g *nativeGamepadGC) stopTouchTracking() {
	for _, e := range g.touchElements {
		for _, h := range e.handlers {
			e.element.Send(h.setter, uintptr(0))
			h.block.Release()
		}
		e.element.Send(sel_release)
	}
	g.touchElements = nil
}

// reportTouch records one finger's report from its handler.
func (g *nativeGamepadGC) reportTouch(slot int, st gcTouchState) {
	g.touchMu.Lock()
	defer g.touchMu.Unlock()

	g.recordTouch(slot, st)
}

// recordTouch records one finger's report in the tracker. The framework's positive y points up; the
// touch API has -1 at the top of the surface.
//
// recordTouch must be called with g.touchMu held.
func (g *nativeGamepadGC) recordTouch(slot int, st gcTouchState) {
	g.touchTracker.report(slot, st.active, float64(st.x), -float64(st.y))
}

// updateTouches takes the update's snapshot of the tracked fingers.
func (g *nativeGamepadGC) updateTouches() {
	g.touchMu.Lock()
	defer g.touchMu.Unlock()

	for i := range g.touches {
		g.touches[i] = g.touchTracker.contactAt(i)
	}
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
