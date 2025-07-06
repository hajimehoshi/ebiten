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
	"image/color"
	"log"

	"golang.org/x/text/language"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/examples/resources/fonts"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
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

//go:embed NotoSansMyanmar-Regular.ttf
var myanmarTTF []byte

var myanmarFaceSource *text.GoTextFaceSource

func init() {
	s, err := text.NewGoTextFaceSource(bytes.NewReader(myanmarTTF))
	if err != nil {
		log.Fatal(err)
	}
	myanmarFaceSource = s
}

//go:embed NotoSansMongolian-Regular.ttf
var mongolianTTF []byte

var mongolianFaceSource *text.GoTextFaceSource

func init() {
	s, err := text.NewGoTextFaceSource(bytes.NewReader(mongolianTTF))
	if err != nil {
		log.Fatal(err)
	}
	mongolianFaceSource = s
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
	screenHeight = 640
)

type Game struct {
	showOrigins bool
}

func (g *Game) Update() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyO) {
		g.showOrigins = !g.showOrigins
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	ebitenutil.DebugPrint(screen, "Press O to show/hide origins.\nRed points are the original origin positions.\nThe green points are the origin positions after applying the offset.")

	gray := color.RGBA{0x80, 0x80, 0x80, 0xff}

	{
		const arabicText = "لمّا كان الاعتراف بالكرامة المتأصلة في جميع"
		f := &text.GoTextFace{
			Source:    arabicFaceSource,
			Direction: text.DirectionRightToLeft,
			Size:      24,
			Language:  language.Arabic,
		}
		x, y := screenWidth-20, 50
		w, h := text.Measure(arabicText, f, 0)
		// The left upper point is not x but x-w, since the text runs in the rigth-to-left direction.
		vector.FillRect(screen, float32(x)-float32(w), float32(y), float32(w), float32(h), gray, false)
		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(x), float64(y))
		text.Draw(screen, arabicText, f, op)

		if g.showOrigins {
			op := &text.LayoutOptions{}
			for _, g := range text.AppendGlyphs(nil, arabicText, f, op) {
				vector.FillCircle(screen, float32(x)+float32(g.OriginX+g.OriginOffsetX), float32(y)+float32(g.OriginY+g.OriginOffsetY), 2, color.RGBA{0, 0xff, 0, 0xff}, true)
				vector.FillCircle(screen, float32(x)+float32(g.OriginX), float32(y)+float32(g.OriginY), 2, color.RGBA{0xff, 0, 0, 0xff}, true)
			}
		}
	}
	{
		const hindiText = "चूंकि मानव परिवार के सभी सदस्यों के जन्मजात गौरव और समान"
		f := &text.GoTextFace{
			Source:   devanagariFaceSource,
			Size:     24,
			Language: language.Hindi,
		}
		x, y := 20, 110
		w, h := text.Measure(hindiText, f, 0)
		vector.FillRect(screen, float32(x), float32(y), float32(w), float32(h), gray, false)
		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(x), float64(y))
		text.Draw(screen, hindiText, f, op)

		if g.showOrigins {
			op := &text.LayoutOptions{}
			for _, g := range text.AppendGlyphs(nil, hindiText, f, op) {
				vector.FillCircle(screen, float32(x)+float32(g.OriginX+g.OriginOffsetX), float32(y)+float32(g.OriginY+g.OriginOffsetY), 2, color.RGBA{0, 0xff, 0, 0xff}, true)
				vector.FillCircle(screen, float32(x)+float32(g.OriginX), float32(y)+float32(g.OriginY), 2, color.RGBA{0xff, 0, 0, 0xff}, true)
			}
		}
	}
	{
		const myanmarText = "လူခပ်သိမ်း၏ မျိုးရိုးဂုဏ်သိက္ခာနှင့်တကွ"
		f := &text.GoTextFace{
			Source:   myanmarFaceSource,
			Size:     24,
			Language: language.Burmese,
		}
		x, y := 20, 170
		w, h := text.Measure(myanmarText, f, 0)
		vector.FillRect(screen, float32(x), float32(y), float32(w), float32(h), gray, false)
		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(x), float64(y))
		text.Draw(screen, myanmarText, f, op)

		if g.showOrigins {
			op := &text.LayoutOptions{}
			for _, g := range text.AppendGlyphs(nil, myanmarText, f, op) {
				vector.FillCircle(screen, float32(x)+float32(g.OriginX+g.OriginOffsetX), float32(y)+float32(g.OriginY+g.OriginOffsetY), 2, color.RGBA{0, 0xff, 0, 0xff}, true)
				vector.FillCircle(screen, float32(x)+float32(g.OriginX), float32(y)+float32(g.OriginY), 2, color.RGBA{0xff, 0, 0, 0xff}, true)
			}
		}
	}
	{
		const thaiText = "โดยที่การยอมรับนับถือเกียรติศักดิ์ประจำตัว"
		f := &text.GoTextFace{
			Source:   thaiFaceSource,
			Size:     24,
			Language: language.Thai,
		}
		x, y := 20, 230
		w, h := text.Measure(thaiText, f, 0)
		vector.FillRect(screen, float32(x), float32(y), float32(w), float32(h), gray, false)
		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(x), float64(y))
		text.Draw(screen, thaiText, f, op)

		if g.showOrigins {
			op := &text.LayoutOptions{}
			for _, g := range text.AppendGlyphs(nil, thaiText, f, op) {
				vector.FillCircle(screen, float32(x)+float32(g.OriginX+g.OriginOffsetX), float32(y)+float32(g.OriginY+g.OriginOffsetY), 2, color.RGBA{0, 0xff, 0, 0xff}, true)
				vector.FillCircle(screen, float32(x)+float32(g.OriginX), float32(y)+float32(g.OriginY), 2, color.RGBA{0xff, 0, 0, 0xff}, true)
			}
		}
	}
	{
		const mongolianText = "ᠬᠦᠮᠦᠨ ᠪᠦᠷ ᠲᠥᠷᠥᠵᠦ ᠮᠡᠨᠳᠡᠯᠡᠬᠦ\nᠡᠷᠬᠡ ᠴᠢᠯᠥᠭᠡ ᠲᠡᠢ᠂ ᠠᠳᠠᠯᠢᠬᠠᠨ"
		f := &text.GoTextFace{
			Source:    mongolianFaceSource,
			Direction: text.DirectionTopToBottomAndLeftToRight,
			Size:      24,
			Language:  language.Mongolian,
			// language.Mongolian.Script() returns "Cyrl" (Cyrillic), but we want Mongolian script here.
			Script: language.MustParseScript("Mong"),
		}
		const lineSpacing = 48
		x, y := 20, 290
		w, h := text.Measure(mongolianText, f, lineSpacing)
		vector.FillRect(screen, float32(x), float32(y), float32(w), float32(h), gray, false)
		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(x), float64(y))
		op.LineSpacing = lineSpacing
		text.Draw(screen, mongolianText, f, op)

		if g.showOrigins {
			op := &text.LayoutOptions{}
			op.LineSpacing = lineSpacing
			for _, g := range text.AppendGlyphs(nil, mongolianText, f, op) {
				vector.FillCircle(screen, float32(x)+float32(g.OriginX+g.OriginOffsetX), float32(y)+float32(g.OriginY+g.OriginOffsetY), 2, color.RGBA{0, 0xff, 0, 0xff}, true)
				vector.FillCircle(screen, float32(x)+float32(g.OriginX), float32(y)+float32(g.OriginY), 2, color.RGBA{0xff, 0, 0, 0xff}, true)
			}
		}
	}
	{
		const japaneseText = "あのイーハトーヴォの\nすきとおった風、\n夏でも底に冷たさを\nもつ青いそら…\nあHello World.あ"
		f := &text.GoTextFace{
			Source:    japaneseFaceSource,
			Direction: text.DirectionTopToBottomAndRightToLeft,
			Size:      24,
			Language:  language.Japanese,
		}
		const lineSpacing = 48
		x, y := screenWidth-20, 290
		w, h := text.Measure(japaneseText, f, lineSpacing)
		// The left upper point is not x but x-w, since the text runs in the rigth-to-left direction as the secondary direction.
		vector.FillRect(screen, float32(x)-float32(w), float32(y), float32(w), float32(h), gray, false)
		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(x), float64(y))
		op.LineSpacing = lineSpacing
		text.Draw(screen, japaneseText, f, op)

		if g.showOrigins {
			op := &text.LayoutOptions{}
			op.LineSpacing = lineSpacing
			for _, g := range text.AppendGlyphs(nil, japaneseText, f, op) {
				vector.FillCircle(screen, float32(x)+float32(g.OriginX), float32(y)+float32(g.OriginY), 2, color.RGBA{0xff, 0, 0, 0xff}, true)
				vector.FillCircle(screen, float32(x)+float32(g.OriginX+g.OriginOffsetX), float32(y)+float32(g.OriginY+g.OriginOffsetY), 2, color.RGBA{0, 0xff, 0, 0xff}, true)
			}
		}
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
