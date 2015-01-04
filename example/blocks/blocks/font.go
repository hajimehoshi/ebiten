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

type fontImageParts string

func (f fontImageParts) Len() int {
	return len(f)
}

func (f fontImageParts) Dst(i int) (x0, y0, x1, y1 int) {
	x := i - strings.LastIndex(string(f)[:i], "\n") - 1
	y := strings.Count(string(f)[:i], "\n")
	x *= charWidth
	y *= charHeight
	if x < 0 {
		return 0, 0, 0, 0
	}
	return x, y, x + charWidth, y + charHeight
}

func (f fontImageParts) Src(i int) (x0, y0, x1, y1 int) {
	code := int(f[i])
	if code == '\n' {
		return 0, 0, 0, 0
	}
	x := (code % 16) * charWidth
	y := ((code - 32) / 16) * charHeight
	return x, y, x + charWidth, y + charHeight
}

func drawText(rt *ebiten.Image, str string, ox, oy, scale int, c color.Color) error {
	options := &ebiten.DrawImageOptions{
		ImageParts: fontImageParts(str),
	}
	options.GeoM.Scale(float64(scale), float64(scale))
	options.GeoM.Translate(float64(ox), float64(oy))

	ur, ug, ub, ua := c.RGBA()
	const max = math.MaxUint16
	r := float64(ur) / max
	g := float64(ug) / max
	b := float64(ub) / max
	a := float64(ua) / max
	if 0 < a {
		r /= a
		g /= a
		b /= a
	}
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
