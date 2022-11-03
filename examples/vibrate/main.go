// Copyright 2021 The Ebiten Authors
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

package main

import (
	"log"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const (
	screenWidth  = 640
	screenHeight = 480
)

type Game struct {
	touchIDs     []ebiten.TouchID
	gamepadIDs   []ebiten.GamepadID
	touchCounter int
}

func (g *Game) Update() error {
	g.touchIDs = inpututil.AppendJustPressedTouchIDs(g.touchIDs[:0])
	if len(g.touchIDs) > 0 {
		g.touchCounter++
		op := &ebiten.VibrateOptions{
			Duration:  200 * time.Millisecond,
			Magnitude: 0.5*float64(g.touchCounter%2) + 0.5,
		}
		ebiten.Vibrate(op)
	}

	g.gamepadIDs = g.gamepadIDs[:0]
	g.gamepadIDs = ebiten.AppendGamepadIDs(g.gamepadIDs)
	for _, id := range g.gamepadIDs {
		for b := ebiten.GamepadButton0; b <= ebiten.GamepadButtonMax; b++ {
			if !inpututil.IsGamepadButtonJustPressed(id, b) {
				continue
			}
			// TODO: Test weak-magnitude.
			op := &ebiten.VibrateGamepadOptions{
				Duration:        200 * time.Millisecond,
				StrongMagnitude: 1,
				WeakMagnitude:   0,
			}
			ebiten.VibrateGamepad(id, op)
			break
		}
	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	msg := "Touch the screen to vibrate the screen."
	if len(g.gamepadIDs) > 0 {
		msg += "\nPress a gamepad button to vibrate the gamepad."
	}
	ebitenutil.DebugPrint(screen, msg)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	ebiten.SetWindowTitle("Vibrate (Ebitengine Demo)")
	if err := ebiten.RunGame(&Game{}); err != nil {
		log.Fatal(err)
	}
}
