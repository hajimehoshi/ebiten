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
	"io"
	"log"
	"math"
	"net/http"

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

const emojiSampleText = "Sushi🍣"

// emojiFontURL is a URL of Noto Color Emoji, a color emoji font using COLRv1 and SVG glyph data.
// The license is the SIL Open Font License, Version 1.1:
//
// https://fonts.google.com/noto/specimen/Noto+Color+Emoji
// Copyright 2022 Google Inc.
const emojiFontURL = "https://res.ebitengine.org/examples/NotoColorEmoji-Regular.ttf"

var (
	mplusFaceSource *text.GoTextFaceSource
	mplusNormalFace *text.GoTextFace
	mplusBigFace    *text.GoTextFace

	// textWithEmojiFaceCh receives a face combining M+ and Noto Color Emoji.
	textWithEmojiFaceCh = make(chan text.Face, 1)
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
	glyphs            []text.LazyGlyph
	showOrigins       bool
	textWithEmojiFace text.Face
}

func (g *Game) Update() error {
	// Initialize the glyphs for special (colorful) rendering.
	if len(g.glyphs) == 0 {
		op := &text.LayoutOptions{}
		op.LineSpacing = mplusNormalFace.Size * 1.5
		g.glyphs = text.AppendLazyGlyphs(g.glyphs, sampleText, mplusNormalFace, op)
	}
	if g.textWithEmojiFace == nil {
		select {
		case g.textWithEmojiFace = <-textWithEmojiFaceCh:
		default:
		}
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
		const x, y = screenWidth - 20, 20
		str := emojiSampleText
		face := g.textWithEmojiFace
		if face == nil {
			str = "Loading an Emoji font..."
			face = mplusNormalFace
		}
		w, h := text.Measure(str, face, 0)
		vector.FillRect(screen, float32(x)-float32(w), y, float32(w), float32(h), gray, false)
		op := &text.DrawOptions{}
		op.GeoM.Translate(x, y)
		op.PrimaryAlign = text.AlignEnd
		// The color scale scales the text color, except for color glyphs like emojis,
		// to which only the alpha is applied. The emoji keeps its own colors.
		op.ColorScale.ScaleWithColor(color.Black)
		text.Draw(screen, str, face, op)
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
		// g.glyphs is initialized by text.AppendLazyGlyphs.
		// You can customize how to render each glyph.
		// In this example, multiple colors are used to render glyphs.
		//
		// Realize every glyph image before issuing any DrawImage so the
		// atlas write-pixels batch is not flushed on each glyph.
		for _, gl := range g.glyphs {
			gl.Image()
		}
		for i, gl := range g.glyphs {
			img := gl.Image()
			if img == nil {
				continue
			}
			op.GeoM.Reset()
			op.GeoM.Translate(x, y)
			op.GeoM.Translate(float64(gl.ImageBounds.Min.X), float64(gl.ImageBounds.Min.Y))
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
			screen.DrawImage(img, op)
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
	// Downloading the emoji font might take time, so download it in parallel.
	// The text with the emoji is rendered after the download is done.
	go func() {
		res, err := http.Get(emojiFontURL)
		if err != nil {
			log.Fatal(err)
		}
		fontData, err := io.ReadAll(res.Body)
		_ = res.Body.Close()
		if err != nil {
			log.Fatal(err)
		}
		emojiFaceSource, err := text.NewGoTextFaceSource(bytes.NewReader(fontData))
		if err != nil {
			log.Fatal(err)
		}
		f, err := text.NewMultiFace(mplusNormalFace, &text.GoTextFace{
			Source: emojiFaceSource,
			Size:   mplusNormalFace.Size,
		})
		if err != nil {
			log.Fatal(err)
		}
		textWithEmojiFaceCh <- f
	}()

	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Text (Ebitengine Demo)")
	if err := ebiten.RunGame(&Game{}); err != nil {
		log.Fatal(err)
	}
}
