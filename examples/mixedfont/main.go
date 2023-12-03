// Copyright 2023 The Ebitengine Authors
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
	"log"

	"golang.org/x/image/font/gofont/goregular"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/examples/resources/fonts"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

const (
	screenWidth  = 640
	screenHeight = 480
)

var (
	goRegularFaceSource *text.GoTextFaceSource
	mplusFaceSource     *text.GoTextFaceSource
)

func init() {
	s, err := text.NewGoTextFaceSource(bytes.NewReader(fonts.MPlus1pRegular_ttf))
	if err != nil {
		log.Fatal(err)
	}
	mplusFaceSource = s
}

func init() {
	s, err := text.NewGoTextFaceSource(bytes.NewReader(goregular.TTF))
	if err != nil {
		log.Fatal(err)
	}
	goRegularFaceSource = s
}

type Game struct {
	face text.Face
}

func (g *Game) Update() error {
	if g.face == nil {
		// goregular.TTF is used primarily. If a glyph is not found in this font, the second font is used.
		// Use text.LimitedFace to limit the glyphs.
		en := text.NewLimitedFace(&text.GoTextFace{
			Source: goRegularFaceSource,
			Size:   24,
		})
		// Limit the glyphs for ASCII and Latin-1 characters for this face.
		// This means that, for example, '…' (U+2026) is not rendered by this face.
		en.AddUnicodeRange('\u0020', '\u00ff')

		// M+ Font is the second font.
		// Use a relatively big size to see different-sized faces are well mixed.
		ja := &text.GoTextFace{
			Source: mplusFaceSource,
			Size:   32,
		}

		f, err := text.NewMultiFace(en, ja)
		if err != nil {
			return err
		}
		g.face = f
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	op := &text.DrawOptions{}
	op.GeoM.Translate(20, 20)
	op.LineSpacingInPixels = 48
	text.Draw(screen, "HelloこんにちはWorld世界\n日本語とEnglish…", g.face, op)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Mixed Font Faces (Ebitengine Demo)")
	if err := ebiten.RunGame(&Game{}); err != nil {
		log.Fatal(err)
	}
}
