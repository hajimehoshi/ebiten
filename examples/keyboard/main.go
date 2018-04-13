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

// +build example jsgo

package main

import (
	"bytes"
	"image"
	_ "image/png"
	"log"
	"strings"

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"
	"github.com/hajimehoshi/ebiten/examples/keyboard/keyboard"
	rkeyabord "github.com/hajimehoshi/ebiten/examples/resources/images/keyboard"
)

const (
	screenWidth  = 320
	screenHeight = 240
)

var keyboardImage *ebiten.Image

func init() {
	img, _, err := image.Decode(bytes.NewReader(rkeyabord.Keyboard_png))
	if err != nil {
		log.Fatal(err)
	}

	keyboardImage, _ = ebiten.NewImageFromImage(img, ebiten.FilterDefault)
}

func update(screen *ebiten.Image) error {
	pressed := []ebiten.Key{}
	for k := ebiten.Key(0); k <= ebiten.KeyMax; k++ {
		if ebiten.IsKeyPressed(k) {
			pressed = append(pressed, k)
		}
	}

	if ebiten.IsRunningSlowly() {
		return nil
	}

	const (
		offsetX = 24
		offsetY = 40
	)

	// Draw the base (grayed) keyboard image.
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(offsetX, offsetY)
	op.ColorM.Scale(0.5, 0.5, 0.5, 1)
	screen.DrawImage(keyboardImage, op)

	// Draw the highlighted keys.
	op = &ebiten.DrawImageOptions{}
	for _, p := range pressed {
		op.GeoM.Reset()
		r, ok := keyboard.KeyRect(p)
		if !ok {
			continue
		}
		op.GeoM.Translate(float64(r.Min.X), float64(r.Min.Y))
		op.GeoM.Translate(offsetX, offsetY)
		op.SourceRect = &r
		screen.DrawImage(keyboardImage, op)
	}

	keyStrs := []string{}
	for _, p := range pressed {
		keyStrs = append(keyStrs, p.String())
	}
	ebitenutil.DebugPrint(screen, strings.Join(keyStrs, ", "))

	return nil
}

func main() {
	if err := ebiten.Run(update, screenWidth, screenHeight, 2, "Keyboard (Ebiten Demo)"); err != nil {
		log.Fatal(err)
	}
}
