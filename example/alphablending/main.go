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
	ebitenTexture   *ebiten.Texture
}

func (g *Game) Update(gr ebiten.GraphicsContext) error {
	g.count++
	g.count %= 120
	diff := float64(g.count) * 0.5
	if 60 < g.count {
		diff = float64(120-g.count) * 0.5
	}

	gr.PushRenderTarget(g.tmpRenderTarget)
	gr.Clear()
	for i := 0; i < 10; i++ {
		geo := ebiten.TranslateGeometry(15+float64(i)*(20+diff), 20)
		clr := ebiten.ScaleColor(color.RGBA{0xff, 0xff, 0xff, 0x80})
		ebiten.DrawWholeTexture(gr, g.ebitenTexture, geo, clr)
	}
	gr.PopRenderTarget()

	gr.Fill(color.RGBA{0x00, 0x00, 0x80, 0xff})
	for i := 0; i < 10; i++ {
		geo := ebiten.TranslateGeometry(0, float64(i)*(10+diff))
		clr := ebiten.ColorMatrixI()
		ebiten.DrawWholeTexture(gr, g.tmpRenderTarget.Texture(), geo, clr)
	}
	return nil
}

func main() {
	g := new(Game)
	var err error
	g.ebitenTexture, _, err = ebitenutil.NewTextureFromFile("images/ebiten.png", ebiten.FilterNearest)
	if err != nil {
		log.Fatal(err)
	}
	g.tmpRenderTarget, err = ebiten.NewRenderTarget(screenWidth, screenHeight, ebiten.FilterNearest)
	if err != nil {
		log.Fatal(err)
	}
	if err := ebiten.Run(g.Update, screenWidth, screenHeight, 2, "Perspective (Ebiten Demo)"); err != nil {
		log.Fatal(err)
	}
}
