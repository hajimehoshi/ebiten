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
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"
)

const (
	screenWidth  = 320
	screenHeight = 240
)

func update(screen *ebiten.Image) error {
	const gamepadID = 0
	presences := [4]bool{}
	axes := []string{}
	pressedButtons := []string{}

	for i := range presences {
		presences[i] = ebiten.IsGamepadPresent(i)
	}

	maxAxis := ebiten.GamepadAxisNum(gamepadID)
	for a := 0; a < maxAxis; a++ {
		v := ebiten.GamepadAxis(gamepadID, a)
		axes = append(axes, fmt.Sprintf("%d: %0.6f", a, v))
	}

	maxButton := ebiten.GamepadButton(ebiten.GamepadButtonNum(gamepadID))
	for b := ebiten.GamepadButton(gamepadID); b < maxButton; b++ {
		if ebiten.IsGamepadButtonPressed(gamepadID, b) {
			pressedButtons = append(pressedButtons, strconv.Itoa(int(b)))
		}
	}
	if ebiten.IsRunningSlowly() {
		return nil
	}

	ids := []string{}
	for i, p := range presences {
		if p {
			ids = append(ids, strconv.Itoa(i))
		}
	}

	str := `Gamepad ({{.GamepadIDs}})

Gamepad (ID: {{.GamepadID}}) status:
  Axes:
    {{.Axes}}
  Pressed Buttons: {{.Buttons}}`
	str = strings.Replace(str, "{{.GamepadIDs}}", strings.Join(ids, ","), -1)
	str = strings.Replace(str, "{{.GamepadID}}", strconv.Itoa(gamepadID), -1)
	str = strings.Replace(str, "{{.Axes}}", strings.Join(axes, "\n    "), -1)
	str = strings.Replace(str, "{{.Buttons}}", strings.Join(pressedButtons, ", "), -1)
	ebitenutil.DebugPrint(screen, str)
	return nil
}

func main() {
	if err := ebiten.Run(update, screenWidth, screenHeight, 2, "Gamepad (Ebiten Demo)"); err != nil {
		log.Fatal(err)
	}
}
