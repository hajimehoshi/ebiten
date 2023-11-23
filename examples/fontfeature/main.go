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
	_ "embed"
	"fmt"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

const (
	screenWidth  = 640
	screenHeight = 480
)

//go:embed FiraSans-Regular.ttf
var firaSansRegular []byte

var firaSansFaceSource *text.GoTextFaceSource

func init() {
	s, err := text.NewGoTextFaceSource(bytes.NewReader(firaSansRegular))
	if err != nil {
		log.Fatal(err)
	}
	firaSansFaceSource = s
}

type Game struct {
	// liga represents 'Standard Ligatures'.
	// https://learn.microsoft.com/en-us/typography/opentype/spec/features_ko#tag-liga
	liga uint32

	// tnum represents 'Tabular Figures'.
	// https://learn.microsoft.com/en-us/typography/opentype/spec/features_pt#tag-tnum
	tnum uint32

	// smcp represents 'Small Capitals'.
	// https://learn.microsoft.com/en-us/typography/opentype/spec/features_pt#tag-smcp
	smcp uint32

	// zero represents 'Slashed Zero'.
	// https://learn.microsoft.com/en-us/typography/opentype/spec/features_uz#tag-zero
	zero uint32
}

func (g *Game) Update() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyL) {
		if g.liga == 0 {
			g.liga = 1
		} else {
			g.liga = 0
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyT) {
		if g.tnum == 0 {
			g.tnum = 1
		} else {
			g.tnum = 0
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyS) {
		if g.smcp == 0 {
			g.smcp = 1
		} else {
			g.smcp = 0
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyZ) {
		if g.zero == 0 {
			g.zero = 1
		} else {
			g.zero = 0
		}
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	// Draw the instruction.
	inst := fmt.Sprintf(`Press keys to toggle font features.
[L] 'liga' (Standard Ligatures) (%d)
[T] 'tnum' (Tabular Figures) (%d)
[S] 'smcp' (Small Capitals) (%d)
[Z] 'zero' (Slashed Zero) (%d)`, g.liga, g.tnum, g.smcp, g.zero)
	op := &text.DrawOptions{}
	op.GeoM.Translate(20, 20)
	op.LineSpacingInPixels = 30
	text.Draw(screen, inst, &text.GoTextFace{
		Source: firaSansFaceSource,
		Size:   20,
	}, op)

	// Draw the sample text.
	const sampleText = `0 (Number) / O (Alphabet)
ffi
3.14
2.71`
	op = &text.DrawOptions{}
	op.GeoM.Translate(20, screenHeight/2)
	op.LineSpacingInPixels = 50
	f := &text.GoTextFace{
		Source: firaSansFaceSource,
		Size:   40,
	}
	f.SetFeature(text.MustParseTag("liga"), g.liga)
	f.SetFeature(text.MustParseTag("tnum"), g.tnum)
	f.SetFeature(text.MustParseTag("smcp"), g.smcp)
	f.SetFeature(text.MustParseTag("zero"), g.zero)
	text.Draw(screen, sampleText, f, op)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Font Feature (Ebitengine Demo)")
	if err := ebiten.RunGame(&Game{}); err != nil {
		log.Fatal(err)
	}
}
