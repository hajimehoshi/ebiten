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

//go:generate go run gen.go

package ebitenutil

import (
	"bytes"
	_ "embed"
	"image"
	_ "image/png"

	"github.com/hajimehoshi/ebiten/v2"
)

//go:embed text.png
var text_png []byte

var (
	debugPrintTextImage     *ebiten.Image
	debugPrintTextSubImages = map[rune]*ebiten.Image{}
)

func init() {
	img, _, err := image.Decode(bytes.NewReader(text_png))
	if err != nil {
		panic(err)
	}
	debugPrintTextImage = ebiten.NewImageFromImage(img)
}

// DebugPrint draws the string str on the image at (0, 0) position (the upper-left corner in most cases).
//
// The available runes are in U+0000 to U+00FF, which is C0 Controls and Basic Latin and C1 Controls and Latin-1 Supplement.
func DebugPrint(image *ebiten.Image, str string) {
	DebugPrintAt(image, str, 0, 0)
}

// DebugPrintAt draws the string str on the image at (x, y) position.
//
// The available runes are in U+0000 to U+00FF, which is C0 Controls and Basic Latin and C1 Controls and Latin-1 Supplement.
func DebugPrintAt(image *ebiten.Image, str string, x, y int) {
	drawDebugText(image, str, x, y)
}

func drawDebugText(rt *ebiten.Image, str string, ox, oy int) {
	op := &ebiten.DrawImageOptions{}
	x := 0
	y := 0
	w := debugPrintTextImage.Bounds().Dx()
	for _, c := range str {
		const (
			cw = 6
			ch = 16
		)
		if c == '\n' {
			x = 0
			y += ch
			continue
		}
		s, ok := debugPrintTextSubImages[c]
		if !ok {
			n := w / cw
			sx := (int(c) % n) * cw
			sy := (int(c) / n) * ch
			s = debugPrintTextImage.SubImage(image.Rect(sx, sy, sx+cw, sy+ch)).(*ebiten.Image)
			debugPrintTextSubImages[c] = s
		}
		op.GeoM.Reset()
		op.GeoM.Translate(float64(x), float64(y))
		op.GeoM.Translate(float64(ox+1), float64(oy))
		rt.DrawImage(s, op)
		x += cw
	}
}
