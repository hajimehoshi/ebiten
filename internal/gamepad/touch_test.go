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

package gamepad_test

import (
	"slices"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/internal/gamepad"
)

// The tests below never assume which IDs the gamepad hands out or the order it lists them in. They
// take the IDs from the API and check what it guarantees: touches on a surface at the same time
// have distinct IDs, a moving finger keeps its ID, a lifted finger's ID is retired, and a finger put
// down afterward gets one that was never used before.

// touchIDs returns the touch IDs on the surface, so that tests compare slices rather than nil
// against an empty result.
func touchIDs(g *gamepad.Gamepad, surface int) []gamepad.TouchID {
	return g.AppendTouchIDs(surface, []gamepad.TouchID{})
}

// checkTouchCount asserts how many touches the surface has.
func checkTouchCount(t *testing.T, g *gamepad.Gamepad, surface, want int) {
	t.Helper()
	if got := touchIDs(g, surface); len(got) != want {
		t.Fatalf("AppendTouchIDs(%d) = %v; want %d touches", surface, got, want)
	}
}

// onlyTouchID returns the ID of the surface's one touch.
func onlyTouchID(t *testing.T, g *gamepad.Gamepad, surface int) gamepad.TouchID {
	t.Helper()
	ids := touchIDs(g, surface)
	if len(ids) != 1 {
		t.Fatalf("AppendTouchIDs(%d) = %v; want one touch", surface, ids)
	}
	return ids[0]
}

// checkTouchKept asserts that the surface's one touch is still the given one.
func checkTouchKept(t *testing.T, g *gamepad.Gamepad, surface int, id gamepad.TouchID) {
	t.Helper()
	if got := onlyTouchID(t, g, surface); got != id {
		t.Fatalf("the touch on surface %d has ID %d; want the ID %d it had before", surface, got, id)
	}
}

// findTouchID returns the ID of the surface's touch at the position, without assuming the order the
// touches are listed in.
func findTouchID(t *testing.T, g *gamepad.Gamepad, surface int, x, y float64) gamepad.TouchID {
	t.Helper()
	for _, id := range touchIDs(g, surface) {
		if tx, ty := g.TouchPosition(id); tx == x && ty == y {
			return id
		}
	}
	t.Fatalf("surface %d has no touch at (%v, %v); its touches are %v", surface, x, y, touchIDs(g, surface))
	return 0
}

// checkDistinctTouchIDs asserts that the IDs are all different and that none of them is 0, which is
// never a valid ID.
func checkDistinctTouchIDs(t *testing.T, ids ...gamepad.TouchID) {
	t.Helper()
	seen := map[gamepad.TouchID]int{}
	for i, id := range ids {
		if id == 0 {
			t.Errorf("touch %d has the ID 0", i)
			continue
		}
		if j, ok := seen[id]; ok {
			t.Errorf("touches %d and %d share the ID %d", j, i, id)
			continue
		}
		seen[id] = i
	}
}

// checkTouchRetired asserts that the touch is gone: the surface no longer lists it and it has no
// position.
func checkTouchRetired(t *testing.T, g *gamepad.Gamepad, surface int, id gamepad.TouchID) {
	t.Helper()
	if slices.Contains(touchIDs(g, surface), id) {
		t.Errorf("touch %d is still on surface %d", id, surface)
	}
	checkTouchPosition(t, g, id, 0, 0)
}

func checkTouchPosition(t *testing.T, g *gamepad.Gamepad, id gamepad.TouchID, wantX, wantY float64) {
	t.Helper()
	if x, y := g.TouchPosition(id); x != wantX || y != wantY {
		t.Errorf("TouchPosition(%d) = (%v, %v); want (%v, %v)", id, x, y, wantX, wantY)
	}
}

func TestTouchWithoutSurface(t *testing.T) {
	g := gamepad.NewGamepadForTest("")
	g.SetTouchReportForTest(nil)

	if got := g.TouchSurfaceCount(); got != 0 {
		t.Errorf("TouchSurfaceCount() = %d; want 0", got)
	}
	if got := touchIDs(g, 0); len(got) != 0 {
		t.Errorf("AppendTouchIDs(0) = %v; want none", got)
	}
	checkTouchPosition(t, g, 0, 0, 0)
	checkTouchPosition(t, g, 1, 0, 0)
}

