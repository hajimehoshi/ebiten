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
	screenWidth  = 640
	screenHeight = 480
)

func update(screen *ebiten.Image) error {
	const maxGamepadNum = 4
	presences := [maxGamepadNum]bool{}
	axes := [maxGamepadNum][]string{}
	pressedButtons := [maxGamepadNum][]string{}

	for i := range presences {
		presences[i] = ebiten.IsGamepadPresent(i)
	}

	for i := range axes {
		maxAxis := ebiten.GamepadAxisNum(i)
		for a := 0; a < maxAxis; a++ {
			v := ebiten.GamepadAxis(i, a)
			axes[i] = append(axes[i], fmt.Sprintf("%d:%0.2f", a, v))
		}
	}

	for i := range pressedButtons {
		maxButton := ebiten.GamepadButton(ebiten.GamepadButtonNum(i))
		for b := ebiten.GamepadButton(i); b < maxButton; b++ {
			if ebiten.IsGamepadButtonPressed(i, b) {
				pressedButtons[i] = append(pressedButtons[i], strconv.Itoa(int(b)))
			}
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

	str := fmt.Sprintf("Gamepad (%s)\n", strings.Join(ids, ","))
	str += "\n"

	for i, p := range presences {
		if !p {
			continue
		}
		str += fmt.Sprintf("Gamepad (ID: %d):\n", i)
		str += fmt.Sprintf("  Axes:    %s\n", strings.Join(axes[i], ", "))
		str += fmt.Sprintf("  Buttons: %s\n", strings.Join(pressedButtons[i], ", "))
		str += "\n"
	}
	ebitenutil.DebugPrint(screen, str)
	return nil
}

func main() {
	if err := ebiten.Run(update, screenWidth, screenHeight, 1, "Gamepad (Ebiten Demo)"); err != nil {
		log.Fatal(err)
	}
}
