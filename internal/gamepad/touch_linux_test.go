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

// The DualSense touchpad node as the kernel registers it.
const (
	dualSenseTouchXMax = 1919
	dualSenseTouchYMax = 1079
)

var (
	dualSenseTouchKeys = []int{gamepad.BTNLeft, gamepad.BTNToolFinger, gamepad.BTNTouch, gamepad.BTNToolDoubleTap}
	dualSenseTouchAbs  = []int{gamepad.ABSX, gamepad.ABSY, gamepad.ABSMTSlot, gamepad.ABSMTPositionX, gamepad.ABSMTPositionY, gamepad.ABSMTTrackingID}
)

func TestClassifyEvdev(t *testing.T) {
	tests := []struct {
		name  string
		evs   []int
		keys  []int
		abs   []int
		props []int
		want  gamepad.EvdevKind
	}{
		{
			name: "gamepad node",
			evs:  []int{gamepad.EVKey, gamepad.EVAbs, gamepad.EVFF},
			keys: []int{gamepad.BTNSouth},
			abs:  []int{gamepad.ABSX, gamepad.ABSY, gamepad.ABSRZ, gamepad.ABSHat0X, gamepad.ABSHat0Y},
			want: gamepad.EvdevKindGamepad,
		},
		{
			name: "touchpad node",
			evs:  []int{gamepad.EVKey, gamepad.EVAbs},
			keys: dualSenseTouchKeys,
			abs:  dualSenseTouchAbs,
			want: gamepad.EvdevKindTouchSurface,
		},
		{
			name:  "motion sensors node",
			evs:   []int{gamepad.EVAbs},
			abs:   []int{gamepad.ABSX, gamepad.ABSY, gamepad.ABSRZ},
			props: []int{gamepad.InputPropAccelerometer},
			want:  gamepad.EvdevKindOther,
		},
		{
			name: "keyboard",
			evs:  []int{gamepad.EVKey},
			keys: []int{gamepad.BTNLeft},
			want: gamepad.EvdevKindOther,
		},
		{
			// A device with both gamepad buttons and multitouch axes is a gamepad.
			name: "gamepad with multitouch axes",
			evs:  []int{gamepad.EVKey, gamepad.EVAbs},
			keys: []int{gamepad.BTNSouth, gamepad.BTNTouch},
			abs:  dualSenseTouchAbs,
			want: gamepad.EvdevKindGamepad,
		},
		{
			// Multitouch needs the tracking id to tell contacts apart.
			name: "multitouch without tracking id",
			evs:  []int{gamepad.EVKey, gamepad.EVAbs},
			keys: dualSenseTouchKeys,
			abs:  []int{gamepad.ABSX, gamepad.ABSY, gamepad.ABSMTSlot, gamepad.ABSMTPositionX, gamepad.ABSMTPositionY},
			want: gamepad.EvdevKindGamepad,
		},
	}
	for _, test := range tests {
		if got := gamepad.ClassifyEvdevForTest(test.evs, test.keys, test.abs, test.props); got != test.want {
			t.Errorf("%s: classified as %d; want %d", test.name, got, test.want)
		}
	}
}

// The tests below feed a touch surface node the events a kernel sends, advance the gamepad the node
// belongs to, and read the result through the touch API, as a game does.

// newTouchGamepad returns a gamepad with a DualSense touchpad node behind it, and the node to feed
// events to.
func newTouchGamepad(t *testing.T, slots int) (*gamepad.Gamepad, *gamepad.TouchNode) {
	t.Helper()
	node := gamepad.NewTouchNodeForTest(slots, dualSenseTouchXMax, dualSenseTouchYMax)
	g := gamepad.NewLinuxGamepadForTest(node)
	g.UpdateForTest()
	if got := g.TouchSurfaceCount(); got != 1 {
		t.Fatalf("TouchSurfaceCount() = %d; want 1", got)
	}
	return g, node
}

// touchPos converts a DualSense touchpad position to the touch API's 0..1.
func touchPos(x, y int32) (float64, float64) {
	return float64(x) / dualSenseTouchXMax, float64(y) / dualSenseTouchYMax
}

