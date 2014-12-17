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
	gophersTexture ebiten.TextureID
}

func (g *Game) Update(gr ebiten.GraphicsContext) error {
	ebiten.DrawWhole(gr.Texture(g.gophersTexture), 500, 414, ebiten.GeometryMatrixI(), ebiten.ColorMatrixI())
	return nil
}

func main() {
	g := new(Game)
	id, err := ebitenutil.NewTextureIDFromFile("images/gophers.jpg", ebiten.FilterLinear)
	if err != nil {
		log.Fatal(err)
	}
	g.gophersTexture = id
	if err := ebiten.Run(g.Update, screenWidth, screenHeight, 2, "Image (Ebiten Demo)"); err != nil {
		log.Fatal(err)
	}
}
