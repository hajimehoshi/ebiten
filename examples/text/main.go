// Copyright 2020 The Ebiten Authors
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
	"image/color"
	"log"
	"math"
	"math/rand"
	"time"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/examples/resources/fonts"
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	screenWidth  = 640
	screenHeight = 480
)

const sampleText = `  The quick brown fox jumps
over the lazy dog.`

var (
	mplusNormalFont font.Face
	mplusBigFont    font.Face
)

func init() {
	tt, err := opentype.Parse(fonts.MPlus1pRegular_ttf)
	if err != nil {
		log.Fatal(err)
	}

	const dpi = 72
	mplusNormalFont, err = opentype.NewFace(tt, &opentype.FaceOptions{
		Size:    24,
		DPI:     dpi,
		Hinting: font.HintingVertical,
	})
	if err != nil {
		log.Fatal(err)
	}
	mplusBigFont, err = opentype.NewFace(tt, &opentype.FaceOptions{
		Size:    32,
		DPI:     dpi,
		Hinting: font.HintingVertical,
	})
	if err != nil {
		log.Fatal(err)
	}
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

type Game struct {
	counter        int
	kanjiText      []rune
	kanjiTextColor color.RGBA
	glyphs         []text.Glyph
}

func (g *Game) Update() error {
	// Initialize the glyphs for special (colorful) rendering.
	if len(g.glyphs) == 0 {
		g.glyphs = text.AppendGlyphs(g.glyphs, mplusNormalFont, sampleText)
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	gray := color.RGBA{0x80, 0x80, 0x80, 0xff}

	{
		const x, y = 20, 40
		b := text.BoundString(mplusNormalFont, sampleText)
		vector.DrawFilledRect(screen, float32(b.Min.X+x), float32(b.Min.Y+y), float32(b.Dx()), float32(b.Dy()), gray, false)
		text.Draw(screen, sampleText, mplusNormalFont, x, y, color.White)
	}
	{
		const x, y = 20, 140
		b := text.BoundString(mplusBigFont, sampleText)
		vector.DrawFilledRect(screen, float32(b.Min.X+x), float32(b.Min.Y+y), float32(b.Dx()), float32(b.Dy()), gray, false)
		text.Draw(screen, sampleText, mplusBigFont, x, y, color.White)
	}
	{
		const x, y = 20, 240
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Rotate(math.Pi / 4)
		op.GeoM.Translate(x, y)
		op.Filter = ebiten.FilterLinear
		text.DrawWithOptions(screen, sampleText, mplusNormalFont, op)
	}
	{
		const x, y = 160, 240
		const lineHeight = 80
		b := text.BoundString(text.FaceWithLineHeight(mplusBigFont, lineHeight), sampleText)
		vector.DrawFilledRect(screen, float32(b.Min.X+x), float32(b.Min.Y+y), float32(b.Dx()), float32(b.Dy()), gray, false)
		text.Draw(screen, sampleText, text.FaceWithLineHeight(mplusBigFont, lineHeight), x, y, color.White)
	}
	{
		const x, y = 240, 400
		op := &ebiten.DrawImageOptions{}
		// g.glyphs is initialized by text.AppendGlyphs.
		// You can customize how to render each glyph.
		// In this example, multiple colors are used to render glyphs.
		for i, gl := range g.glyphs {
			op.GeoM.Reset()
			op.GeoM.Translate(x, y)
			op.GeoM.Translate(gl.X, gl.Y)
			op.ColorScale.Reset()
			r := float32(1)
			if i%3 == 0 {
				r = 0.5
			}
			g := float32(1)
			if i%3 == 1 {
				g = 0.5
			}
			b := float32(1)
			if i%3 == 2 {
				b = 0.5
			}
			op.ColorScale.Scale(r, g, b, 1)
			screen.DrawImage(gl.Image, op)
		}
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Text (Ebitengine Demo)")
	if err := ebiten.RunGame(&Game{}); err != nil {
		log.Fatal(err)
	}
}
