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
	"bytes"
	"image/color"
	"log"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/examples/resources/fonts"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
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
	mplusFaceSource *text.GoTextFaceSource
	mplusNormalFace *text.GoTextFace
	mplusBigFace    *text.GoTextFace
)

func init() {
	s, err := text.NewGoTextFaceSource(bytes.NewReader(fonts.MPlus1pRegular_ttf))
	if err != nil {
		log.Fatal(err)
	}
	mplusFaceSource = s

	mplusNormalFace = &text.GoTextFace{
		Source: mplusFaceSource,
		Size:   24,
	}
	mplusBigFace = &text.GoTextFace{
		Source: mplusFaceSource,
		Size:   32,
	}
}

type Game struct {
	glyphs      []text.Glyph
	showOrigins bool
}

func (g *Game) Update() error {
	// Initialize the glyphs for special (colorful) rendering.
	if len(g.glyphs) == 0 {
		op := &text.LayoutOptions{}
		op.LineSpacing = mplusNormalFace.Size * 1.5
		g.glyphs = text.AppendGlyphs(g.glyphs, sampleText, mplusNormalFace, op)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyO) {
		g.showOrigins = !g.showOrigins
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	ebitenutil.DebugPrint(screen, "Press O to show/hide origins")

	gray := color.RGBA{0x80, 0x80, 0x80, 0xff}

	{
		const x, y = 20, 20
		w, h := text.Measure(sampleText, mplusNormalFace, mplusNormalFace.Size*1.5)
		vector.FillRect(screen, x, y, float32(w), float32(h), gray, false)
		op := &text.DrawOptions{}
		op.GeoM.Translate(x, y)
		op.LineSpacing = mplusNormalFace.Size * 1.5
		text.Draw(screen, sampleText, mplusNormalFace, op)
	}
	{
		const x, y = 20, 120
		w, h := text.Measure(sampleText, mplusBigFace, mplusBigFace.Size*1.5)
		vector.FillRect(screen, x, y, float32(w), float32(h), gray, false)
		op := &text.DrawOptions{}
		op.GeoM.Translate(x, y)
		op.LineSpacing = mplusBigFace.Size * 1.5
		text.Draw(screen, sampleText, mplusBigFace, op)
	}
	{
		const x, y = 20, 220
		op := &text.DrawOptions{}
		op.GeoM.Rotate(math.Pi / 4)
		op.GeoM.Translate(x, y)
		op.Filter = ebiten.FilterLinear
		op.LineSpacing = mplusNormalFace.Size * 1.5
		text.Draw(screen, sampleText, mplusNormalFace, op)
	}
	{
		const x, y = 160, 220
		const lineSpacingInPixels = 80
		w, h := text.Measure(sampleText, mplusBigFace, lineSpacingInPixels)
		vector.FillRect(screen, x, y, float32(w), float32(h), gray, false)
		op := &text.DrawOptions{}
		// Add the width as the text rendering region's upper-right position comes to (0, 0)
		// when the horizontal alignment is right. The alignment is specified later (PrimaryAlign).
		op.GeoM.Translate(x+w, y)
		op.LineSpacing = lineSpacingInPixels
		// The primary alignment for the left-to-right direction is a horizontal alignment, and the end means the right.
		op.PrimaryAlign = text.AlignEnd
		text.Draw(screen, sampleText, mplusBigFace, op)
	}
	{
		const x, y = 240, 360
		op := &ebiten.DrawImageOptions{}
		// g.glyphs is initialized by text.AppendGlyphs.
		// You can customize how to render each glyph.
		// In this example, multiple colors are used to render glyphs.
		for i, gl := range g.glyphs {
			if gl.Image == nil {
				continue
			}
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

		if g.showOrigins {
			for _, gl := range g.glyphs {
				vector.FillCircle(screen, x+float32(gl.OriginX), y+float32(gl.OriginY), 2, color.RGBA{0xff, 0, 0, 0xff}, true)
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
