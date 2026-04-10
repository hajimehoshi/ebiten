// Copyright 2026 The Ebitengine Authors
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
	"fmt"
	"image/color"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/examples/resources/fonts"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

const (
	screenWidth  = 900
	screenHeight = 450
)

const sampleText = "The quick brown fox jumps over the lazy dog."

var terminusFaceSource *text.GoTextFaceSource

func init() {
	s, err := text.NewGoTextFaceSource(bytes.NewReader(fonts.TerminusTTF_ttf))
	if err != nil {
		log.Fatal(err)
	}
	terminusFaceSource = s
}

var fontSizes = []float64{12, 14, 16, 20, 24, 32, 48}

type Game struct{}

func (g *Game) Update() error {
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	y := 16.0
	for _, size := range fontSizes {
		op := &text.DrawOptions{}
		op.GeoM.Translate(20, y)
		op.ColorScale.ScaleWithColor(color.White)
		text.Draw(screen, fmt.Sprintf("%.0fpx: %s", size, sampleText), &text.GoTextFace{
			Source: terminusFaceSource,
			Size:   size,
		}, op)
		y += size * 1.5
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Bitmap Font (Ebitengine Demo)")
	if err := ebiten.RunGame(&Game{}); err != nil {
		log.Fatal(err)
	}
}
