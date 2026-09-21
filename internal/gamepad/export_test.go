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
	"github.com/hajimehoshi/ebiten/v2/internal/gamepaddb"
)

var MotorMagnitude = motorMagnitude

// nativeGamepadForTest is a gamepad backend whose reports a test supplies. Its axes and buttons
// reuse the virtual backend, and it adds the hats and touch surfaces that a virtual gamepad never
// has.
type nativeGamepadForTest struct {
	nativeGamepadVirtual

	hats     []int
	surfaces [][]TouchContactForTest

	// axisButtons is the gamepad's own standard layout: each standard button reads a raw axis, as an
	// analog trigger on a backend that reports triggers as axes does.
	axisButtons map[gamepaddb.StandardButton]int
}

func (g *nativeGamepadForTest) hasOwnStandardLayoutMapping() bool {
	return len(g.axisButtons) > 0 || g.nativeGamepadVirtual.hasOwnStandardLayoutMapping()
}

func (g *nativeGamepadForTest) standardButtonInOwnMapping(button gamepaddb.StandardButton) mappingInput {
	if a, ok := g.axisButtons[button]; ok {
		return axisMappingInput{g: g, axis: a}
	}
	return g.nativeGamepadVirtual.standardButtonInOwnMapping(button)
}

func (g *nativeGamepadForTest) hatCount() int {
	return len(g.hats)
}

func (g *nativeGamepadForTest) hatState(hat int) int {
	if hat < 0 || hat >= len(g.hats) {
		return hatCentered
	}
	return g.hats[hat]
}

func (g *nativeGamepadForTest) touchSurfaceCount() int {
	return len(g.surfaces)
}

func (g *nativeGamepadForTest) touchSlotCount(surface int) int {
	return len(g.surfaces[surface])
}

func (g *nativeGamepadForTest) touchContactAt(surface, slot int) touchContact {
	c := g.surfaces[surface][slot]
	return touchContact{
		active: c.Active,
		id:     c.ID,
		x:      c.X,
		y:      c.Y,
	}
}

// TouchContactForTest is one touch slot's report from the test backend: whether a finger is on the
// slot, the device's own contact identifier if it has one, and the position in -1..1.
type TouchContactForTest struct {
	Active bool
	ID     int
	X, Y   float64
}

// TouchSlotTracker is a touch slot tracker driven by a test's reports.
type TouchSlotTracker struct {
	t touchSlotTracker
}

func NewTouchSlotTrackerForTest(slotCount int) *TouchSlotTracker {
	return &TouchSlotTracker{
		t: newTouchSlotTracker(slotCount),
	}
}

// Report records one observation of a slot, as a backend's callback does.
func (t *TouchSlotTracker) Report(slot int, active bool, x, y float64) {
	t.t.report(slot, active, x, y)
}

// Contacts returns the tracker's slots as a backend hands them to the update.
func (t *TouchSlotTracker) Contacts() []TouchContactForTest {
	contacts := make([]TouchContactForTest, len(t.t.slots))
	for i := range contacts {
		c := t.t.contactAt(i)
		contacts[i] = TouchContactForTest{
			Active: c.active,
			ID:     c.id,
			X:      c.x,
			Y:      c.y,
		}
	}
	return contacts
}

// NewGamepadForTest returns a gamepad with the given SDL ID that takes its standard layout from
// gamepaddb, as a device does. The gamepad is not in the gamepad list, and its raw state is written
// with [Gamepad.SetReportForTest].
func NewGamepadForTest(sdlID string) *Gamepad {
	return &Gamepad{
		sdlID:  sdlID,
		native: &nativeGamepadForTest{},
	}
}

// NewGamepadWithAxisButtonsForTest returns a gamepad with the given SDL ID whose own standard layout
// reads each given standard button from a raw axis, as a device with analog triggers on axes does.
// The gamepad is not in the gamepad list, and its raw state is written with [Gamepad.SetReportForTest].
func NewGamepadWithAxisButtonsForTest(sdlID string, axisButtons map[gamepaddb.StandardButton]int) *Gamepad {
	return &Gamepad{
		sdlID: sdlID,
		native: &nativeGamepadForTest{
			axisButtons: axisButtons,
		},
	}
}

// SetReportForTest replaces the gamepad's raw state with one report, as an update from a device does.
func (g *Gamepad) SetReportForTest(axes []float64, buttons []bool, hats []int) {
	withNative(g, func(n *nativeGamepadForTest) {
		n.axes = append(n.axes[:0], axes...)
		n.buttons = append(n.buttons[:0], buttons...)
		n.hats = append(n.hats[:0], hats...)
	})
}

// SetTouchReportForTest replaces the gamepad's touch slots, indexed by surface and then slot, and
// runs the update that derives the touch IDs from them, as the gamepad list's update does for a
// device.
func (g *Gamepad) SetTouchReportForTest(surfaces [][]TouchContactForTest) {
	withNative(g, func(n *nativeGamepadForTest) {
		n.surfaces = surfaces
	})
	if err := g.update(nil); err != nil {
		panic(err)
	}
}
