// Copyright 2015 Hajime Hoshi
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

// +build example

package common

import (
	"image"
	"image/color"
	"image/draw"
	"math"

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/examples/common/internal/assets"
)

var (
	ArcadeFont *Font
)

type Font struct {
	image          *ebiten.Image
	origImage      image.Image
	offset         int
	charNumPerLine int
	charWidth      int
	charHeight     int
}

func (f *Font) TextWidth(str string) int {
	// TODO: Take care about '\n'
	return f.charWidth * len(str)
}

func (f *Font) TextHeight(str string) int {
	// TODO: Take care about '\n'
	return f.charHeight
}

func init() {
	img := assets.ArcadeFontImage()
	eimg, _ := ebiten.NewImageFromImage(img, ebiten.FilterNearest)
	ArcadeFont = &Font{eimg, img, 32, 16, 8, 8}
}

type part struct {
	sx, sy, dx0, dy0, dx1, dy1 int
}

func (f *Font) parts(str string) []part {
	ps := []part{}
	x := 0
	y := 0
	for _, c := range str {
		if c == '\n' {
			x = 0
			y += f.charHeight
			continue
		}
		sx := (int(c) % f.charNumPerLine) * f.charWidth
		sy := ((int(c) - f.offset) / f.charNumPerLine) * f.charHeight
		dx0 := x
		dy0 := y
		dx1 := dx0 + f.charWidth
		dy1 := dy0 + f.charHeight
		ps = append(ps, part{sx, sy, dx0, dy0, dx1, dy1})
		x += f.charWidth
	}
	return ps
}

func (f *Font) DrawText(rt *ebiten.Image, str string, ox, oy, scale int, c color.Color) {
	op := &ebiten.DrawImageOptions{}
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
	op.ColorM.Scale(r, g, b, a)

	// TODO: There is same logic in parts. Refactor this.
	x := 0
	y := 0
	for _, c := range str {
		if c == '\n' {
			x = 0
			y += f.charHeight
			continue
		}
		sx := (int(c) % f.charNumPerLine) * f.charWidth
		sy := ((int(c) - f.offset) / f.charNumPerLine) * f.charHeight
		r := image.Rect(sx, sy, sx+f.charWidth, sy+f.charHeight)
		op.SourceRect = &r
		op.GeoM.Reset()
		op.GeoM.Translate(float64(x), float64(y))
		op.GeoM.Scale(float64(scale), float64(scale))
		op.GeoM.Translate(float64(ox), float64(oy))
		rt.DrawImage(f.image, op)
		x += f.charWidth
	}
}

func (f *Font) DrawTextOnImage(rt draw.Image, str string, ox, oy int) {
	// TODO: This function is needed only by examples/keyboard/keyboard.
	// This is executed without Ebiten, so ebiten.Image can't be used.
	// When ebiten.Image can be used without ebiten.Run, this function can be removed.
	for _, p := range f.parts(str) {
		draw.Draw(rt, image.Rect(p.dx0+ox, p.dy0+oy, p.dx1+ox, p.dy1+oy),
			f.origImage, image.Pt(p.sx, p.sy), draw.Over)
	}
}

func (f *Font) DrawTextWithShadow(rt *ebiten.Image, str string, x, y, scale int, clr color.Color) {
	f.DrawText(rt, str, x+1, y+1, scale, color.NRGBA{0, 0, 0, 0x80})
	f.DrawText(rt, str, x, y, scale, clr)
}
