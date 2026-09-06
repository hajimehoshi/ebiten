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

//go:build ebitenginevmguest

// This guest reports through its screen whether the endpoint it was activated with is still in its
// environment: white when it is gone, red when it survives. A child process started with the default
// environment inherits exactly what this reports; see vmhost's endpoint-consumption test.
package main

import (
	"image/color"
	"log"
	"os"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

type game struct{}

func (game) Update() error {
	return nil
}

func (game) Draw(screen *ebiten.Image) {
	if endpointInherited() {
		screen.Fill(color.RGBA{R: 0xff, A: 0xff})
		return
	}
	screen.Fill(color.White)
}

func (game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return outsideWidth, outsideHeight
}

// endpointInherited reports whether the endpoint variable is still part of the environment a child
// process would inherit. os.Environ is what os/exec copies into a child by default.
func endpointInherited() bool {
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "EBITENGINE_VM_ENDPOINT=") {
			return true
		}
	}
	return false
}

func main() {
	if err := ebiten.RunGame(game{}); err != nil {
		log.Fatal(err)
	}
}
