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

// touchIDs returns the touch IDs on the surface, so that tests compare slices rather than nil
// against an empty result.
func touchIDs(g *gamepad.Gamepad, surface int) []gamepad.TouchID {
	return g.AppendTouchIDs(surface, []gamepad.TouchID{})
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
	if got := touchIDs(g, 0); len(got) != 0 {
		t.Errorf("AppendTouchIDs(0) = %v; want none", got)
	}

	// A finger down in slot 0 gets the first ID.
	g.SetTouchReportForTest([][]gamepad.TouchContactForTest{
		{{Active: true, X: -0.5, Y: 0.25}, {}},
	})
	if got, want := touchIDs(g, 0), []gamepad.TouchID{1}; !slices.Equal(got, want) {
		t.Fatalf("AppendTouchIDs(0) = %v; want %v", got, want)
	}
	checkTouchPosition(t, g, 1, -0.5, 0.25)

	// The finger moving keeps its ID and updates its position.
	g.SetTouchReportForTest([][]gamepad.TouchContactForTest{
		{{Active: true, X: 0.5, Y: 0.75}, {}},
	})
	if got, want := touchIDs(g, 0), []gamepad.TouchID{1}; !slices.Equal(got, want) {
		t.Fatalf("AppendTouchIDs(0) = %v; want %v", got, want)
	}
	checkTouchPosition(t, g, 1, 0.5, 0.75)

	// A second finger gets the next ID; the first keeps its own.
	g.SetTouchReportForTest([][]gamepad.TouchContactForTest{
		{{Active: true, X: 0.5, Y: 0.75}, {Active: true, X: 1, Y: -1}},
	})
	if got, want := touchIDs(g, 0), []gamepad.TouchID{1, 2}; !slices.Equal(got, want) {
		t.Fatalf("AppendTouchIDs(0) = %v; want %v", got, want)
	}
	checkTouchPosition(t, g, 1, 0.5, 0.75)
	checkTouchPosition(t, g, 2, 1, -1)

	// Lifting the first finger retires its ID and its position.
	g.SetTouchReportForTest([][]gamepad.TouchContactForTest{
		{{}, {Active: true, X: 1, Y: -1}},
	})
	if got, want := touchIDs(g, 0), []gamepad.TouchID{2}; !slices.Equal(got, want) {
		t.Fatalf("AppendTouchIDs(0) = %v; want %v", got, want)
	}
	checkTouchPosition(t, g, 1, 0, 0)
	checkTouchPosition(t, g, 2, 1, -1)

	// A finger back in slot 0 is a new touch with a new ID, never a reused one.
	g.SetTouchReportForTest([][]gamepad.TouchContactForTest{
		{{Active: true}, {Active: true, X: 1, Y: -1}},
	})
	if got, want := touchIDs(g, 0), []gamepad.TouchID{3, 2}; !slices.Equal(got, want) {
		t.Fatalf("AppendTouchIDs(0) = %v; want %v", got, want)
	}

	// Everything lifted.
	g.SetTouchReportForTest([][]gamepad.TouchContactForTest{
		{{}, {}},
	})
	if got := touchIDs(g, 0); len(got) != 0 {
		t.Errorf("AppendTouchIDs(0) = %v; want none", got)
	}
	for id := gamepad.TouchID(1); id <= 3; id++ {
		checkTouchPosition(t, g, id, 0, 0)
	}
}

func TestTouchContactIDChangeIsNewTouch(t *testing.T) {
	g := gamepad.NewGamepadForTest("")

	// The device numbers its contacts, so a finger lifted and put back between two updates shows up
	// as the slot staying active with a different contact ID.
	g.SetTouchReportForTest([][]gamepad.TouchContactForTest{
		{{Active: true, ID: 7, X: 0.1, Y: 0.2}},
	})
	if got, want := touchIDs(g, 0), []gamepad.TouchID{1}; !slices.Equal(got, want) {
		t.Fatalf("AppendTouchIDs(0) = %v; want %v", got, want)
	}

	g.SetTouchReportForTest([][]gamepad.TouchContactForTest{
		{{Active: true, ID: 7, X: 0.3, Y: 0.4}},
	})
	if got, want := touchIDs(g, 0), []gamepad.TouchID{1}; !slices.Equal(got, want) {
		t.Fatalf("AppendTouchIDs(0) = %v; want %v", got, want)
	}

	g.SetTouchReportForTest([][]gamepad.TouchContactForTest{
		{{Active: true, ID: 8, X: 0.3, Y: 0.4}},
	})
	if got, want := touchIDs(g, 0), []gamepad.TouchID{2}; !slices.Equal(got, want) {
		t.Fatalf("AppendTouchIDs(0) = %v; want %v", got, want)
	}
	checkTouchPosition(t, g, 1, 0, 0)
	checkTouchPosition(t, g, 2, 0.3, 0.4)
}

func TestTouchSurfaces(t *testing.T) {
	g := gamepad.NewGamepadForTest("")
	g.SetTouchReportForTest([][]gamepad.TouchContactForTest{
		{{Active: true, X: -1, Y: -1}},
		{{}, {Active: true, X: 1, Y: 1}},
	})

	if got := g.TouchSurfaceCount(); got != 2 {
		t.Errorf("TouchSurfaceCount() = %d; want 2", got)
	}

	// IDs are unique across the gamepad's surfaces, and each surface lists only its own touches.
	if got, want := touchIDs(g, 0), []gamepad.TouchID{1}; !slices.Equal(got, want) {
		t.Errorf("AppendTouchIDs(0) = %v; want %v", got, want)
	}
	if got, want := touchIDs(g, 1), []gamepad.TouchID{2}; !slices.Equal(got, want) {
		t.Errorf("AppendTouchIDs(1) = %v; want %v", got, want)
	}
	checkTouchPosition(t, g, 1, -1, -1)
	checkTouchPosition(t, g, 2, 1, 1)

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
		{{Active: true}, {Active: true}},
	})
	if got, want := touchIDs(g, 0), []gamepad.TouchID{1, 2}; !slices.Equal(got, want) {
		t.Fatalf("AppendTouchIDs(0) = %v; want %v", got, want)
	}

	// A backend reporting a different layout starts over, and the IDs it hands out afterward are
	// still fresh.
	g.SetTouchReportForTest([][]gamepad.TouchContactForTest{
		{{Active: true}},
		{{Active: true}},
	})
	if got, want := touchIDs(g, 0), []gamepad.TouchID{3}; !slices.Equal(got, want) {
		t.Errorf("AppendTouchIDs(0) = %v; want %v", got, want)
	}
	if got, want := touchIDs(g, 1), []gamepad.TouchID{4}; !slices.Equal(got, want) {
		t.Errorf("AppendTouchIDs(1) = %v; want %v", got, want)
	}
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
