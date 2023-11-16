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
	"bytes"
	"image/color"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/examples/resources/fonts"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

const (
	arcadeFontBaseSize = 8
)

var (
	arcadeFaceSource *text.GoTextFaceSource
)

func init() {
	s, err := text.NewGoTextFaceSource(bytes.NewReader(fonts.PressStart2P_ttf))
	if err != nil {
		log.Fatal(err)
	}
	arcadeFaceSource = s
}

var (
	shadowColor = color.RGBA{0, 0, 0, 0x80}
)

func drawTextWithShadow(rt *ebiten.Image, str string, x, y, scale int, clr color.Color, primaryAlign, secondaryAlign text.Align) {
	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(x)+1, float64(y)+1)
	op.ColorScale.ScaleWithColor(shadowColor)
	op.LineSpacingInPixels = arcadeFontBaseSize * float64(scale)
	op.PrimaryAlign = primaryAlign
	op.SecondaryAlign = secondaryAlign
	text.Draw(rt, str, &text.GoTextFace{
		Source: arcadeFaceSource,
		Size:   arcadeFontBaseSize * float64(scale),
	}, op)

	op.GeoM.Reset()
	op.GeoM.Translate(float64(x), float64(y))
	op.ColorScale.Reset()
	op.ColorScale.ScaleWithColor(clr)
	text.Draw(rt, str, &text.GoTextFace{
		Source: arcadeFaceSource,
		Size:   arcadeFontBaseSize * float64(scale),
	}, op)
}
