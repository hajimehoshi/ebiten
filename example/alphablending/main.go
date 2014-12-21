/*
Copyright 2014 Hajime Hoshi

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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

type Game struct {
	count           int
	tmpRenderTarget *ebiten.RenderTarget
	ebitenImage     *ebiten.Image
}

func (g *Game) Update(r *ebiten.RenderTarget) error {
	g.count++
	g.count %= 600
	diff := float64(g.count) * 0.2
	switch {
	case 480 < g.count:
		diff = 0
	case 240 < g.count:
		diff = float64(480-g.count) * 0.2
	}
	_ = diff

	if err := g.tmpRenderTarget.Clear(); err != nil {
		return err
	}
	for i := 0; i < 10; i++ {
		geo := ebiten.TranslateGeometry(15+float64(i)*(diff), 20)
		clr := ebiten.ScaleColor(1.0, 1.0, 1.0, 0.5)
		if err := ebiten.DrawWholeImage(g.tmpRenderTarget, g.ebitenImage, geo, clr); err != nil {
			return err
		}
	}

	r.Fill(color.NRGBA{0x00, 0x00, 0x80, 0xff})
	for i := 0; i < 10; i++ {
		geo := ebiten.TranslateGeometry(0, float64(i)*(diff))
		clr := ebiten.ColorMatrixI()
		if err := ebiten.DrawWholeImage(r, g.tmpRenderTarget.Image(), geo, clr); err != nil {
			return err
		}
	}
	return nil
}

func main() {
	g := new(Game)
	var err error
	g.ebitenImage, _, err = ebitenutil.NewImageFromFile("images/ebiten.png", ebiten.FilterNearest)
	if err != nil {
		log.Fatal(err)
	}
	g.tmpRenderTarget, err = ebiten.NewRenderTarget(screenWidth, screenHeight, ebiten.FilterNearest)
	if err != nil {
		log.Fatal(err)
	}
	if err := ebiten.Run(g.Update, screenWidth, screenHeight, 2, "Alpha Blending (Ebiten Demo)"); err != nil {
		log.Fatal(err)
	}
}
