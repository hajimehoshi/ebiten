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
	"image"
	"log"

	"github.com/ebitengine/debugui"

	"github.com/hajimehoshi/ebiten/v2"
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
	debugui debugui.DebugUI

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
	if _, err := g.debugui.Update(func(ctx *debugui.Context) error {
		ctx.Window("Font Feature", image.Rect(10, 10, 210, 160), func(layout debugui.ContainerLayout) {
			var liga bool
			if g.liga == 1 {
				liga = true
			}
			ctx.Checkbox(&liga, "'liga' (Standard Ligatures)").On(func() {
				if liga {
					g.liga = 1
				} else {
					g.liga = 0
				}
			})

			var tnum bool
			if g.tnum == 1 {
				tnum = true
			}
			ctx.Checkbox(&tnum, "'tnum' (Tabular Figures)").On(func() {
				if tnum {
					g.tnum = 1
				} else {
					g.tnum = 0
				}
			})

			var smcp bool
			if g.smcp == 1 {
				smcp = true
			}
			ctx.Checkbox(&smcp, "'smcp' (Small Capitals)").On(func() {
				if smcp {
					g.smcp = 1
				} else {
					g.smcp = 0
				}
			})

			var zero bool
			if g.zero == 1 {
				zero = true
			}
			ctx.Checkbox(&zero, "'zero' (Slashed Zero)").On(func() {
				if zero {
					g.zero = 1
				} else {
					g.zero = 0
				}
			})
		})
		return nil
	}); err != nil {
		return err
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	// Draw the sample text.
	const sampleText = `0 (Number) / O (Alphabet)
ffi
3.14
2.71`
	op := &text.DrawOptions{}
	op.GeoM.Translate(20, screenHeight/2)
	op.LineSpacing = 50
	f := &text.GoTextFace{
		Source: firaSansFaceSource,
		Size:   40,
	}
	f.SetFeature(text.MustParseTag("liga"), g.liga)
	f.SetFeature(text.MustParseTag("tnum"), g.tnum)
	f.SetFeature(text.MustParseTag("smcp"), g.smcp)
	f.SetFeature(text.MustParseTag("zero"), g.zero)
	text.Draw(screen, sampleText, f, op)

	g.debugui.Draw(screen)
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
