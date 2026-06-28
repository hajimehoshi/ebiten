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

// End-to-end gamepad forwarding: the host injects a gamepad's full state, the guest reads it back
// through the public ebiten API and renders green only if every value matches, so a correct
// round-trip is provable from the rendered screen.

package vmhost_test

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/exp/vmhost"
)

func TestGamepadForwarding(t *testing.T) {
	guest := startGuest(t, "./testdata/gamepad", activateByEnv, "unix")

	const w, h = 32, 32
	scale := ebiten.Monitor().DeviceScaleFactor()
	pw, ph := int(w*scale), int(h*scale)
	outsideScreen := ebiten.NewImage(pw, ph)
	if err := guest.SetOutsideScreen(outsideScreen); err != nil {
		t.Fatal(err)
	}

	// The injected state mirrors the fixture's expectation; keep the two in sync. The standard layer is
	// set independently of the raw buttons, exercising the host-supplied standard mapping.
	guest.UpdateGamepads([]vmhost.GamepadState{
		{
			ID:      0,
			SDLID:   "00000000000000000000000076543210",
			Name:    "VM Test Controller",
			Axes:    []float64{0.5, -0.5},
			Buttons: []bool{true, false, true},
			StandardButtons: map[ebiten.StandardGamepadButton]vmhost.GamepadStandardButtonState{
				ebiten.StandardGamepadButtonRightBottom: {Pressed: true, Value: 1},
				ebiten.StandardGamepadButtonLeftTop:     {Pressed: false, Value: 0.25},
			},
			StandardAxes: map[ebiten.StandardGamepadAxis]float64{
				ebiten.StandardGamepadAxisLeftStickHorizontal: 0.5,
			},
		},
	})

	// The injected state is queued before the tick, so the guest's Update observes it.
	tickAndFrame(t, guest)

	pixels := make([]byte, 4*pw*ph)
	outsideScreen.ReadPixels(pixels)
	i := 4 * ((ph/2)*pw + pw/2)
	r, g, b, a := pixels[i], pixels[i+1], pixels[i+2], pixels[i+3]
	if r != 0x00 || g != 0xff || b != 0x00 || a != 0xff {
		t.Errorf("screen center = (%d, %d, %d, %d); want (0, 255, 0, 255) — the guest reported a gamepad mismatch (see its log above)",
			r, g, b, a)
	}
}