// checkTouchPositionAt asserts that the touch is where the touchpad position (x, y) maps to.
func checkTouchPositionAt(t *testing.T, g *gamepad.Gamepad, id gamepad.TouchID, x, y int32) {
	t.Helper()
	wantX, wantY := touchPos(x, y)
	checkTouchPosition(t, g, id, wantX, wantY)
}

// findTouchAt returns the ID of the surface's touch at the touchpad position (x, y).
func findTouchAt(t *testing.T, g *gamepad.Gamepad, x, y int32) gamepad.TouchID {
	t.Helper()
	wantX, wantY := touchPos(x, y)
	return findTouchID(t, g, 0, wantX, wantY)
}

func TestTouchNodeOneFinger(t *testing.T) {
	g, node := newTouchGamepad(t, 2)
	checkTouchCount(t, g, 0, 0)

	// Finger down on slot 0, which is current from the start so no slot event precedes it.
	node.HandleAbsEventForTest(gamepad.ABSMTTrackingID, 4)
	node.HandleAbsEventForTest(gamepad.ABSMTPositionX, 1139)
	node.HandleAbsEventForTest(gamepad.ABSMTPositionY, 522)
	g.UpdateForTest()
	id := onlyTouchID(t, g, 0)
	checkTouchPositionAt(t, g, id, 1139, 522)

	// Movement is position events only, and the touch keeps its ID.
	node.HandleAbsEventForTest(gamepad.ABSMTPositionX, 1132)
	node.HandleAbsEventForTest(gamepad.ABSMTPositionY, 518)
	g.UpdateForTest()
	checkTouchKept(t, g, 0, id)
	checkTouchPositionAt(t, g, id, 1132, 518)

	// One axis can change alone.
	node.HandleAbsEventForTest(gamepad.ABSMTPositionY, 390)
	g.UpdateForTest()
	checkTouchKept(t, g, 0, id)
	checkTouchPositionAt(t, g, id, 1132, 390)

	// Lift is a tracking id of -1.
	node.HandleAbsEventForTest(gamepad.ABSMTTrackingID, -1)
	g.UpdateForTest()
	checkTouchCount(t, g, 0, 0)
	checkTouchRetired(t, g, 0, id)
}