func TestTouchIDsFollowContacts(t *testing.T) {
	g := gamepad.NewGamepadForTest("")

	// An empty surface has no touches but still counts.
	g.SetTouchReportForTest([][]gamepad.TouchContactForTest{
		{{}, {}},
	})
	if got := g.TouchSurfaceCount(); got != 1 {
		t.Errorf("TouchSurfaceCount() = %d; want 1", got)
	}
	checkTouchCount(t, g, 0, 0)

	// A finger down in slot 0 is a touch.
	g.SetTouchReportForTest([][]gamepad.TouchContactForTest{
		{{Active: true, X: 0.25, Y: 0.625}, {}},
	})
	first := onlyTouchID(t, g, 0)
	checkTouchPosition(t, g, first, 0.25, 0.625)

	// The finger moving keeps its ID and updates its position.
	g.SetTouchReportForTest([][]gamepad.TouchContactForTest{
		{{Active: true, X: 0.5, Y: 0.75}, {}},
	})
	checkTouchKept(t, g, 0, first)
	checkTouchPosition(t, g, first, 0.5, 0.75)

	// A second finger is another touch; the first keeps its own ID.
	g.SetTouchReportForTest([][]gamepad.TouchContactForTest{
		{{Active: true, X: 0.5, Y: 0.75}, {Active: true, X: 1, Y: 0}},
	})
	checkTouchCount(t, g, 0, 2)
	second := findTouchID(t, g, 0, 1, 0)
	checkDistinctTouchIDs(t, first, second)
	checkTouchPosition(t, g, first, 0.5, 0.75)

	// Lifting the first finger retires its ID and its position, and leaves the second alone.
	g.SetTouchReportForTest([][]gamepad.TouchContactForTest{
		{{}, {Active: true, X: 1, Y: 0}},
	})
	checkTouchKept(t, g, 0, second)
	checkTouchRetired(t, g, 0, first)
	checkTouchPosition(t, g, second, 1, 0)

	// A finger back in slot 0 is a new touch, never the retired one.
	g.SetTouchReportForTest([][]gamepad.TouchContactForTest{
		{{Active: true, X: 0, Y: 0.5}, {Active: true, X: 1, Y: 0}},
	})
	checkTouchCount(t, g, 0, 2)
	third := findTouchID(t, g, 0, 0, 0.5)
	checkDistinctTouchIDs(t, first, second, third)

	// Everything lifted.
	g.SetTouchReportForTest([][]gamepad.TouchContactForTest{
		{{}, {}},
	})
	checkTouchCount(t, g, 0, 0)
	for _, id := range []gamepad.TouchID{first, second, third} {
		checkTouchRetired(t, g, 0, id)
	}
}

func TestTouchContactIDChangeIsNewTouch(t *testing.T) {
	g := gamepad.NewGamepadForTest("")

	// The device numbers its contacts, so a finger lifted and put back between two updates shows up
	// as the slot staying active with a different contact ID.
	g.SetTouchReportForTest([][]gamepad.TouchContactForTest{
		{{Active: true, ID: 7, X: 0.1, Y: 0.2}},
	})
	first := onlyTouchID(t, g, 0)

	g.SetTouchReportForTest([][]gamepad.TouchContactForTest{
		{{Active: true, ID: 7, X: 0.3, Y: 0.4}},
	})
	checkTouchKept(t, g, 0, first)

	g.SetTouchReportForTest([][]gamepad.TouchContactForTest{
		{{Active: true, ID: 8, X: 0.3, Y: 0.4}},
	})
	second := onlyTouchID(t, g, 0)
	checkDistinctTouchIDs(t, first, second)
	checkTouchRetired(t, g, 0, first)
	checkTouchPosition(t, g, second, 0.3, 0.4)
}

