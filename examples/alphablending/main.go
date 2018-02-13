// Copyright 2014 Hajime Hoshi
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
	"image/color"
	_ "image/png"
	"log"

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"
)

const (
	screenWidth  = 320
	screenHeight = 240
)

var (
	count       int
	ebitenImage *ebiten.Image
)

func update(screen *ebiten.Image) error {
	count++
	count %= ebiten.FPS * 10
	diff := float64(count) * 0.2
	switch {
	case 480 < count:
		diff = 0
	case 240 < count:
		diff = float64(480-count) * 0.2
	}

	if ebiten.IsRunningSlowly() {
		return nil
	}

	screen.Fill(color.NRGBA{0x00, 0x00, 0x80, 0xff})

	// Draw 100 Ebitens
	op := &ebiten.DrawImageOptions{}
	op.ColorM.Scale(1.0, 1.0, 1.0, 0.5)
	for i := 0; i < 10*10; i++ {
		op.GeoM.Reset()
		x := float64(i%10)*diff + 15
		y := float64(i/10)*diff + 20
		op.GeoM.Translate(x, y)
		screen.DrawImage(ebitenImage, op)
	}
	return nil
}

func main() {
	var err error
	ebitenImage, _, err = ebitenutil.NewImageFromFile("_resources/images/ebiten.png", ebiten.FilterDefault)
	if err != nil {
		log.Fatal(err)
	}
	if err := ebiten.Run(update, screenWidth, screenHeight, 2, "Alpha Blending (Ebiten Demo)"); err != nil {
		log.Fatal(err)
	}
}
