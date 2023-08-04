// Copyright 2014 Hajime Hoshi
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

package blocks

import (
	"image/color"
	"log"
	"strings"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/examples/resources/fonts"
	"github.com/hajimehoshi/ebiten/v2/text"
)

const (
	arcadeFontBaseSize = 8
)

var (
	arcadeFonts map[int]font.Face
)

func getArcadeFonts(scale int) font.Face {
	if arcadeFonts == nil {
		tt, err := opentype.Parse(fonts.PressStart2P_ttf)
		if err != nil {
			log.Fatal(err)
		}

		arcadeFonts = map[int]font.Face{}
		for i := 1; i <= 4; i++ {
			const dpi = 72
			arcadeFonts[i], err = opentype.NewFace(tt, &opentype.FaceOptions{
				Size:    float64(arcadeFontBaseSize * i),
				DPI:     dpi,
				Hinting: font.HintingFull,
			})
			if err != nil {
				log.Fatal(err)
			}
		}
	}
	return arcadeFonts[scale]
}

func textWidth(str string) int {
	maxW := 0
	for _, line := range strings.Split(str, "\n") {
		a := font.MeasureString(getArcadeFonts(1), line)
		w := a.Floor()
		if maxW < w {
			maxW = w
		}
	}
	return maxW
}

var (
	shadowColor = color.NRGBA{0, 0, 0, 0x80}
)

func drawTextWithShadow(rt *ebiten.Image, str string, x, y, scale int, clr color.Color) {
	offsetY := arcadeFontBaseSize * scale
	for _, line := range strings.Split(str, "\n") {
		y += offsetY
		text.Draw(rt, line, getArcadeFonts(scale), x+1, y+1, shadowColor)
		text.Draw(rt, line, getArcadeFonts(scale), x, y, clr)
	}
}

func drawTextWithShadowCenter(rt *ebiten.Image, str string, x, y, scale int, clr color.Color, width int) {
	w := textWidth(str) * scale
	x += (width - w) / 2
	drawTextWithShadow(rt, str, x, y, scale, clr)
}

func drawTextWithShadowRight(rt *ebiten.Image, str string, x, y, scale int, clr color.Color, width int) {
	w := textWidth(str) * scale
	x += width - w
	drawTextWithShadow(rt, str, x, y, scale, clr)
}
