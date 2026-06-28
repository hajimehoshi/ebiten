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

//go:build ebitenginevm

// This is a guest that verifies the gamepad state the host forwards. During Update it reads the
// gamepad through the public ebiten API and compares it against the fixed expectation the host
// injects; it fills the screen green when everything matches and red otherwise (logging each
// mismatch), so the outcome is observable in the rendered screen.
//
// It is launched by a host; see vmhost's gamepad test.
package main

import (
	"image/color"
	"log"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// The expected state mirrors what the host injects; keep the two in sync.
const (
	wantSDLID = "00000000000000000000000076543210"
	wantName  = "VM Test Controller"
)

var (
	wantAxes    = []float64{0.5, -0.5}
	wantButtons = []bool{true, false, true}
)

type game struct {
	ok bool
}

func (g *game) Update() error {
	g.ok = g.check()
	return nil
}

func (g *game) check() bool {
	ids := ebiten.AppendGamepadIDs(nil)
	if len(ids) != 1 || ids[0] != 0 {
		log.Printf("gamepad IDs = %v; want [0]", ids)
		return false
	}
	const id = 0

	ok := true
	fail := func(format string, args ...any) {
		log.Printf(format, args...)
		ok = false
	}

	if got := ebiten.GamepadSDLID(id); got != wantSDLID {
		fail("GamepadSDLID = %q; want %q", got, wantSDLID)
	}
	if got := ebiten.GamepadName(id); got != wantName {
		fail("GamepadName = %q; want %q", got, wantName)
	}

	if got := ebiten.GamepadAxisCount(id); got != len(wantAxes) {
		fail("GamepadAxisCount = %d; want %d", got, len(wantAxes))
	} else {
		for a, want := range wantAxes {
			if got := ebiten.GamepadAxisValue(id, a); !approx(got, want) {
				fail("GamepadAxisValue(%d) = %v; want %v", a, got, want)
			}
		}
	}

	if got := ebiten.GamepadButtonCount(id); got != len(wantButtons) {
		fail("GamepadButtonCount = %d; want %d", got, len(wantButtons))
	} else {
		for b, want := range wantButtons {
			if got := ebiten.IsGamepadButtonPressed(id, ebiten.GamepadButton(b)); got != want {
				fail("IsGamepadButtonPressed(%d) = %v; want %v", b, got, want)
			}
		}
	}

	if !ebiten.IsStandardGamepadLayoutAvailable(id) {
		fail("IsStandardGamepadLayoutAvailable = false; want true")
	}
	if got := ebiten.IsStandardGamepadButtonPressed(id, ebiten.StandardGamepadButtonRightBottom); !got {
		fail("IsStandardGamepadButtonPressed(RightBottom) = false; want true")
	}
	if got := ebiten.StandardGamepadButtonValue(id, ebiten.StandardGamepadButtonLeftTop); !approx(got, 0.25) {
		fail("StandardGamepadButtonValue(LeftTop) = %v; want 0.25", got)
	}
	if ebiten.IsStandardGamepadButtonAvailable(id, ebiten.StandardGamepadButtonRightTop) {
		fail("IsStandardGamepadButtonAvailable(RightTop) = true; want false")
	}
	if got := ebiten.StandardGamepadAxisValue(id, ebiten.StandardGamepadAxisLeftStickHorizontal); !approx(got, 0.5) {
		fail("StandardGamepadAxisValue(LeftStickHorizontal) = %v; want 0.5", got)
	}

	return ok
}

func approx(a, b float64) bool {
	return math.Abs(a-b) < 1e-6
}

func (g *game) Draw(screen *ebiten.Image) {
	if g.ok {
		screen.Fill(color.RGBA{R: 0x00, G: 0xff, B: 0x00, A: 0xff})
		return
	}
	screen.Fill(color.RGBA{R: 0xff, G: 0x00, B: 0x00, A: 0xff})
}

func (g *game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return outsideWidth, outsideHeight
}

func main() {
	if err := ebiten.RunGame(&game{}); err != nil {
		log.Fatal(err)
	}
}
