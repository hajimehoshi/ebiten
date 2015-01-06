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

// TODO: Add Key.String()

var keys = []ebiten.Key{
	ebiten.Key0, ebiten.Key1, ebiten.Key2, ebiten.Key3, ebiten.Key4,
	ebiten.Key5, ebiten.Key6, ebiten.Key7, ebiten.Key8, ebiten.Key9,
	ebiten.KeyA, ebiten.KeyB, ebiten.KeyC, ebiten.KeyD, ebiten.KeyE,
	ebiten.KeyF, ebiten.KeyG, ebiten.KeyH, ebiten.KeyI, ebiten.KeyJ,
	ebiten.KeyK, ebiten.KeyL, ebiten.KeyM, ebiten.KeyN, ebiten.KeyO,
	ebiten.KeyP, ebiten.KeyQ, ebiten.KeyR, ebiten.KeyS, ebiten.KeyT,
	ebiten.KeyU, ebiten.KeyV, ebiten.KeyW, ebiten.KeyX, ebiten.KeyY,
	ebiten.KeyZ,
}

func update(screen *ebiten.Image) error {
	pressed := []string{}
	for _, key := range keys {
		if ebiten.IsKeyPressed(key) {
			pressed = append(pressed, strconv.Itoa(int(key)))
		}
	}
	str := "Pressed Keys: " + strings.Join(pressed, ",")
	ebitenutil.DebugPrint(screen, str)
	return nil
}

func main() {
	if err := ebiten.Run(update, screenWidth, screenHeight, 2, "Keyboard (Ebiten Demo)"); err != nil {
		log.Fatal(err)
	}
}
