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

// The guest uses ordinary (atlas-packed) images: several distinct-colored images share a backend at
// different offsets, drawn at known positions. A mishandled packed source offset would make a draw
// sample a neighbor's pixels, so verifying each color at its position proves the atlas offsets
// round-trip to the host exactly.

package vmhost_test

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestAtlasPackedImages(t *testing.T) {
	guest := startGuest(t, "./testdata/atlas", activateByEnv, "unix")

	const w, h = 128, 128

	// The guest renders at the host's device scale factor, so the target screen is physical-sized.
	scale := ebiten.Monitor().DeviceScaleFactor()
	pw, ph := int(w*scale), int(h*scale)
	outsideScreen := ebiten.NewImage(pw, ph)

	if err := guest.SetOutsideScreen(outsideScreen); err != nil {
		t.Fatal(err)
	}
	tickAndFrame(t, guest)

	pixels := make([]byte, 4*pw*ph)
	outsideScreen.ReadPixels(pixels)
	at := func(x, y int) (r, g, b, a uint8) {
		i := 4 * (y*pw + x)
		return pixels[i], pixels[i+1], pixels[i+2], pixels[i+3]
	}

	// Sample the interior (+8) of each 16x16 tile at the positions the guest drew them, scaled to
	// physical pixels by the device scale factor. The colors match testdata/atlas.
	checks := []struct {
		name    string
		x, y    int
		r, g, b uint8
	}{
		{"red", 10, 10, 0xff, 0x00, 0x00},
		{"green", 40, 10, 0x00, 0xff, 0x00},
		{"blue", 10, 40, 0x00, 0x00, 0xff},
		{"yellow", 40, 40, 0xff, 0xff, 0x00},
		{"magenta", 70, 10, 0xff, 0x00, 0xff},
		{"cyan", 70, 40, 0x00, 0xff, 0xff},
	}
	for _, c := range checks {
		x := int(float64(c.x+8) * scale)
		y := int(float64(c.y+8) * scale)
		r, g, b, a := at(x, y)
		if r != c.r || g != c.g || b != c.b || a != 0xff {
			t.Errorf("%s tile at (%d,%d) = (%d,%d,%d,%d); want (%d,%d,%d,255) — atlas offset mishandled",
				c.name, x, y, r, g, b, a, c.r, c.g, c.b)
		}
	}
}
