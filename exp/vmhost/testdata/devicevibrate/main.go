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

// This is a guest that requests a device vibration during each Update, with a fixed magnitude and
// duration, so the host can verify the guest→host device-vibration channel. It is launched by a host;
// see vmhost's device-vibration test.
package main

import (
	"image/color"
	"log"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// The requested vibration mirrors what the host expects; keep the two in sync.
const (
	vibrateDuration  = 250 * time.Millisecond
	vibrateMagnitude = 0.5
)

type game struct{}

func (g *game) Update() error {
	ebiten.Vibrate(&ebiten.VibrateOptions{
		Duration:  vibrateDuration,
		Magnitude: vibrateMagnitude,
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
