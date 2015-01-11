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

package main

import (
	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"
	"log"
	"strconv"
	"strings"
)

const (
	screenWidth  = 320
	screenHeight = 240
)

func update(screen *ebiten.Image) error {
	pressed := []string{}
	for b := ebiten.GamepadButton(0); b < 16; b++ {
		if ebiten.IsGamepadButtonPressed(0, b) {
			pressed = append(pressed, strconv.Itoa(int(b)))
		}
	}
	str := "Pressed Keys: " + strings.Join(pressed, ", ")
	ebitenutil.DebugPrint(screen, str)
	return nil
}

func main() {
	if err := ebiten.Run(update, screenWidth, screenHeight, 2, "Gamepad (Ebiten Demo)"); err != nil {
		log.Fatal(err)
	}
}
