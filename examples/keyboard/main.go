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
	"log"
	"strconv"

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"
	"github.com/hajimehoshi/ebiten/examples/keyboard/keyboard"
)

const (
	screenWidth  = 320
	screenHeight = 240
)

var keyboardImage *ebiten.Image

func init() {
	var err error
	keyboardImage, _, err = ebitenutil.NewImageFromFile("images/keyboard/keyboard.png", ebiten.FilterNearest)
	if err != nil {
		log.Fatal(err)
	}
}

var keyNames = map[ebiten.Key]string{
	ebiten.KeyBackspace: "BS",
	ebiten.KeyComma:     ",",
	ebiten.KeyDelete:    "Del",
	ebiten.KeyEnter:     "Enter",
	ebiten.KeyEscape:    "Esc",
	ebiten.KeyPeriod:    ".",
	ebiten.KeySpace:     "Space",
	ebiten.KeyTab:       "Tab",

	// Arrows
	ebiten.KeyDown:  "Down",
	ebiten.KeyLeft:  "Left",
	ebiten.KeyRight: "Right",
	ebiten.KeyUp:    "Up",

	// Mods
	ebiten.KeyShift:   "Shift",
	ebiten.KeyControl: "Ctrl",
	ebiten.KeyAlt:     "Alt",
}

type pressedKeysParts []string

func (p pressedKeysParts) Len() int {
	return len(p)
}

func (p pressedKeysParts) Dst(i int) (x0, y0, x1, y1 int) {
	k := p[i]
	r, ok := keyboard.KeyRect(k)
	if !ok {
		return 0, 0, 0, 0
	}
	return r.Min.X, r.Min.Y, r.Max.X, r.Max.Y
}

func (p pressedKeysParts) Src(i int) (x0, y0, x1, y1 int) {
	return p.Dst(i)
}

func update(screen *ebiten.Image) error {
	const offsetX, offsetY = 24, 40
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(offsetX, offsetY)
	op.ColorM.Scale(0.5, 0.5, 0.5, 1)
	screen.DrawImage(keyboardImage, op)

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
	for i := 1; i <= 12; i++ {
		if ebiten.IsKeyPressed(ebiten.Key(i) + ebiten.KeyF1 - 1) {
			pressed = append(pressed, "F"+strconv.Itoa(i))
		}
	}
	for key, name := range keyNames {
		if ebiten.IsKeyPressed(key) {
			pressed = append(pressed, name)
		}
	}

	op = &ebiten.DrawImageOptions{
		ImageParts: pressedKeysParts(pressed),
	}
	op.GeoM.Translate(offsetX, offsetY)
	screen.DrawImage(keyboardImage, op)

	return nil
}

func main() {
	if err := ebiten.Run(update, screenWidth, screenHeight, 2, "Keyboard (Ebiten Demo)"); err != nil {
		log.Fatal(err)
	}
}
