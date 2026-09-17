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
	"slices"
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

// touchPos converts a DualSense touchpad position to the touch API's -1..1.
func touchPos(x, y int32) (float64, float64) {
	return float64(x)/dualSenseTouchXMax*2 - 1, float64(y)/dualSenseTouchYMax*2 - 1
}

func checkContacts(t *testing.T, node *gamepad.TouchNode, want []gamepad.TouchContactForTest) {
	t.Helper()
	got := node.ContactsForTest()
	if len(got) != len(want) {
		t.Fatalf("contacts = %+v; want %+v", got, want)
	}
	for i := range got {
		// A lifted slot keeps its last position, which the API never reads; compare only what
		// matters for an inactive slot.
		if !want[i].Active {
			if got[i].Active {
				t.Errorf("slot %d = %+v; want inactive", i, got[i])
			}
			continue
		}
		if got[i] != want[i] {
			t.Errorf("slot %d = %+v; want %+v", i, got[i], want[i])
		}
	}
}

func TestTouchNodeOneFinger(t *testing.T) {
	node := gamepad.NewTouchNodeForTest(2, dualSenseTouchXMax, dualSenseTouchYMax)
	checkContacts(t, node, []gamepad.TouchContactForTest{{}, {}})

	// Finger down on slot 0, which is current from the start so no slot event precedes it.
	node.HandleAbsEventForTest(gamepad.ABSMTTrackingID, 4)
	node.HandleAbsEventForTest(gamepad.ABSMTPositionX, 1139)
	node.HandleAbsEventForTest(gamepad.ABSMTPositionY, 522)
	x, y := touchPos(1139, 522)
	checkContacts(t, node, []gamepad.TouchContactForTest{{Active: true, ID: 4, X: x, Y: y}, {}})

	// Movement is position events only.
	node.HandleAbsEventForTest(gamepad.ABSMTPositionX, 1132)
	node.HandleAbsEventForTest(gamepad.ABSMTPositionY, 518)
	x, y = touchPos(1132, 518)
	checkContacts(t, node, []gamepad.TouchContactForTest{{Active: true, ID: 4, X: x, Y: y}, {}})

	// One axis can change alone.
	node.HandleAbsEventForTest(gamepad.ABSMTPositionY, 390)
	x, y = touchPos(1132, 390)
	checkContacts(t, node, []gamepad.TouchContactForTest{{Active: true, ID: 4, X: x, Y: y}, {}})

	// Lift is a tracking id of -1; the position is not reset.
	node.HandleAbsEventForTest(gamepad.ABSMTTrackingID, -1)
	checkContacts(t, node, []gamepad.TouchContactForTest{{}, {}})
}

