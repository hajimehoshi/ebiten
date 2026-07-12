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
// through the public ebiten API and renders green only if every value matched on every tick, so a
// correct round-trip is provable from the rendered screen.

package vmhost_test

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/exp/vmhost"
)

// fillGamepadState fills state with the snapshot the fixture guest expects, reusing state's slices and
// maps; keep it in sync with the fixture's expectation. The standard layer is set independently of the
// raw buttons, exercising the host-supplied standard mapping.
func fillGamepadState(state *vmhost.GamepadState) {
	state.ID = 0
	state.SDLID = "00000000000000000000000076543210"
	state.Name = "VM Test Controller"
	state.Axes = append(state.Axes[:0], 0.5, -0.5)
	state.Buttons = append(state.Buttons[:0], true, false, true)
	if state.StandardButtons == nil {
		state.StandardButtons = map[ebiten.StandardGamepadButton]vmhost.GamepadStandardButtonState{}
	}
	clear(state.StandardButtons)
	state.StandardButtons[ebiten.StandardGamepadButtonRightBottom] = vmhost.GamepadStandardButtonState{Pressed: true, Value: 1}
	state.StandardButtons[ebiten.StandardGamepadButtonLeftTop] = vmhost.GamepadStandardButtonState{Pressed: false, Value: 0.25}
	if state.StandardAxes == nil {
		state.StandardAxes = map[ebiten.StandardGamepadAxis]float64{}
	}
	clear(state.StandardAxes)
	state.StandardAxes[ebiten.StandardGamepadAxisLeftStickHorizontal] = 0.5
}

// checkScreenGreen fails the test unless the screen's center pixel is the fixture's all-matched green.
func checkScreenGreen(t *testing.T, screen *ebiten.Image) {
	t.Helper()
	b := screen.Bounds()
	pw, ph := b.Dx(), b.Dy()
	pixels := make([]byte, 4*pw*ph)
	screen.ReadPixels(pixels)
	i := 4 * ((ph/2)*pw + pw/2)
	r, g, bl, a := pixels[i], pixels[i+1], pixels[i+2], pixels[i+3]
	if r != 0x00 || g != 0xff || bl != 0x00 || a != 0xff {
		t.Errorf("screen center = (%d, %d, %d, %d); want (0, 255, 0, 255) — the guest reported a gamepad mismatch (see its log above)",
			r, g, bl, a)
	}
}

func TestGamepadForwarding(t *testing.T) {
	guest := startGuest(t, "./testdata/gamepad", activateByEnv, "unix")

	const w, h = 32, 32
	scale := ebiten.Monitor().DeviceScaleFactor()
	outsideScreen := ebiten.NewImage(int(w*scale), int(h*scale))
	if err := guest.SetOutsideScreen(outsideScreen); err != nil {
		t.Fatal(err)
	}

	states := make([]vmhost.GamepadState, 1)
	fillGamepadState(&states[0])
	guest.UpdateGamepads(states)

	// The injected state is queued before the tick, so the guest's Update observes it.
	tickAndFrame(t, guest)

	checkScreenGreen(t, outsideScreen)
}

// TestGamepadForwardingWithReusedBuffers proves the UpdateGamepads ownership contract: the snapshot
// is copied out before it returns, so the caller may scramble and refill the same buffers while
// earlier messages are still queued. Under -race it also proves the encode never reads them.
func TestGamepadForwardingWithReusedBuffers(t *testing.T) {
	guest := startGuest(t, "./testdata/gamepad", activateByEnv, "unix")

	const w, h = 32, 32
	scale := ebiten.Monitor().DeviceScaleFactor()
	outsideScreen := ebiten.NewImage(int(w*scale), int(h*scale))
	if err := guest.SetOutsideScreen(outsideScreen); err != nil {
		t.Fatal(err)
	}

	// One snapshot reused for every injection: each round refills the same slice, element, inner
	// slices, and maps, then scrambles them in place as soon as UpdateGamepads returns. The rounds are
	// posted much faster than the guest ticks, so scrambled buffers coexist with queued messages.
	states := make([]vmhost.GamepadState, 1)
	for range 20 {
		fillGamepadState(&states[0])
		guest.UpdateGamepads(states)
		guest.AdvanceTicks(1)
		scrambleGamepadState(&states[0])
	}

	guest.AdvanceFrame()
	if !guest.WaitFrame() {
		t.Fatalf("rendering the guest frame failed: %v", guest.Err())
	}
	if !guest.CompositeFrame() {
		t.Fatalf("compositing the guest frame failed: %v", guest.Err())
	}

	checkScreenGreen(t, outsideScreen)
}

// scrambleGamepadState overwrites state's slices and maps in place with values the fixture rejects,
// mutating the same backing storage the preceding UpdateGamepads was fed.
func scrambleGamepadState(state *vmhost.GamepadState) {
	for i := range state.Axes {
		state.Axes[i] = -123
	}
	for i := range state.Buttons {
		state.Buttons[i] = !state.Buttons[i]
	}
	clear(state.StandardAxes)
	state.StandardAxes[ebiten.StandardGamepadAxisRightStickVertical] = -1
	clear(state.StandardButtons)
	state.StandardButtons[ebiten.StandardGamepadButtonCenterCenter] = vmhost.GamepadStandardButtonState{Pressed: true, Value: 1}
}
