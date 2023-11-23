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

// This example is a demonstration to render languages that cannot be rendered with the `text` package.
// We plan to provide a useful API to render them more easily (#2454). Stay tuned!

package main

import (
	"bytes"
	_ "embed"
	"image/color"
	"log"

	"golang.org/x/text/language"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/examples/resources/fonts"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

//go:embed NotoSansArabic-Regular.ttf
var arabicTTF []byte

var arabicFaceSource *text.GoTextFaceSource

func init() {
	s, err := text.NewGoTextFaceSource(bytes.NewReader(arabicTTF))
	if err != nil {
		log.Fatal(err)
	}
	arabicFaceSource = s
}

//go:embed NotoSansDevanagari-Regular.ttf
var devanagariTTF []byte

var devanagariFaceSource *text.GoTextFaceSource

func init() {
	s, err := text.NewGoTextFaceSource(bytes.NewReader(devanagariTTF))
	if err != nil {
		log.Fatal(err)
	}
	devanagariFaceSource = s
}

//go:embed NotoSansThai-Regular.ttf
var thaiTTF []byte

var thaiFaceSource *text.GoTextFaceSource

func init() {
	s, err := text.NewGoTextFaceSource(bytes.NewReader(thaiTTF))
	if err != nil {
		log.Fatal(err)
	}
	thaiFaceSource = s
}

var japaneseFaceSource *text.GoTextFaceSource

func init() {
	s, err := text.NewGoTextFaceSource(bytes.NewReader(fonts.MPlus1pRegular_ttf))
	if err != nil {
		log.Fatal(err)
	}
	japaneseFaceSource = s
}

const (
	screenWidth  = 640
	screenHeight = 480
)

type Game struct {
}

func (g *Game) Update() error {
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	gray := color.RGBA{0x80, 0x80, 0x80, 0xff}

	{
		const arabicText = "لمّا كان الاعتراف بالكرامة المتأصلة في جميع"
		f := &text.GoTextFace{
			Source:    arabicFaceSource,
			Direction: text.DirectionRightToLeft,
			Size:      24,
			Language:  language.Arabic,
		}
		x, y := screenWidth-20, 40
		w, h := text.Measure(arabicText, f, 0)
		// The left upper point is not x but x-w, since the text runs in the rigth-to-left direction.
		vector.DrawFilledRect(screen, float32(x)-float32(w), float32(y), float32(w), float32(h), gray, false)
		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(x), float64(y))
		text.Draw(screen, arabicText, f, op)
	}
	{
		const hindiText = "चूंकि मानव परिवार के सभी सदस्यों के जन्मजात गौरव और समान"
		f := &text.GoTextFace{
			Source:   devanagariFaceSource,
			Size:     24,
			Language: language.Hindi,
		}
		x, y := 20, 100
		w, h := text.Measure(hindiText, f, 0)
		vector.DrawFilledRect(screen, float32(x), float32(y), float32(w), float32(h), gray, false)
		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(x), float64(y))
		text.Draw(screen, hindiText, f, op)
	}
	{
		const thaiText = "โดยที่การไม่นำพาและการหมิ่นในคุณค่าของสิทธิมนุษยชน"
		f := &text.GoTextFace{
			Source:   thaiFaceSource,
			Size:     24,
			Language: language.Thai,
		}
		x, y := 20, 160
		w, h := text.Measure(thaiText, f, 0)
		vector.DrawFilledRect(screen, float32(x), float32(y), float32(w), float32(h), gray, false)
		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(x), float64(y))
		text.Draw(screen, thaiText, f, op)
	}
	{
		const japaneseText = "あのイーハトーヴォの\nすきとおった風、\n夏でも底に冷たさを\nもつ青いそら…"
		f := &text.GoTextFace{
			Source:    japaneseFaceSource,
			Direction: text.DirectionTopToBottomAndRightToLeft,
			Size:      24,
			Language:  language.Japanese,
		}
		const lineSpacing = 48
		x, y := screenWidth-20, 210
		w, h := text.Measure(japaneseText, f, lineSpacing)
		// The left upper point is not x but x-w, since the text runs in the rigth-to-left direction as the secondary direction.
		vector.DrawFilledRect(screen, float32(x)-float32(w), float32(y), float32(w), float32(h), gray, false)
		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(x), float64(y))
		op.LineSpacingInPixels = lineSpacing
		text.Draw(screen, japaneseText, f, op)
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Text I18N (Ebitengine Demo)")
	if err := ebiten.RunGame(&Game{}); err != nil {
		log.Fatal(err)
	}
}