func TestTouchNodeTwoFingers(t *testing.T) {
	node := gamepad.NewTouchNodeForTest(2, dualSenseTouchXMax, dualSenseTouchYMax)

	// Finger A down on slot 0.
	node.HandleAbsEventForTest(gamepad.ABSMTTrackingID, 5)
	node.HandleAbsEventForTest(gamepad.ABSMTPositionX, 751)
	node.HandleAbsEventForTest(gamepad.ABSMTPositionY, 304)
	ax, ay := touchPos(751, 304)

	// Finger B down on slot 1.
	node.HandleAbsEventForTest(gamepad.ABSMTSlot, 1)
	node.HandleAbsEventForTest(gamepad.ABSMTTrackingID, 6)
	node.HandleAbsEventForTest(gamepad.ABSMTPositionX, 1573)
	node.HandleAbsEventForTest(gamepad.ABSMTPositionY, 698)
	bx, by := touchPos(1573, 698)
	checkContacts(t, node, []gamepad.TouchContactForTest{
		{Active: true, ID: 5, X: ax, Y: ay},
		{Active: true, ID: 6, X: bx, Y: by},
	})

	// Reports alternate between the slots, each carrying only what changed.
	node.HandleAbsEventForTest(gamepad.ABSMTSlot, 0)
	node.HandleAbsEventForTest(gamepad.ABSMTPositionX, 752)
	node.HandleAbsEventForTest(gamepad.ABSMTSlot, 1)
	node.HandleAbsEventForTest(gamepad.ABSMTPositionX, 1569)
	node.HandleAbsEventForTest(gamepad.ABSMTPositionY, 701)
	ax, ay = touchPos(752, 304)
	bx, by = touchPos(1569, 701)
	checkContacts(t, node, []gamepad.TouchContactForTest{
		{Active: true, ID: 5, X: ax, Y: ay},
		{Active: true, ID: 6, X: bx, Y: by},
	})

	// B lifts first: slot 1 ends, A continues on slot 0.
	node.HandleAbsEventForTest(gamepad.ABSMTTrackingID, -1)
	node.HandleAbsEventForTest(gamepad.ABSMTSlot, 0)
	node.HandleAbsEventForTest(gamepad.ABSMTPositionX, 731)
	ax, ay = touchPos(731, 304)
	checkContacts(t, node, []gamepad.TouchContactForTest{
		{Active: true, ID: 5, X: ax, Y: ay},
		{},
	})

	// B comes back on slot 1 as a new contact.
	node.HandleAbsEventForTest(gamepad.ABSMTSlot, 1)
	node.HandleAbsEventForTest(gamepad.ABSMTTrackingID, 7)
	node.HandleAbsEventForTest(gamepad.ABSMTPositionX, 1671)
	node.HandleAbsEventForTest(gamepad.ABSMTPositionY, 107)
	bx, by = touchPos(1671, 107)
	checkContacts(t, node, []gamepad.TouchContactForTest{
		{Active: true, ID: 5, X: ax, Y: ay},
		{Active: true, ID: 7, X: bx, Y: by},
	})

	// A lifts while B stays: B keeps slot 1 and its id.
	node.HandleAbsEventForTest(gamepad.ABSMTSlot, 0)
	node.HandleAbsEventForTest(gamepad.ABSMTTrackingID, -1)
	node.HandleAbsEventForTest(gamepad.ABSMTSlot, 1)
	node.HandleAbsEventForTest(gamepad.ABSMTPositionX, 1558)
	bx, by = touchPos(1558, 107)
	checkContacts(t, node, []gamepad.TouchContactForTest{
		{},
		{Active: true, ID: 7, X: bx, Y: by},
	})

	// B lifts last, on the current slot with no slot event.
	node.HandleAbsEventForTest(gamepad.ABSMTTrackingID, -1)
	checkContacts(t, node, []gamepad.TouchContactForTest{{}, {}})
}

func TestTouchNodeIgnoresUnknownSlot(t *testing.T) {
	node := gamepad.NewTouchNodeForTest(2, dualSenseTouchXMax, dualSenseTouchYMax)

	// Events for a slot the node did not report are dropped until a known slot is selected.
	node.HandleAbsEventForTest(gamepad.ABSMTSlot, 5)
	node.HandleAbsEventForTest(gamepad.ABSMTTrackingID, 9)
	node.HandleAbsEventForTest(gamepad.ABSMTPositionX, 100)
	checkContacts(t, node, []gamepad.TouchContactForTest{{}, {}})

	node.HandleAbsEventForTest(gamepad.ABSMTSlot, 1)
	node.HandleAbsEventForTest(gamepad.ABSMTTrackingID, 9)
	node.HandleAbsEventForTest(gamepad.ABSMTPositionX, 100)
	node.HandleAbsEventForTest(gamepad.ABSMTPositionY, 200)
	x, y := touchPos(100, 200)
	checkContacts(t, node, []gamepad.TouchContactForTest{{}, {Active: true, ID: 9, X: x, Y: y}})
}

func TestTouchNodePositionRange(t *testing.T) {
	node := gamepad.NewTouchNodeForTest(1, dualSenseTouchXMax, dualSenseTouchYMax)
	node.HandleAbsEventForTest(gamepad.ABSMTTrackingID, 0)

	tests := []struct {
		x, y         int32
		wantX, wantY float64
	}{
		{x: 0, y: 0, wantX: -1, wantY: -1},
		{x: dualSenseTouchXMax, y: dualSenseTouchYMax, wantX: 1, wantY: 1},
		{x: 0, y: dualSenseTouchYMax, wantX: -1, wantY: 1},

		// A device reporting past the range it declared stays inside the range the touch API
		// documents.
		{x: dualSenseTouchXMax + 1, y: dualSenseTouchYMax * 4, wantX: 1, wantY: 1},
		{x: -1, y: -30000, wantX: -1, wantY: -1},
	}
	for _, test := range tests {
		node.HandleAbsEventForTest(gamepad.ABSMTPositionX, test.x)
		node.HandleAbsEventForTest(gamepad.ABSMTPositionY, test.y)
		c := node.ContactsForTest()[0]
		if c.X != test.wantX || c.Y != test.wantY {
			t.Errorf("position (%d, %d) = (%v, %v); want (%v, %v)", test.x, test.y, c.X, c.Y, test.wantX, test.wantY)
		}
	}
}

