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

//go:embed RobotoFlex.ttf
var robotoFlexRegular []byte

var robotoFlexFaceSource *text.GoTextFaceSource

func init() {
	s, err := text.NewGoTextFaceSource(bytes.NewReader(robotoFlexRegular))
	if err != nil {
		log.Fatal(err)
	}
	robotoFlexFaceSource = s
}

type Game struct {
	// wght represents 'Weight'.
	// https://learn.microsoft.com/en-us/typography/opentype/spec/dvaraxistag_wght
	wght float32

	// wdth represents 'Width'.
	// https://learn.microsoft.com/en-us/typography/opentype/spec/dvaraxistag_wdth
	wdth float32

	// slnt represents 'Slant'.
	// https://learn.microsoft.com/en-us/typography/opentype/spec/dvaraxistag_slnt
	slnt float32
}

func NewGame() *Game {
	return &Game{
		wght: 400,
		wdth: 100,
		slnt: 0,
	}
}

const (
	minWght = 100
	maxWght = 1000
	minWdth = 30
	maxWdth = 150
	minSlnt = -10
	maxSlnt = 0
)

func (g *Game) Update() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyQ) {
		if g.wght > minWght {
			g.wght -= 100
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyW) {
		if g.wght < maxWght {
			g.wght += 100
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyA) {
		if g.wdth > minWdth {
			g.wdth -= 10
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyS) {
		if g.wdth < maxWdth {
			g.wdth += 10
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyZ) {
		if g.slnt > minSlnt {
			g.slnt -= 1
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyX) {
		if g.slnt < maxSlnt {
			g.slnt += 1
		}
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	// Draw the instruction.
	inst := fmt.Sprintf(`Press keys to adjust font variations.
[Q, W]: wght (Weight): %0.0f [%d-%d]
[A, S]: wdth (Width): %0.0f [%d-%d]
[Z, X]: slnt (Slant): %0.0f [%d-%d]`, g.wght, minWght, maxWght, g.wdth, minWdth, maxWdth, g.slnt, minSlnt, maxSlnt)
	op := &text.DrawOptions{}
	op.GeoM.Translate(20, 20)
	op.LineSpacingInPixels = 30
	text.Draw(screen, inst, &text.GoTextFace{
		Source: robotoFlexFaceSource,
		Size:   20,
	}, op)

	// Draw the sample text.
	const sampleText = `The quick brown fox jumps
over the lazy dog.`
	op = &text.DrawOptions{}
	op.GeoM.Translate(20, screenHeight/2)
	op.LineSpacingInPixels = 50
	f := &text.GoTextFace{
		Source: robotoFlexFaceSource,
		Size:   40,
	}
	f.SetVariation(text.MustParseTag("wght"), g.wght)
	f.SetVariation(text.MustParseTag("wdth"), g.wdth)
	f.SetVariation(text.MustParseTag("slnt"), g.slnt)
	text.Draw(screen, sampleText, f, op)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Font Variation (Ebitengine Demo)")
	if err := ebiten.RunGame(NewGame()); err != nil {
		log.Fatal(err)
	}
}
