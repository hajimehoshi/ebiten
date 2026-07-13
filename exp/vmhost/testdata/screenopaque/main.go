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

// This guest does not request a transparent screen, so the host's CompositeFrame composites its frame
// over opaque black; see vmhost's screen-transparent test.
package main

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"
)

type game struct{}

func (game) Update() error {
	return nil
}

func (game) Draw(screen *ebiten.Image) {
}

func (game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return outsideWidth, outsideHeight
}

func main() {
	if err := ebiten.RunGameWithOptions(game{}, &ebiten.RunGameOptions{
		ScreenTransparent: false,
	}); err != nil {
		log.Fatal(err)
	}
}
