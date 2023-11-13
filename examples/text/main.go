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
	"strings"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/examples/resources/fonts"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	screenWidth  = 640
	screenHeight = 480
)

const sampleText = `  The quick brown fox jumps
over the lazy dog.`

var (
	mplusNormalFace *text.StdFace
	mplusBigFace    *text.StdFace
)

func init() {
	tt, err := opentype.Parse(fonts.MPlus1pRegular_ttf)
	if err != nil {
		log.Fatal(err)
	}

	const dpi = 72
	mplusNormalFont, err := opentype.NewFace(tt, &opentype.FaceOptions{
		Size:    24,
		DPI:     dpi,
		Hinting: font.HintingVertical,
	})
	if err != nil {
		log.Fatal(err)
	}
	mplusNormalFace = text.NewStdFace(mplusNormalFont)

	mplusBigFont, err := opentype.NewFace(tt, &opentype.FaceOptions{
		Size:    32,
		DPI:     dpi,
		Hinting: font.HintingVertical,
	})
	if err != nil {
		log.Fatal(err)
	}
	mplusBigFace = text.NewStdFace(mplusBigFont)
}

type Game struct {
	counter        int
	kanjiText      []rune
	kanjiTextColor color.RGBA
	glyphs         [][]text.Glyph
}

func (g *Game) Update() error {
	// Initialize the glyphs for special (colorful) rendering.
	if len(g.glyphs) == 0 {
		for _, line := range strings.Split(sampleText, "\n") {
			g.glyphs = append(g.glyphs, text.AppendGlyphs(nil, line, mplusNormalFace, 0, 0))
		}
	}
	return nil
}

func fixed26_6ToFloat32(x fixed.Int26_6) float32 {
	return float32(x>>6) + float32(x&((1<<6)-1))/(1<<6)
}

func (g *Game) Draw(screen *ebiten.Image) {
	gray := color.RGBA{0x80, 0x80, 0x80, 0xff}

	{
		const x, y = 20, 20
		w, h := text.Measure(sampleText, mplusNormalFace, mplusNormalFace.Metrics().Height)
		vector.DrawFilledRect(screen, x, y, float32(w), float32(h), gray, false)
		op := &text.DrawOptions{}
		op.GeoM.Translate(x, y)
		op.LineHeightInPixels = mplusNormalFace.Metrics().Height
		text.Draw(screen, sampleText, mplusNormalFace, op)
	}
	{
		const x, y = 20, 120
		w, h := text.Measure(sampleText, mplusBigFace, mplusBigFace.Metrics().Height)
		vector.DrawFilledRect(screen, x, y, float32(w), float32(h), gray, false)
		op := &text.DrawOptions{}
		op.GeoM.Translate(x, y)
		op.LineHeightInPixels = mplusBigFace.Metrics().Height
		text.Draw(screen, sampleText, mplusBigFace, op)
	}
	{
		const x, y = 20, 220
		op := &text.DrawOptions{}
		op.GeoM.Rotate(math.Pi / 4)
		op.GeoM.Translate(x, y)
		op.Filter = ebiten.FilterLinear
		op.LineHeightInPixels = mplusNormalFace.Metrics().Height
		text.Draw(screen, sampleText, mplusNormalFace, op)
	}
	{
		const x, y = 160, 220
		const lineHeight = 80
		w, h := text.Measure(sampleText, mplusBigFace, lineHeight)
		vector.DrawFilledRect(screen, x, y, float32(w), float32(h), gray, false)
		op := &text.DrawOptions{}
		// Add the width as the text rendering region's upper-right position comes to (0, 0)
		// when the horizontal alignment is right. The alignment is specified later (PrimaryAlign).
		op.GeoM.Translate(x+w, y)
		op.LineHeightInPixels = lineHeight
		// The primary alignment for the left-to-right direction is a horizontal alignment, and the end means the right.
		op.PrimaryAlign = text.AlignEnd
		text.Draw(screen, sampleText, mplusBigFace, op)
	}
	{
		const x, y = 240, 380
		op := &ebiten.DrawImageOptions{}
		// g.glyphs is initialized by text.AppendGlyphs.
		// You can customize how to render each glyph.
		// In this example, multiple colors are used to render glyphs.
		for j, line := range g.glyphs {
			for i, gl := range line {
				op.GeoM.Reset()
				op.GeoM.Translate(x, y)
				op.GeoM.Translate(0, float64(j)*mplusNormalFace.Metrics().Height)
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
