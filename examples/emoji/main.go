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
	"io"
	"log"
	"net/http"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

const (
	screenWidth  = 640
	screenHeight = 480
)

// fontURL is a URL of Noto Color Emoji, a color emoji font using COLRv1 and SVG glyph data.
// The license is the SIL Open Font License, Version 1.1:
//
// https://github.com/googlefonts/noto-emoji
// Copyright 2022 Google Inc.
const fontURL = "https://res.ebitengine.org/examples/NotoColorEmoji-Regular.ttf"

const sampleText = "😀🍤🍣🍵😅\n🖐🏻🖐🏼🖐🏽🖐🏾🖐🏿"

type Game struct {
	faces []*text.GoTextFace
}

func (g *Game) Update() error {
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	y := 8.0
	for _, face := range g.faces {
		op := &text.DrawOptions{}
		op.GeoM.Translate(8, y)
		op.LineSpacing = face.Size * 1.25
		// Do not set a color scale: a color scale would also scale the colors of emojis.
		text.Draw(screen, sampleText, face, op)
		y += op.LineSpacing * 2
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	// You can specify your own emoji font file as an argument.
	// Color emoji fonts in the SVG, COLRv0, sbix, and CBDT formats are supported.
	// A COLRv1 font works when the font also has an SVG table, like Noto Color Emoji.
	var in io.ReadCloser
	if len(os.Args) > 1 {
		f, err := os.Open(os.Args[1])
		if err != nil {
			log.Fatal(err)
		}
		in = f
	} else {
		fmt.Println("Downloading the font. You can specify a font file as an argument instead.")
		res, err := http.Get(fontURL)
		if err != nil {
			log.Fatal(err)
		}
		in = res.Body
	}
	fontData, err := io.ReadAll(in)
	_ = in.Close()
	if err != nil {
		log.Fatal(err)
	}

	src, err := text.NewGoTextFaceSource(bytes.NewReader(fontData))
	if err != nil {
		log.Fatal(err)
	}

	g := &Game{}
	for _, size := range []float64{24, 48, 96} {
		g.faces = append(g.faces, &text.GoTextFace{
			Source: src,
			Size:   size,
		})
	}

	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Emoji (Ebitengine Demo)")
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
