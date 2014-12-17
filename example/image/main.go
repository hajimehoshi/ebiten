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
	"image"
	_ "image/jpeg"
	"log"
	"math"
)

const (
	screenWidth  = 320
	screenHeight = 240
)

type Game struct {
	count                int
	gophersTexture       *ebiten.Texture
	gophersTextureWidth  int
	gophersTextureHeight int
}

func (g *Game) Update(gr ebiten.GraphicsContext) error {
	g.count++

	geo := ebiten.TranslateGeometry(-float64(g.gophersTextureWidth)/2, -float64(g.gophersTextureHeight)/2)
	geo.Concat(ebiten.ScaleGeometry(0.5, 0.5))
	geo.Concat(ebiten.TranslateGeometry(screenWidth/2, screenHeight/2))
	clr := ebiten.RotateHue(float64(g.count%180) * 2 * math.Pi / 180)
	ebiten.DrawWhole(gr.Texture(g.gophersTexture), g.gophersTextureWidth, g.gophersTextureHeight, geo, clr)
	return nil
}

func main() {
	g := new(Game)
	var img image.Image
	var err error
	g.gophersTexture, img, err = ebitenutil.NewTextureFromFile("images/gophers.jpg", ebiten.FilterLinear)
	if err != nil {
		log.Fatal(err)
	}
	g.gophersTextureWidth = img.Bounds().Size().X
	g.gophersTextureHeight = img.Bounds().Size().Y
	if err := ebiten.Run(g.Update, screenWidth, screenHeight, 2, "Image (Ebiten Demo)"); err != nil {
		log.Fatal(err)
	}
}
