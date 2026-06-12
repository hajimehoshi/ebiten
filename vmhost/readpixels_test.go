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

// End-to-end ReadPixels round-trip: the guest has no GPU, so when it reads an offscreen image back
// mid-tick the host renders the image on demand and returns the pixels. The guest paints the screen
// with what it read, so a correct round-trip is provable from the rendered screen.

package vmhost_test

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestReadPixelsRoundTrip(t *testing.T) {
	guest := startGuest(t, "./testdata/readpixels", activateByEnv, "unix")

	// These calls run inside the host's run loop (the test runs from the game's Update), so issuing
	// ebiten draws and reading pixels back is allowed.
	const w, h = 64, 48

	// The guest renders at the host's device scale factor, so the target screen is physical-sized.
	scale := ebiten.Monitor().DeviceScaleFactor()
	pw, ph := int(w*scale), int(h*scale)
	outsideScreen := ebiten.NewImage(pw, ph)

	if err := guest.SetOutsideScreen(outsideScreen); err != nil {
		t.Fatal(err)
	}

	// The round-trip happens inside this tick's Update; the host serves it on demand.
	if err := guest.AdvanceTick(); err != nil {
		t.Fatalf("advancing a tick failed: %v", err)
	}
	guest.AdvanceFrame()
	// AdvanceFrame defers its errors to the next AdvanceTick.
	if err := guest.AdvanceTick(); err != nil {
		t.Fatalf("rendering the guest frame failed: %v", err)
	}

	// The guest filled its offscreen with (0x12, 0x34, 0x56, 0xff), read it back through the host, and
	// painted the screen with exactly those bytes. Recover them from the screen.
	pixels := make([]byte, 4*pw*ph)
	outsideScreen.ReadPixels(pixels)
	i := 4 * ((ph/2)*pw + pw/2)
	r, g8, b, a := pixels[i], pixels[i+1], pixels[i+2], pixels[i+3]
	t.Logf("screen center = (%d, %d, %d, %d)", r, g8, b, a)
	if r != 0x12 || g8 != 0x34 || b != 0x56 || a != 0xff {
		t.Errorf("screen center = (%d, %d, %d, %d); want (0x12, 0x34, 0x56, 0xff) — ReadPixels did not round-trip",
			r, g8, b, a)
	}
}