// A node whose axes have an empty range has no position to report, so every value is the center
// rather than the raw report leaking through.
func TestTouchNodeEmptyPositionRange(t *testing.T) {
	node := gamepad.NewTouchNodeForTest(1, 0, 0)
	node.HandleAbsEventForTest(gamepad.ABSMTTrackingID, 0)

	for _, v := range []int32{-5000, 0, 1, 5000} {
		node.HandleAbsEventForTest(gamepad.ABSMTPositionX, v)
		node.HandleAbsEventForTest(gamepad.ABSMTPositionY, v)
		c := node.ContactsForTest()[0]
		if c.X != 0 || c.Y != 0 {
			t.Errorf("position %d = (%v, %v); want (0, 0)", v, c.X, c.Y)
		}
	}
}

func TestLinuxGamepadTouch(t *testing.T) {
	// A gamepad without a touch surface node.
	g := gamepad.NewLinuxGamepadForTest(nil)
	g.UpdateForTest()
	if got := g.TouchSurfaceCount(); got != 0 {
		t.Errorf("TouchSurfaceCount() = %d; want 0", got)
	}

	// A gamepad with its touch surface node paired: the slots reach the public touch IDs.
	node := gamepad.NewTouchNodeForTest(2, dualSenseTouchXMax, dualSenseTouchYMax)
	g = gamepad.NewLinuxGamepadForTest(node)
	g.UpdateForTest()
	if got := g.TouchSurfaceCount(); got != 1 {
		t.Fatalf("TouchSurfaceCount() = %d; want 1", got)
	}
	if got := g.AppendTouchIDs(0, nil); len(got) != 0 {
		t.Errorf("AppendTouchIDs(0) = %v; want none", got)
	}

	node.HandleAbsEventForTest(gamepad.ABSMTTrackingID, 5)
	node.HandleAbsEventForTest(gamepad.ABSMTPositionX, 0)
	node.HandleAbsEventForTest(gamepad.ABSMTPositionY, dualSenseTouchYMax)
	g.UpdateForTest()
	if got, want := g.AppendTouchIDs(0, nil), []gamepad.TouchID{1}; !slices.Equal(got, want) {
		t.Fatalf("AppendTouchIDs(0) = %v; want %v", got, want)
	}
	if x, y := g.TouchPosition(1); x != -1 || y != 1 {
		t.Errorf("TouchPosition(1) = (%v, %v); want (-1, 1)", x, y)
	}

	// The tracking id changing between two updates is a new touch, even though the slot stayed
	// active throughout.
	node.HandleAbsEventForTest(gamepad.ABSMTTrackingID, -1)
	node.HandleAbsEventForTest(gamepad.ABSMTTrackingID, 6)
	g.UpdateForTest()
	if got, want := g.AppendTouchIDs(0, nil), []gamepad.TouchID{2}; !slices.Equal(got, want) {
		t.Errorf("AppendTouchIDs(0) = %v; want %v", got, want)
	}

	node.HandleAbsEventForTest(gamepad.ABSMTSlot, 1)
	node.HandleAbsEventForTest(gamepad.ABSMTTrackingID, 7)
	g.UpdateForTest()
	if got, want := g.AppendTouchIDs(0, nil), []gamepad.TouchID{2, 3}; !slices.Equal(got, want) {
		t.Errorf("AppendTouchIDs(0) = %v; want %v", got, want)
	}

	node.HandleAbsEventForTest(gamepad.ABSMTSlot, 0)
	node.HandleAbsEventForTest(gamepad.ABSMTTrackingID, -1)
	g.UpdateForTest()
	if got, want := g.AppendTouchIDs(0, nil), []gamepad.TouchID{3}; !slices.Equal(got, want) {
		t.Errorf("AppendTouchIDs(0) = %v; want %v", got, want)
	}
}
