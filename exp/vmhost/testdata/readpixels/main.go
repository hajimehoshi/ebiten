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

// This is a guest that exercises the virtualization ReadPixels round-trip. During
// Update it fills an offscreen image with a known color and reads it back; because the guest has no
// GPU, the read-back is served by the host process mid-tick. It then paints the whole screen with the
// bytes it read, so a correct round-trip is observable in the rendered screen: the screen color
// equals the fill color only if the guest received the host's pixels.
//
// It is launched by a host; see vmhost's ReadPixels test.
package main

import (
	"image/color"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
)

// fillColor is the opaque color the offscreen is filled with. Opaque so it survives the screen
// composite byte-for-byte (premultiplied alpha is a no-op).
var fillColor = color.RGBA{R: 0x12, G: 0x34, B: 0x56, A: 0xff}

type game struct {
	offscreen *ebiten.Image
	readback  color.RGBA
	haveRead  bool
}

func (g *game) Update() error {
	if g.offscreen == nil {
		g.offscreen = ebiten.NewImage(4, 4)
	}
	g.offscreen.Fill(fillColor)

	// Read the offscreen back. The guest has no GPU, so this blocks on a host render + read-back.
	pixels := make([]byte, 4*4*4)
	g.offscreen.ReadPixels(pixels)
	g.readback = color.RGBA{R: pixels[0], G: pixels[1], B: pixels[2], A: pixels[3]}
	g.haveRead = true
	return nil
}

func (g *game) Draw(screen *ebiten.Image) {
	if g.haveRead {
		screen.Fill(g.readback)
	}
}

func (g *game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return outsideWidth, outsideHeight
}

func main() {
	if err := ebiten.RunGame(&game{}); err != nil {
		log.Fatal(err)
	}
}
