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

package common

import (
	"image"
	"image/color"
	"image/draw"
	"math"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"
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

func init() {
	dir := ""
	if runtime.GOARCH != "js" {
		// Get the path of this file (font.go).
		_, path, _, _ := runtime.Caller(0)
		path = filepath.Dir(path)
		dir = filepath.Join(path, "..")
	}
	arcadeFontPath := filepath.Join(dir, "_resources", "images", "arcadefont.png")

	arcadeFontImage, origImage, err := ebitenutil.NewImageFromFile(arcadeFontPath, ebiten.FilterNearest)
	if err != nil {
		panic(err)
	}
	ArcadeFont = &Font{arcadeFontImage, origImage, 32, 16, 8, 8}
}

type fontImageParts struct {
	str  string
	font *Font
}

func (f *fontImageParts) Len() int {
	return len(f.str)
}

func (f *fontImageParts) Dst(i int) (x0, y0, x1, y1 int) {
	x := i - strings.LastIndex(f.str[:i], "\n") - 1
	y := strings.Count(f.str[:i], "\n")
	x *= f.font.charWidth
	y *= f.font.charHeight
	if x < 0 {
		return 0, 0, 0, 0
	}
	return x, y, x + f.font.charWidth, y + f.font.charHeight
}

func (f *fontImageParts) Src(i int) (x0, y0, x1, y1 int) {
	code := int(f.str[i])
	if code == '\n' {
		return 0, 0, 0, 0
	}
	x := (code % f.font.charNumPerLine) * f.font.charWidth
	y := ((code - f.font.offset) / f.font.charNumPerLine) * f.font.charHeight
	return x, y, x + f.font.charWidth, y + f.font.charHeight
}

func (f *Font) DrawText(rt *ebiten.Image, str string, ox, oy, scale int, c color.Color) error {
	options := &ebiten.DrawImageOptions{
		ImageParts: &fontImageParts{str, f},
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

	return rt.DrawImage(f.image, options)
}

func (f *Font) DrawTextOnImage(rt draw.Image, str string, ox, oy int) error {
	parts := &fontImageParts{str, f}
	for i := 0; i < parts.Len(); i++ {
		dx0, dy0, dx1, dy1 := parts.Dst(i)
		sx0, sy0, _, _ := parts.Src(i)
		draw.Draw(rt, image.Rect(dx0+ox, dy0+oy, dx1+ox, dy1+oy), f.origImage, image.Pt(sx0, sy0), draw.Over)
	}
	return nil
}

func (f *Font) DrawTextWithShadow(rt *ebiten.Image, str string, x, y, scale int, clr color.Color) error {
	if err := f.DrawText(rt, str, x+1, y+1, scale, color.NRGBA{0, 0, 0, 0x80}); err != nil {
		return err
	}
	if err := f.DrawText(rt, str, x, y, scale, clr); err != nil {
		return err
	}
	return nil
}
