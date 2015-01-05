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

package main

import (
	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"
	"image/color"
	_ "image/png"
	"log"
)

const (
	screenWidth  = 320
	screenHeight = 240
)

var (
	count           int
	tmpRenderTarget *ebiten.Image
	ebitenImage     *ebiten.Image
	saved           bool
)

func update(screen *ebiten.Image) error {
	count++
	count %= 600
	diff := float64(count) * 0.2
	switch {
	case 480 < count:
		diff = 0
	case 240 < count:
		diff = float64(480-count) * 0.2
	}

	if err := tmpRenderTarget.Clear(); err != nil {
		return err
	}
	for i := 0; i < 10; i++ {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(15+float64(i)*diff, 20)
		op.ColorM.Scale(1.0, 1.0, 1.0, 0.5)
		if err := tmpRenderTarget.DrawImage(ebitenImage, op); err != nil {
			return err
		}
	}

	screen.Fill(color.NRGBA{0x00, 0x00, 0x80, 0xff})
	for i := 0; i < 10; i++ {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(0, float64(i)*diff)
		if err := screen.DrawImage(tmpRenderTarget, op); err != nil {
			return err
		}
	}

	return nil
}

func main() {
	var err error
	ebitenImage, _, err = ebitenutil.NewImageFromFile("images/ebiten.png", ebiten.FilterNearest)
	if err != nil {
		log.Fatal(err)
	}
	tmpRenderTarget, err = ebiten.NewImage(screenWidth, screenHeight, ebiten.FilterNearest)
	if err != nil {
		log.Fatal(err)
	}
	//update := update
	//f, _ := os.Create("out.gif")
	//update = ebitenutil.RecordScreenAsGIF(update, f, 100)
	if err := ebiten.Run(update, screenWidth, screenHeight, 2, "Alpha Blending (Ebiten Demo)"); err != nil {
		log.Fatal(err)
	}
}
