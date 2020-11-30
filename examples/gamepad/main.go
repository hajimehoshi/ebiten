// Copyright 2015 Hajime Hoshi
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

// +build example

package main

import (
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const (
	screenWidth  = 640
	screenHeight = 480
)

type Game struct {
	gamepadIDs     map[ebiten.GamepadID]struct{}
	axes           map[ebiten.GamepadID][]string
	pressedButtons map[ebiten.GamepadID][]string
}

func (g *Game) Update() error {
	if g.gamepadIDs == nil {
		g.gamepadIDs = map[ebiten.GamepadID]struct{}{}
	}

	// Log the gamepad connection events.
	for _, id := range inpututil.JustConnectedGamepadIDs() {
		log.Printf("gamepad connected: id: %d", id)
		g.gamepadIDs[id] = struct{}{}
	}
	for id := range g.gamepadIDs {
		if inpututil.IsGamepadJustDisconnected(id) {
			log.Printf("gamepad disconnected: id: %d", id)
			delete(g.gamepadIDs, id)
		}
	}

	g.axes = map[ebiten.GamepadID][]string{}
	g.pressedButtons = map[ebiten.GamepadID][]string{}
	for id := range g.gamepadIDs {
		maxAxis := ebiten.GamepadAxisNum(id)
		for a := 0; a < maxAxis; a++ {
			v := ebiten.GamepadAxis(id, a)
			g.axes[id] = append(g.axes[id], fmt.Sprintf("%d:%0.2f", a, v))
		}
		maxButton := ebiten.GamepadButton(ebiten.GamepadButtonNum(id))
		for b := ebiten.GamepadButton(id); b < maxButton; b++ {
			if ebiten.IsGamepadButtonPressed(id, b) {
				g.pressedButtons[id] = append(g.pressedButtons[id], strconv.Itoa(int(b)))
			}

			// Log button events.
			if inpututil.IsGamepadButtonJustPressed(id, b) {
				log.Printf("button pressed: id: %d, button: %d", id, b)
			}
			if inpututil.IsGamepadButtonJustReleased(id, b) {
				log.Printf("button released: id: %d, button: %d", id, b)
			}
		}
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	// Draw the current gamepad status.
	str := ""
	if len(g.gamepadIDs) > 0 {
		ids := make([]ebiten.GamepadID, 0, len(g.gamepadIDs))
		for id := range g.gamepadIDs {
			ids = append(ids, id)
		}
		sort.Slice(ids, func(a, b int) bool {
			return ids[a] < ids[b]
		})
		for _, id := range ids {
			str += fmt.Sprintf("Gamepad (ID: %d, SDL ID: %s):\n", id, ebiten.GamepadSDLID(id))
			str += fmt.Sprintf("  Axes:    %s\n", strings.Join(g.axes[id], ", "))
			str += fmt.Sprintf("  Buttons: %s\n", strings.Join(g.pressedButtons[id], ", "))
			str += "\n"
		}
	} else {
		str = "Please connect your gamepad."
	}
	ebitenutil.DebugPrint(screen, str)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Gamepad (Ebiten Demo)")
	if err := ebiten.RunGame(&Game{}); err != nil {
		log.Fatal(err)
	}
}
