// Copyright 2018 The Ebiten Authors
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

// This example is an experiment to minify images with various filters.
// When linear filter is used, mipmap images should be used for high-quality rendering (#578).

package main

import (
	"bytes"
	"image"
	_ "image/jpeg"
	"log"
	"math"

	"github.com/ebitengine/debugui"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/examples/resources/images"
)

const (
	screenWidth  = 1000
	screenHeight = 640
)

var (
	gophersImage *ebiten.Image
)

type Game struct {
	debugui debugui.DebugUI

	rotate    bool
	clip      bool
	autoScale bool
	counter   int
	scale     float64
}

func (g *Game) Update() error {
	if g.scale == 0 {
		g.scale = 1
	}
	g.counter++
	if g.counter == 480 {
		g.counter = 0
	}
	if g.autoScale {
		g.scale = 1.5 / math.Pow(1.01, float64(g.counter))
	}

	if _, err := g.debugui.Update(func(ctx *debugui.Context) error {
		ctx.Window("Control", image.Rect(10, 10, 260, 160), func(layout debugui.ContainerLayout) {
			ctx.SetGridLayout([]int{-1, -2}, nil)
			ctx.Text("Rotate")
			ctx.Checkbox(&g.rotate, "")
			ctx.Text("Clip")
			ctx.Checkbox(&g.clip, "")
			ctx.Text("Scale")
			ctx.SliderF(&g.scale, 0.01, 2, 0.01, 2)
			ctx.Text("Auto Scale")
			ctx.Checkbox(&g.autoScale, "")
		})
		ctx.Window("Info", image.Rect(270, 10, 520, 160), func(layout debugui.ContainerLayout) {
			ctx.Text("Minifying Images\nLeft:   Nearest filter\nCenter: Linear filter (w/ mipmaps)\nRight:  Linear Filter (w/o mipmaps)")
		})
		return nil
	}); err != nil {
		return err
	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	s := g.scale

	clippedGophersImage := gophersImage.SubImage(image.Rect(100, 100, 200, 200)).(*ebiten.Image)
	for i := range 3 {
		w, h := gophersImage.Bounds().Dx(), gophersImage.Bounds().Dy()

		op := &ebiten.DrawImageOptions{}
		if g.rotate {
			op.GeoM.Translate(-float64(w)/2, -float64(h)/2)
			op.GeoM.Rotate(float64(g.counter) / 300 * 2 * math.Pi)
			op.GeoM.Translate(float64(w)/2, float64(h)/2)
		}
		op.GeoM.Scale(s, s)
		op.GeoM.Translate(32+float64(i*w)*s+float64(i*4), 200)
		if i == 0 {
			op.Filter = ebiten.FilterNearest
		} else {
			op.Filter = ebiten.FilterLinear
		}
		if i == 2 {
			op.DisableMipmaps = true
		}
		if g.clip {
			screen.DrawImage(clippedGophersImage, op)
		} else {
			screen.DrawImage(gophersImage, op)
		}
	}

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
	ebiten.SetWindowTitle("Minify (Ebitengine Demo)")
	if err := ebiten.RunGame(&Game{}); err != nil {
		log.Fatal(err)
	}
}
