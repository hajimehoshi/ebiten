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
	"sort"
	"strings"
)

const (
	screenWidth  = 320
	screenHeight = 240
)

// TODO: Add Key.String() by stringer

var keyNames = map[ebiten.Key]string{
	ebiten.KeyBackspace: "Backspace",
	ebiten.KeyComma:     "','",
	ebiten.KeyDelete:    "Delete",
	ebiten.KeyEnter:     "Enter",
	ebiten.KeyEscape:    "Escape",
	ebiten.KeyPeriod:    "'.'",
	ebiten.KeySpace:     "Space",
	ebiten.KeyTab:       "Tab",

	// Arrows
	ebiten.KeyDown:  "Down",
	ebiten.KeyLeft:  "Left",
	ebiten.KeyRight: "Right",
	ebiten.KeyUp:    "Up",

	// Mods
	ebiten.KeyLeftShift:   "Shift",
	ebiten.KeyLeftControl: "Ctrl",
	ebiten.KeyLeftAlt:     "Alt",
}

func update(screen *ebiten.Image) error {
	pressed := []string{}
	for i := 0; i <= 9; i++ {
		if ebiten.IsKeyPressed(ebiten.Key(i) + ebiten.Key0) {
			pressed = append(pressed, string(i+'0'))
		}
	}
	for c := 'A'; c <= 'Z'; c++ {
		if ebiten.IsKeyPressed(ebiten.Key(c) - 'A' + ebiten.KeyA) {
			pressed = append(pressed, string(c))
		}
	}
	for key, name := range keyNames {
		if ebiten.IsKeyPressed(key) {
			pressed = append(pressed, name)
		}
	}
	sort.Strings(pressed)
	str := "Pressed Keys: " + strings.Join(pressed, ", ")
	ebitenutil.DebugPrint(screen, str)
	return nil
}

func main() {
	if err := ebiten.Run(update, screenWidth, screenHeight, 2, "Keyboard (Ebiten Demo)"); err != nil {
		log.Fatal(err)
	}
}
