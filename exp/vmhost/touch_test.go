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

// End-to-end touch forwarding: the host injects a sequence of press, move, and release events, the
// guest reads the current touches back through the public ebiten API and renders green only if they
// match the expectation for each tick, so a correct round-trip is provable from the rendered screen.

package vmhost_test

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestTouchForwarding(t *testing.T) {
	guest := startGuest(t, "./testdata/touch", activateByEnv, "unix")

	const w, h = 32, 32
	scale := ebiten.Monitor().DeviceScaleFactor()
	pw, ph := int(w*scale), int(h*scale)
	outsideScreen := ebiten.NewImage(pw, ph)
	if err := guest.SetOutsideScreen(outsideScreen); err != nil {
		t.Fatal(err)
	}

	pixels := make([]byte, 4*pw*ph)
	assertGreen := func(phase string) {
		t.Helper()
		outsideScreen.ReadPixels(pixels)
		i := 4 * ((ph/2)*pw + pw/2)
		r, g, b, a := pixels[i], pixels[i+1], pixels[i+2], pixels[i+3]
		if r != 0x00 || g != 0xff || b != 0x00 || a != 0xff {
			t.Errorf("%s: screen center = (%d, %d, %d, %d); want (0, 255, 0, 255) — the guest reported a touch mismatch (see its log above)",
				phase, r, g, b, a)
		}
	}

	// The injected events mirror the fixture's per-tick expectation; keep the two in sync. The positions
	// are in outside-screen device-independent pixels, within the 32x32 outside size.

	// Tick 0: two touches begin.
	guest.PressTouch(1, 3, 4)
	guest.PressTouch(2, 30, 20)
	tickAndFrame(t, guest)
	assertGreen("after press")

	// Tick 1: touch 1 moves, touch 2 ends.
	guest.MoveTouch(1, 5, 6)
	guest.ReleaseTouch(2)
	tickAndFrame(t, guest)
	assertGreen("after move and release")
}
