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
	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil/internal/assets"
	"image/color"
	"math"
	"strings"
)

type debugPrintImageParts string

func (f debugPrintImageParts) Len() int {
	return len(f)
}

func (f debugPrintImageParts) Dst(i int) (x0, y0, x1, y1 int) {
	cw, ch := assets.TextImageCharWidth, assets.TextImageCharHeight
	x := i - strings.LastIndex(string(f)[:i], "\n") - 1
	y := strings.Count(string(f)[:i], "\n")
	x *= cw
	y *= ch
	if x < 0 {
		return 0, 0, 0, 0
	}
	return x, y, x + cw, y + ch
}

func (f debugPrintImageParts) Src(i int) (x0, y0, x1, y1 int) {
	cw, ch := assets.TextImageCharWidth, assets.TextImageCharHeight
	const n = assets.TextImageWidth / assets.TextImageCharWidth
	code := int(f[i])
	if code == '\n' {
		return 0, 0, 0, 0
	}
	x := (code % n) * cw
	y := (code / n) * ch
	return x, y, x + cw, y + ch
}

type debugPrintState struct {
	textImage              *ebiten.Image
	debugPrintRenderTarget *ebiten.Image
}

var defaultDebugPrintState = &debugPrintState{}

// DebugPrint draws the string str on the image.
//
// DebugPrint always returns nil as of 1.5.0-alpha.
func DebugPrint(image *ebiten.Image, str string) error {
	defaultDebugPrintState.DebugPrint(image, str)
	return nil
}

func (d *debugPrintState) drawText(rt *ebiten.Image, str string, x, y int, c color.Color) {
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
	op := &ebiten.DrawImageOptions{
		ImageParts: debugPrintImageParts(str),
	}
	op.GeoM.Translate(float64(x+1), float64(y))
	op.ColorM.Scale(r, g, b, a)
	_ = rt.DrawImage(d.textImage, op)
}

// DebugPrint prints the given text str on the given image r.
func (d *debugPrintState) DebugPrint(r *ebiten.Image, str string) {
	if d.textImage == nil {
		img := assets.TextImage()
		d.textImage, _ = ebiten.NewImageFromImage(img, ebiten.FilterNearest)
	}
	if d.debugPrintRenderTarget == nil {
		width, height := 256, 256
		d.debugPrintRenderTarget, _ = ebiten.NewImage(width, height, ebiten.FilterNearest)
	}
	d.drawText(r, str, 1, 1, color.NRGBA{0x00, 0x00, 0x00, 0x80})
	d.drawText(r, str, 0, 0, color.NRGBA{0xff, 0xff, 0xff, 0xff})
}
