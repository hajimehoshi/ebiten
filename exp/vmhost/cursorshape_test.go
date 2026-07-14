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

// The guest keeps the default cursor shape, switches to a pointer at tick 3, and back to the default
// at tick 5 (see testdata/cursorshape). The host asserts CursorShape is the default before any tick
// and tracks each change afterward.

package vmhost_test

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestCursorShapeForwarding(t *testing.T) {
	guest := startGuest(t, "./testdata/cursorshape", activateByEnv, "unix")

	// AdvanceTicks requires a screen even though only the cursor shape is observed.
	scale := ebiten.Monitor().DeviceScaleFactor()
	if err := guest.SetOutsideScreen(ebiten.NewImage(int(64*scale), int(64*scale))); err != nil {
		t.Fatal(err)
	}

	// Before the guest reports anything, the host assumes the default shape.
	if shape := guest.CursorShape(); shape != ebiten.CursorShapeDefault {
		t.Errorf("CursorShape() before any tick = %d; want %d", shape, ebiten.CursorShapeDefault)
	}

	tests := []struct {
		tick int
		want ebiten.CursorShapeType
	}{
		{tick: 1, want: ebiten.CursorShapeDefault},
		{tick: 2, want: ebiten.CursorShapeDefault},
		{tick: 3, want: ebiten.CursorShapePointer},
		{tick: 4, want: ebiten.CursorShapePointer},
		{tick: 5, want: ebiten.CursorShapeDefault},
	}
	for _, tt := range tests {
		guest.AdvanceTicks(1)
		if !guest.WaitTicks() {
			t.Fatalf("waiting for tick %d failed: %v", tt.tick, guest.Err())
		}
		if got := guest.ProcessedTicks(); got != tt.tick {
			t.Errorf("ProcessedTicks() after tick %d = %d; want %d", tt.tick, got, tt.tick)
		}
		if shape := guest.CursorShape(); shape != tt.want {
			t.Errorf("CursorShape() after tick %d = %d; want %d", tt.tick, shape, tt.want)
		}
	}
}
