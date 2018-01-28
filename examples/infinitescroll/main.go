// Copyright 2016 The Ebiten Authors
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
	bgImage *ebiten.Image
)

func init() {
	var err error
	bgImage, _, err = ebitenutil.NewImageFromFile(ebitenutil.JoinStringsIntoFilePath("_resources", "images", "tile.png"), ebiten.FilterNearest)
	if err != nil {
		log.Fatal(err)
	}
}

var (
	theViewport = &viewport{}
)

type viewport struct {
	x16 int
	y16 int
}

func (p *viewport) Move() {
	w, h := bgImage.Size()
	maxX16 := w * 16
	maxY16 := h * 16

	p.x16 += w / 32
	p.y16 += h / 32
	p.x16 %= maxX16
	p.y16 %= maxY16
}

func (p *viewport) Position() (int, int) {
	return p.x16, p.y16
}

func update(screen *ebiten.Image) error {
	theViewport.Move()

	if ebiten.IsRunningSlowly() {
		return nil
	}

	x16, y16 := theViewport.Position()
	offsetX, offsetY := float64(-x16)/16, float64(-y16)/16

	// Draw bgImage on the screen repeatedly.
	const repeat = 3
	w, h := bgImage.Size()
	for j := 0; j < repeat; j++ {
		for i := 0; i < repeat; i++ {
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64(w*i), float64(h*j))
			op.GeoM.Translate(offsetX, offsetY)
			screen.DrawImage(bgImage, op)
		}
	}

	ebitenutil.DebugPrint(screen, fmt.Sprintf("FPS: %0.2f", ebiten.CurrentFPS()))
	return nil
}

func main() {
	if err := ebiten.Run(update, screenWidth, screenHeight, 2, "Infinite Scroll (Ebiten Demo)"); err != nil {
		log.Fatal(err)
	}
}
