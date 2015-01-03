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
	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"
	"image"
	"image/color"
	"math"
	"strings"
)

var imageFont *ebiten.Image

func init() {
	var err error
	imageFont, _, err = ebitenutil.NewImageFromFile("images/blocks/font.png", ebiten.FilterNearest)
	if err != nil {
		panic(err)
	}
}

const charWidth = 8
const charHeight = 8

func textWidth(str string) int {
	// TODO: Take care about '\n'
	return charWidth * len(str)
}

func drawText(rt *ebiten.Image, str string, ox, oy, scale int, c color.Color) error {
	parts := make([]ebiten.ImagePart, len(strings.Replace(str, "\n", "", -1)))

	locationX, locationY := 0, 0
	i := 0
	for _, c := range str {
		if c == '\n' {
			locationX = 0
			locationY += charHeight
			continue
		}
		code := int(c)
		x := (code % 16) * charWidth
		y := ((code - 32) / 16) * charHeight
		parts[i].Dst = image.Rect(locationX, locationY, locationX+charWidth, locationY+charHeight)
		parts[i].Src = image.Rect(x, y, x+charWidth, y+charHeight)
		i++
		locationX += charWidth
	}

	options := &ebiten.DrawImageOptions{
		Parts: parts,
	}
	options.GeoM.Scale(float64(scale), float64(scale))
	options.GeoM.Translate(float64(ox), float64(oy))

	c2 := color.NRGBA64Model.Convert(c).(color.NRGBA64)
	const max = math.MaxUint16
	r := float64(c2.R) / max
	g := float64(c2.G) / max
	b := float64(c2.B) / max
	a := float64(c2.A) / max
	options.ColorM.Scale(r, g, b, a)

	return rt.DrawImage(imageFont, options)
}

func drawTextWithShadow(rt *ebiten.Image, str string, x, y, scale int, clr color.Color) error {
	if err := drawText(rt, str, x+1, y+1, scale, color.NRGBA{0, 0, 0, 0x80}); err != nil {
		return err
	}
	if err := drawText(rt, str, x, y, scale, clr); err != nil {
		return err
	}
	return nil
}

func drawTextWithShadowCenter(rt *ebiten.Image, str string, x, y, scale int, clr color.Color, width int) error {
	w := textWidth(str) * scale
	x += (width - w) / 2
	return drawTextWithShadow(rt, str, x, y, scale, clr)
}

func drawTextWithShadowRight(rt *ebiten.Image, str string, x, y, scale int, clr color.Color, width int) error {
	w := textWidth(str) * scale
	x += width - w
	return drawTextWithShadow(rt, str, x, y, scale, clr)
}
