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
	debugui debugui.DebugUI

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

func (g *Game) Update() error {
	const (
		minWght = 100
		maxWght = 1000
		minWdth = 30
		maxWdth = 150
		minSlnt = -10
		maxSlnt = 0
	)

	if _, err := g.debugui.Update(func(ctx *debugui.Context) error {
		ctx.Window("Font Variation", image.Rect(10, 10, 310, 160), func(layout debugui.ContainerLayout) {
			ctx.SetGridLayout([]int{-1, -2}, nil)
			ctx.Text("wght (Weight)")
			wght := float64(g.wght)
			ctx.SliderF(&wght, minWght, maxWght, 100, 0).On(func() {
				g.wght = float32(wght)
			})
			ctx.Text("wdth (Width)")
			wdth := float64(g.wdth)
			ctx.SliderF(&wdth, minWdth, maxWdth, 10, 0).On(func() {
				g.wdth = float32(wdth)
			})
			ctx.Text("slnt (Slant)")
			slnt := float64(g.slnt)
			ctx.SliderF(&slnt, minSlnt, maxSlnt, 1, 0).On(func() {
				g.slnt = float32(slnt)
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
	const sampleText = `The quick brown fox jumps
over the lazy dog.`
	op := &text.DrawOptions{}
	op.GeoM.Translate(20, screenHeight/2)
	op.LineSpacing = 50
	f := &text.GoTextFace{
		Source: robotoFlexFaceSource,
		Size:   40,
	}
	f.SetVariation(text.MustParseTag("wght"), g.wght)
	f.SetVariation(text.MustParseTag("wdth"), g.wdth)
	f.SetVariation(text.MustParseTag("slnt"), g.slnt)
	text.Draw(screen, sampleText, f, op)

	g.debugui.Draw(screen)
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
