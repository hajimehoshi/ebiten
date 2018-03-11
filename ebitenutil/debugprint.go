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
	debugPrintTextImage       *ebiten.Image
	debugPrintTextShadowImage *ebiten.Image
)

func init() {
	img := assets.CreateTextImage()
	debugPrintTextImage, _ = ebiten.NewImageFromImage(img, ebiten.FilterDefault)

	// Using color matrices for shadow color is not efficient.
	// Instead, use a different image, that shares the same texture in highly possibility.
	s := img.Bounds().Size()
	for j := 0; j < s.Y; j++ {
		for i := 0; i < s.X; i++ {
			idx := (img.Stride)*j + 4*i
			if img.Pix[idx+3] != 0 {
				img.Pix[idx] = 0
				img.Pix[idx+1] = 0
				img.Pix[idx+2] = 0
				img.Pix[idx+3] = 0x80
			}
		}
	}
	debugPrintTextShadowImage, _ = ebiten.NewImageFromImage(img, ebiten.FilterDefault)
}

// DebugPrint draws the string str on the image.
//
// The available runes are in U+0000 to U+00FF, which is C0 Controls and Basic Latin and C1 Controls and Latin-1 Supplement.
//
// DebugPrint always returns nil as of 1.5.0-alpha.
func DebugPrint(image *ebiten.Image, str string) error {
	drawDebugText(image, str, 1, 1, debugPrintTextShadowImage)
	drawDebugText(image, str, 0, 0, debugPrintTextImage)
	return nil
}

func drawDebugText(rt *ebiten.Image, str string, ox, oy int, src *ebiten.Image) {
	op := &ebiten.DrawImageOptions{}
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
		_ = rt.DrawImage(src, op)
		x += cw
	}
}
