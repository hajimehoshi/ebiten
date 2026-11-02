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

// End-to-end pinch and rotation forwarding: the host injects gestures, the guest reads the amounts back
// through the public ebiten API and renders green only if it matches the expectation for each tick,
// so a correct round-trip is provable from the rendered screen.

package vmhost_test

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestPinchAndRotationForwarding(t *testing.T) {
	guest := startGuest(t, "./testdata/pinch", activateByEnv, "unix")

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
			t.Errorf("%s: screen center = (%d, %d, %d, %d); want (0, 255, 0, 255) — the guest reported a gesture mismatch (see its log above)",
				phase, r, g, b, a)
		}
	}

	// The injected gestures mirror the fixture's per-tick expectation; keep the two in sync.

	// Tick 0: two gestures of each kind within one tick are composed.
	guest.PinchBy(0.1)
	guest.PinchBy(0.15)
	guest.RotateBy(0.25)
	guest.RotateBy(0.125)
	tickAndFrame(t, guest)
	assertGreen("after two gestures")

	// Tick 1: nothing is injected, so the amounts must be back to 0.
	tickAndFrame(t, guest)
	assertGreen("after no gesture")

	// Tick 2: a pinch that shrinks and a counterclockwise rotation.
	guest.PinchBy(-0.125)
	guest.RotateBy(-0.5)
	tickAndFrame(t, guest)
	assertGreen("after a shrinking and counterclockwise gesture")
}