func TestTouchSlotTrackerTellsContactsApart(t *testing.T) {
	g := gamepad.NewGamepadWithTrackedTouchForTest(2)

	// A finger down between two updates is a touch at the next one.
	g.ReportTouchForTest(0, true, 0.25, 0.625)
	g.UpdateForTest()
	first := onlyTouchID(t, g, 0)
	checkTouchPosition(t, g, first, 0.25, 0.625)

	// Movement within one contact, however many reports of it arrive between updates, keeps the ID
	// and updates the position.
	g.ReportTouchForTest(0, true, 0, 0)
	g.ReportTouchForTest(0, true, 0.5, 0.75)
	g.UpdateForTest()
	checkTouchKept(t, g, 0, first)
	checkTouchPosition(t, g, first, 0.5, 0.75)

	// The finger lifted and another put down in the same slot between two updates is a new touch
	// with a new ID, even though the slot is active at both updates.
	g.ReportTouchForTest(0, false, 0.5, 0.75)
	g.ReportTouchForTest(0, true, 1, 1)
	g.UpdateForTest()
	second := onlyTouchID(t, g, 0)
	checkDistinctTouchIDs(t, first, second)
	checkTouchRetired(t, g, 0, first)
	checkTouchPosition(t, g, second, 1, 1)

	// A contact in another slot does not disturb the first slot's contact.
	g.ReportTouchForTest(1, true, 0, 1)
	g.UpdateForTest()
	checkTouchCount(t, g, 0, 2)
	third := findTouchID(t, g, 0, 0, 1)
	checkDistinctTouchIDs(t, first, second, third)
	checkTouchPosition(t, g, second, 1, 1)

	// A finger lifted between two updates retires its ID; one lifted and put back in the same slot
	// is a new touch.
	g.ReportTouchForTest(0, false, 1, 1)
	g.ReportTouchForTest(1, false, 0, 1)
	g.ReportTouchForTest(1, true, 0.5, 0.5)
	g.UpdateForTest()
	fourth := onlyTouchID(t, g, 0)
	checkDistinctTouchIDs(t, first, second, third, fourth)
	checkTouchRetired(t, g, 0, second)
	checkTouchRetired(t, g, 0, third)
	checkTouchPosition(t, g, fourth, 0.5, 0.5)

	// A slot the tracker does not have is ignored.
	g.ReportTouchForTest(2, true, 0, 0)
	g.ReportTouchForTest(-1, true, 0, 0)
	g.UpdateForTest()
	checkTouchKept(t, g, 0, fourth)
}

func TestTouchSurfaces(t *testing.T) {
	g := gamepad.NewGamepadForTest("")
	g.SetTouchReportForTest([][]gamepad.TouchContactForTest{
		{{Active: true, X: 0.25, Y: 0.75}},
		{{}, {Active: true, X: 1, Y: 1}},
	})

	if got := g.TouchSurfaceCount(); got != 2 {
		t.Errorf("TouchSurfaceCount() = %d; want 2", got)
	}

	// IDs are unique across the gamepad's surfaces, and each surface lists only its own touches.
	first := onlyTouchID(t, g, 0)
	second := onlyTouchID(t, g, 1)
	checkDistinctTouchIDs(t, first, second)
	checkTouchPosition(t, g, first, 0.25, 0.75)
	checkTouchPosition(t, g, second, 1, 1)

	// A surface the gamepad does not have appends nothing and leaves the buffer as it was.
	for _, surface := range []int{-1, 2} {
		buf := []gamepad.TouchID{9}
		if got, want := g.AppendTouchIDs(surface, buf), []gamepad.TouchID{9}; !slices.Equal(got, want) {
			t.Errorf("AppendTouchIDs(%d) = %v; want %v", surface, got, want)
		}
	}
}

func TestTouchSlotLayoutChange(t *testing.T) {
	g := gamepad.NewGamepadForTest("")
	g.SetTouchReportForTest([][]gamepad.TouchContactForTest{
		{{Active: true, X: 0.25, Y: 0.25}, {Active: true, X: 0.75, Y: 0.75}},
	})
	checkTouchCount(t, g, 0, 2)
	before := touchIDs(g, 0)
	checkDistinctTouchIDs(t, before...)

	// A backend reporting a different layout starts over, and the IDs it hands out afterward are
	// still fresh.
	g.SetTouchReportForTest([][]gamepad.TouchContactForTest{
		{{Active: true, X: 0.25, Y: 0.25}},
		{{Active: true, X: 0.75, Y: 0.75}},
	})
	after := append(touchIDs(g, 0), touchIDs(g, 1)...)
	checkTouchCount(t, g, 0, 1)
	checkTouchCount(t, g, 1, 1)
	checkDistinctTouchIDs(t, append(before, after...)...)
}

func TestVirtualGamepadHasNoTouchSurface(t *testing.T) {
	updateVirtualGamepads(t, []gamepad.VirtualGamepadState{
		{ID: 0, Name: "Pad", Buttons: []bool{true}},
	})
	defer updateVirtualGamepads(t, []gamepad.VirtualGamepadState{})

	g := gamepad.Get(0)
	if g == nil {
		t.Fatal("gamepad 0 is not connected")
	}
	if got := g.TouchSurfaceCount(); got != 0 {
		t.Errorf("TouchSurfaceCount() = %d; want 0", got)
	}
	if got := touchIDs(g, 0); len(got) != 0 {
		t.Errorf("AppendTouchIDs(0) = %v; want none", got)
	}
	checkTouchPosition(t, g, 1, 0, 0)
}