func TestTouchNodeTwoFingers(t *testing.T) {
	g, node := newTouchGamepad(t, 2)

	// Finger A down on slot 0.
	node.HandleAbsEventForTest(gamepad.ABSMTTrackingID, 5)
	node.HandleAbsEventForTest(gamepad.ABSMTPositionX, 751)
	node.HandleAbsEventForTest(gamepad.ABSMTPositionY, 304)

	// Finger B down on slot 1.
	node.HandleAbsEventForTest(gamepad.ABSMTSlot, 1)
	node.HandleAbsEventForTest(gamepad.ABSMTTrackingID, 6)
	node.HandleAbsEventForTest(gamepad.ABSMTPositionX, 1573)
	node.HandleAbsEventForTest(gamepad.ABSMTPositionY, 698)
	g.UpdateForTest()

	// Both fingers are on the surface at once, so they have distinct IDs.
	checkTouchCount(t, g, 0, 2)
	a := findTouchAt(t, g, 751, 304)
	b := findTouchAt(t, g, 1573, 698)
	checkDistinctTouchIDs(t, a, b)

	// Reports alternate between the slots, each carrying only what changed. Both fingers keep their
	// IDs while they move.
	node.HandleAbsEventForTest(gamepad.ABSMTSlot, 0)
	node.HandleAbsEventForTest(gamepad.ABSMTPositionX, 752)
	node.HandleAbsEventForTest(gamepad.ABSMTSlot, 1)
	node.HandleAbsEventForTest(gamepad.ABSMTPositionX, 1569)
	node.HandleAbsEventForTest(gamepad.ABSMTPositionY, 701)
	g.UpdateForTest()
	checkTouchCount(t, g, 0, 2)
	checkTouchPositionAt(t, g, a, 752, 304)
	checkTouchPositionAt(t, g, b, 1569, 701)

	// B lifts first: its ID is retired, and A continues on slot 0 with its own.
	node.HandleAbsEventForTest(gamepad.ABSMTTrackingID, -1)
	node.HandleAbsEventForTest(gamepad.ABSMTSlot, 0)
	node.HandleAbsEventForTest(gamepad.ABSMTPositionX, 731)
	g.UpdateForTest()
	checkTouchKept(t, g, 0, a)
	checkTouchRetired(t, g, 0, b)
	checkTouchPositionAt(t, g, a, 731, 304)

	// B comes back on slot 1 as a new contact, which is a new touch.
	node.HandleAbsEventForTest(gamepad.ABSMTSlot, 1)
	node.HandleAbsEventForTest(gamepad.ABSMTTrackingID, 7)
	node.HandleAbsEventForTest(gamepad.ABSMTPositionX, 1671)
	node.HandleAbsEventForTest(gamepad.ABSMTPositionY, 107)
	g.UpdateForTest()
	checkTouchCount(t, g, 0, 2)
	b2 := findTouchAt(t, g, 1671, 107)
	checkDistinctTouchIDs(t, a, b, b2)

	// A lifts while B stays: B keeps its ID.
	node.HandleAbsEventForTest(gamepad.ABSMTSlot, 0)
	node.HandleAbsEventForTest(gamepad.ABSMTTrackingID, -1)
	node.HandleAbsEventForTest(gamepad.ABSMTSlot, 1)
	node.HandleAbsEventForTest(gamepad.ABSMTPositionX, 1558)
	g.UpdateForTest()
	checkTouchKept(t, g, 0, b2)
	checkTouchRetired(t, g, 0, a)
	checkTouchPositionAt(t, g, b2, 1558, 107)

	// B lifts last, on the current slot with no slot event.
	node.HandleAbsEventForTest(gamepad.ABSMTTrackingID, -1)
	g.UpdateForTest()
	checkTouchCount(t, g, 0, 0)
	checkTouchRetired(t, g, 0, b2)
}

// A finger lifted and put back in the same slot between two updates is a new touch, even though the
// slot is active at both updates.
func TestTouchNodeReleaseAndPressBetweenUpdates(t *testing.T) {
	g, node := newTouchGamepad(t, 2)

	node.HandleAbsEventForTest(gamepad.ABSMTTrackingID, 5)
	node.HandleAbsEventForTest(gamepad.ABSMTPositionX, 0)
	node.HandleAbsEventForTest(gamepad.ABSMTPositionY, dualSenseTouchYMax)
	g.UpdateForTest()
	first := onlyTouchID(t, g, 0)
	checkTouchPositionAt(t, g, first, 0, dualSenseTouchYMax)

	node.HandleAbsEventForTest(gamepad.ABSMTTrackingID, -1)
	node.HandleAbsEventForTest(gamepad.ABSMTTrackingID, 6)
	node.HandleAbsEventForTest(gamepad.ABSMTPositionX, dualSenseTouchXMax)
	node.HandleAbsEventForTest(gamepad.ABSMTPositionY, 0)
	g.UpdateForTest()
	second := onlyTouchID(t, g, 0)
	checkDistinctTouchIDs(t, first, second)
	checkTouchRetired(t, g, 0, first)
	checkTouchPositionAt(t, g, second, dualSenseTouchXMax, 0)
}

func TestTouchNodeIgnoresUnknownSlot(t *testing.T) {
	g, node := newTouchGamepad(t, 2)

	// Events for a slot the node did not report are dropped until a known slot is selected.
	node.HandleAbsEventForTest(gamepad.ABSMTSlot, 5)
	node.HandleAbsEventForTest(gamepad.ABSMTTrackingID, 9)
	node.HandleAbsEventForTest(gamepad.ABSMTPositionX, 100)
	g.UpdateForTest()
	checkTouchCount(t, g, 0, 0)

	node.HandleAbsEventForTest(gamepad.ABSMTSlot, 1)
	node.HandleAbsEventForTest(gamepad.ABSMTTrackingID, 9)
	node.HandleAbsEventForTest(gamepad.ABSMTPositionX, 100)
	node.HandleAbsEventForTest(gamepad.ABSMTPositionY, 200)
	g.UpdateForTest()
	id := onlyTouchID(t, g, 0)
	checkTouchPositionAt(t, g, id, 100, 200)
}

