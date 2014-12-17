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
	"image"
	_ "image/jpeg"
	"log"
	"os"
)

const (
	screenWidth  = 320
	screenHeight = 240
)

type Game struct {
	gophersTexture ebiten.TextureID
}

func (g *Game) Update(gr ebiten.GraphicsContext) error {
	if g.gophersTexture.IsNil() {
		file, err := os.Open("images/gophers.jpg")
		if err != nil {
			return err
		}
		defer file.Close()
		img, _, err := image.Decode(file)
		if err != nil {
			return err
		}
		id, err := ebiten.NewTextureID(img, ebiten.FilterLinear)
		if err != nil {
			return err
		}
		g.gophersTexture = id
	}
	if g.gophersTexture.IsNil() {
		return nil
	}
	ebiten.DrawWhole(gr.Texture(g.gophersTexture), 500, 414, ebiten.GeometryMatrixI(), ebiten.ColorMatrixI())
	return nil
}

func main() {
	g := new(Game)
	if err := ebiten.Run(g.Update, screenWidth, screenHeight, 2, "Image (Ebiten Demo)"); err != nil {
		log.Fatal(err)
	}
}
