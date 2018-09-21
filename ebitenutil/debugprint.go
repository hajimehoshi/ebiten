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

package ebitenutil

import (
	"image"

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil/internal/assets"
)

var (
	debugPrintTextImage *ebiten.Image
)

func init() {
	img := assets.CreateTextImage()
	debugPrintTextImage, _ = ebiten.NewImageFromImage(img, ebiten.FilterDefault)
}

// DebugPrint draws the string str on the image on left top corner.
//
// The available runes are in U+0000 to U+00FF, which is C0 Controls and Basic Latin and C1 Controls and Latin-1 Supplement.
//
// DebugPrint always returns nil as of 1.5.0-alpha.
func DebugPrint(image *ebiten.Image, str string) error {
	return DebugPrintAt(image, str, 0, 0)
}


// DebugPrintAt draws the string str on the image at X and Y coordinates.
//
// The available runes are in U+0000 to U+00FF, which is C0 Controls and Basic Latin and C1 Controls and Latin-1 Supplement.
//
// DebugPrintAt always returns nil as of 1.5.0-alpha.
func DebugPrintAt(image *ebiten.Image, str string, x, y int) error {
	drawDebugText(image, str, x+1, y+1, true)
	drawDebugText(image, str, x, y, false)
	return nil
}

func drawDebugText(rt *ebiten.Image, str string, ox, oy int, shadow bool) {
	op := &ebiten.DrawImageOptions{}
	if shadow {
		op.ColorM.Scale(0, 0, 0, 0.5)
	}
	x := 0
	y := 0
	w, _ := debugPrintTextImage.Size()
	var r image.Rectangle
	for _, c := range str {
		const (
			cw = assets.CharWidth
			ch = assets.CharHeight
		)
		if c == '\n' {
			x = 0
			y += ch
			continue
		}
		n := w / cw
		sx := (int(c) % n) * cw
		sy := (int(c) / n) * ch
		r.Min.X = sx
		r.Min.Y = sy
		r.Max.X = sx + cw
		r.Max.Y = sy + ch
		op.SourceRect = &r
		op.GeoM.Reset()
		op.GeoM.Translate(float64(x), float64(y))
		op.GeoM.Translate(float64(ox+1), float64(oy))
		_ = rt.DrawImage(debugPrintTextImage, op)
		x += cw
	}
}
