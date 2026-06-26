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

// This is a guest fixture activated through RunGameOptions.VMGuestEndpoint. It deliberately has no
// ebitenginevm build tag: setting the option in code must run the game as a guest in any build. The
// endpoint arrives as a command-line argument so that EBITENGINE_VM_ENDPOINT plays no part. It fills
// the screen with a fixed color for the test to assert.
package main

import (
	"image/color"
	"log"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
)

type game struct{}

func (game) Update() error {
	return nil
}

func (game) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{R: 0x80, G: 0x20, B: 0x60, A: 0xff})
}

func (game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return outsideWidth, outsideHeight
}

func main() {
	if len(os.Args) < 2 {
		log.Fatal("usage: options <endpoint>")
	}
	if err := ebiten.RunGameWithOptions(game{}, &ebiten.RunGameOptions{
		VMGuestEndpoint: os.Args[1],
	}); err != nil {
		log.Fatal(err)
	}
}
