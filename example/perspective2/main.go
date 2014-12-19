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
	_ "image/jpeg"
	"log"
)

const (
	screenWidth  = 320
	screenHeight = 240
)

type Game struct {
	gophersTexture   *ebiten.Texture
	textRenderTarget *ebiten.RenderTarget
}

func (g *Game) Update(gr ebiten.GraphicsContext) error {
	gr.PushRenderTarget(g.textRenderTarget)
	gr.Fill(0x80, 0x40, 0x40)
	ebitenutil.DebugPrint(gr, "Hello, World!")
	gr.PopRenderTarget()

	gr.Fill(0x40, 0x40, 0x80)
	geo := ebiten.GeometryMatrixI()
	clr := ebiten.ColorMatrixI()
	ebiten.DrawWholeTexture(gr, g.textRenderTarget.Texture(), geo, clr)
	return nil
}

func main() {
	g := new(Game)
	var err error
	g.gophersTexture, _, err = ebitenutil.NewTextureFromFile("images/gophers.jpg", ebiten.FilterNearest)
	if err != nil {
		log.Fatal(err)
	}
	g.textRenderTarget, err = ebiten.NewRenderTarget(screenWidth, screenHeight, ebiten.FilterNearest)
	if err != nil {
		log.Fatal(err)
	}
	if err := ebiten.Run(g.Update, screenWidth, screenHeight, 2, "Perspective (Ebiten Demo)"); err != nil {
		log.Fatal(err)
	}
}