func TestTouchNodePositionRange(t *testing.T) {
	g, node := newTouchGamepad(t, 1)
	node.HandleAbsEventForTest(gamepad.ABSMTTrackingID, 0)

	tests := []struct {
		name         string
		x, y         int32
		wantX, wantY float64
	}{
		{name: "top left", x: 0, y: 0, wantX: 0, wantY: 0},
		{name: "top right", x: dualSenseTouchXMax, y: 0, wantX: 1, wantY: 0},
		{name: "bottom left", x: 0, y: dualSenseTouchYMax, wantX: 0, wantY: 1},
		{name: "bottom right", x: dualSenseTouchXMax, y: dualSenseTouchYMax, wantX: 1, wantY: 1},

		// A device reporting past the range it declared stays inside the range the touch API
		// documents.
		{name: "past the maxima", x: dualSenseTouchXMax + 1, y: dualSenseTouchYMax * 4, wantX: 1, wantY: 1},
		{name: "below the minima", x: -1, y: -30000, wantX: 0, wantY: 0},
	}
	for _, test := range tests {
		node.HandleAbsEventForTest(gamepad.ABSMTPositionX, test.x)
		node.HandleAbsEventForTest(gamepad.ABSMTPositionY, test.y)
		g.UpdateForTest()
		id := onlyTouchID(t, g, 0)
		if x, y := g.TouchPosition(id); x != test.wantX || y != test.wantY {
			t.Errorf("%s: position (%d, %d) = (%v, %v); want (%v, %v)", test.name, test.x, test.y, x, y, test.wantX, test.wantY)
		}
	}
}

// A node whose axes have an odd number of values has a sample on the center of its surface.
func TestTouchNodeCenterPosition(t *testing.T) {
	const (
		xMax = 1000
		yMax = 500
	)
	node := gamepad.NewTouchNodeForTest(1, xMax, yMax)
	g := gamepad.NewLinuxGamepadForTest(node)

	node.HandleAbsEventForTest(gamepad.ABSMTTrackingID, 0)
	node.HandleAbsEventForTest(gamepad.ABSMTPositionX, xMax/2)
	node.HandleAbsEventForTest(gamepad.ABSMTPositionY, yMax/2)
	g.UpdateForTest()

	checkTouchPosition(t, g, onlyTouchID(t, g, 0), 0.5, 0.5)
}

// A node whose axes have an empty range has no position to report, so the raw report does not leak
// through.
func TestTouchNodeEmptyPositionRange(t *testing.T) {
	node := gamepad.NewTouchNodeForTest(1, 0, 0)
	g := gamepad.NewLinuxGamepadForTest(node)
	node.HandleAbsEventForTest(gamepad.ABSMTTrackingID, 0)

	for _, v := range []int32{-5000, 0, 1, 5000} {
		node.HandleAbsEventForTest(gamepad.ABSMTPositionX, v)
		node.HandleAbsEventForTest(gamepad.ABSMTPositionY, v)
		g.UpdateForTest()
		id := onlyTouchID(t, g, 0)
		if x, y := g.TouchPosition(id); x != 0 || y != 0 {
			t.Errorf("position %d = (%v, %v); want (0, 0)", v, x, y)
		}
	}
}

// A gamepad whose touch surface node was never paired has no touch surface.
func TestLinuxGamepadWithoutTouchNode(t *testing.T) {
	g := gamepad.NewLinuxGamepadForTest(nil)
	g.UpdateForTest()
	if got := g.TouchSurfaceCount(); got != 0 {
		t.Errorf("TouchSurfaceCount() = %d; want 0", got)
	}
	checkTouchCount(t, g, 0, 0)
}
