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

// This is a guest that requests a gamepad vibration during each Update, with fixed magnitudes and
// duration, so the host can verify the guest→host vibration channel. It is launched by a host; see
// vmhost's vibrate test.
package main

import (
	"image/color"
	"log"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// The requested vibration mirrors what the host expects; keep the two in sync.
const (
	vibrateDuration = 500 * time.Millisecond
	vibrateStrong   = 0.25
	vibrateWeak     = 0.75
)

type game struct{}

func (g *game) Update() error {
	// Gamepad 0 is injected by the host before the tick, so it is connected here.
	ebiten.VibrateGamepad(0, &ebiten.VibrateGamepadOptions{
		Duration:        vibrateDuration,
		StrongMagnitude: vibrateStrong,
		WeakMagnitude:   vibrateWeak,
	})
	return nil
}

func (g *game) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{R: 0x00, G: 0xff, B: 0x00, A: 0xff})
}

func (g *game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return outsideWidth, outsideHeight
}

func main() {
	if err := ebiten.RunGame(&game{}); err != nil {
		log.Fatal(err)
	}
}
