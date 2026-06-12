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

// End-to-end host rendering with the host as a normal ebiten application: launch the guest as a
// separate process, drive it over a socket, and replay its command stream through the ordinary ebiten
// API, then read the rendered pixels back. ebiten owns the driver and render thread, so there is no
// graphics-driver-layer access and no thread juggling here.

package vmhost_test

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	etesting "github.com/hajimehoshi/ebiten/v2/internal/testing"
)

func TestMain(m *testing.M) {
	etesting.MainWithRunLoop(m)
}

func TestHostRendersGuestFrame(t *testing.T) {
	// Everything between the endpoint and the pixels is transport-agnostic, so running the full
	// pipeline once per endpoint network covers both transports end-to-end.
	for _, network := range []string{"unix", "tcp"} {
		t.Run(network, func(t *testing.T) {
			testHostRendersGuestFrame(t, network)
		})
	}
}

func testHostRendersGuestFrame(t *testing.T, network string) {
	guest := startGuest(t, "./testdata/guest", activateByEnv, network)

	const w, h = 320, 240
	const spriteX, spriteY = 40, 40

	// The guest renders at the host's device scale factor, so the target screen is physical-sized while
	// the guest's outside size stays w x h device-independent pixels.
	scale := ebiten.Monitor().DeviceScaleFactor()
	pw, ph := int(w*scale), int(h*scale)
	outsideScreen := ebiten.NewImage(pw, ph)

	if err := guest.SetOutsideScreen(outsideScreen); err != nil {
		t.Fatal(err)
	}
	if err := guest.MoveCursor(spriteX, spriteY); err != nil {
		t.Fatal(err)
	}

	// Advance one tick (the guest sees the cursor), then render the frame into the host-owned screen.
	// This code runs inside the host's run loop (the test runs from the game's Update), so issuing
	// ebiten draws and reading pixels back is allowed.
	if err := guest.AdvanceTick(); err != nil {
		t.Fatalf("advancing a tick failed: %v", err)
	}
	guest.DrawFrame()
	// DrawFrame defers its errors to the next AdvanceTick.
	if err := guest.AdvanceTick(); err != nil {
		t.Fatalf("rendering the guest frame failed: %v", err)
	}

	pixels := make([]byte, 4*pw*ph)
	outsideScreen.ReadPixels(pixels)
	at := func(x, y int) (r, g, b, a uint8) {
		i := 4 * (y*pw + x)
		return pixels[i], pixels[i+1], pixels[i+2], pixels[i+3]
	}

	// Sample in physical pixels: the guest's logical coordinates are scaled by the device scale factor.
	bgX, bgY := int(8*scale), int(8*scale)
	spX, spY := int((spriteX+8)*scale), int((spriteY+8)*scale)
	fr8, fg8, fb8, _ := at(bgX, bgY)
	sr8, sg8, sb8, _ := at(spX, spY)
	t.Logf("fill(%d,%d)=(%d,%d,%d); sprite(%d,%d)=(%d,%d,%d)", bgX, bgY, fr8, fg8, fb8, spX, spY, sr8, sg8, sb8)

	if !(fb8 > fr8 && fb8 > 0x40) {
		t.Errorf("background fill is not bluish: (%d,%d,%d)", fr8, fg8, fb8)
	}
	if !(sr8 > 0xc0 && sg8 > 0xc0 && sb8 > 0xc0) {
		t.Errorf("sprite is not near-white: (%d,%d,%d)", sr8, sg8, sb8)
	}
}
