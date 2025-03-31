// Copyright 2016 Hajime Hoshi
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
	"bytes"
	"image"
	_ "image/jpeg"
	"log"

	"github.com/ebitengine/debugui"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/colorm"
	"github.com/hajimehoshi/ebiten/v2/examples/resources/images"
)

const (
	screenWidth  = 640
	screenHeight = 480
)

var (
	gophersImage *ebiten.Image
)

type Game struct {
	debugui debugui.DebugUI

	hue        float64
	saturation float64
	value      float64
	inverted   bool
}

func NewGame() *Game {
	return &Game{
		saturation: 1,
		value:      1,
	}
}

func (g *Game) Update() error {
	if _, err := g.debugui.Update(func(ctx *debugui.Context) error {
		ctx.Window("HSV", image.Rect(10, 10, 260, 160), func(layout debugui.ContainerLayout) {
			ctx.SetGridLayout([]int{-1, -2}, nil)
			ctx.Text("Hue")
			ctx.SliderF(&g.hue, -4, 4, 0.01, 2)
			ctx.Text("Saturation")
			ctx.SliderF(&g.saturation, 0, 2, 0.01, 2)
			ctx.Text("Value")
			ctx.SliderF(&g.value, 0, 2, 0.01, 2)
			ctx.Text("Inverted")
			ctx.Checkbox(&g.inverted, "")
			ctx.Text("Reset")
			ctx.Button("Reset").On(func() {
				g.hue = 0
				g.saturation = 1
				g.value = 1
				g.inverted = false
			})
		})
		return nil
	}); err != nil {
		return err
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	// Center the image on the screen.
	s := gophersImage.Bounds().Size()
	op := &colorm.DrawImageOptions{}
	op.GeoM.Translate(-float64(s.X)/2, -float64(s.Y)/2)
	op.GeoM.Scale(2, 2)
	op.GeoM.Translate(float64(screenWidth)/2, float64(screenHeight)/2)

	// Change HSV.
	var c colorm.ColorM
	c.ChangeHSV(g.hue, g.saturation, g.value)

	// Invert the color.
	if g.inverted {
		c.Scale(-1, -1, -1, 1)
		c.Translate(1, 1, 1, 0)
	}

	colorm.DrawImage(screen, gophersImage, c, op)

	g.debugui.Draw(screen)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	// Decode an image from the image file's byte slice.
	img, _, err := image.Decode(bytes.NewReader(images.Gophers_jpg))
	if err != nil {
		log.Fatal(err)
	}
	gophersImage = ebiten.NewImageFromImage(img)

	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("HSV (Ebitengine Demo)")
	if err := ebiten.RunGame(NewGame()); err != nil {
		log.Fatal(err)
	}
}
