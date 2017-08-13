// Copyright 2017 The Ebiten Authors
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
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten"
)

var emptyImage *ebiten.Image

func init() {
	emptyImage, _ = ebiten.NewImage(16, 16, ebiten.FilterLinear)
	_ = emptyImage.Fill(color.White)
}

func colorScale(clr color.Color) (rf, gf, bf, af float64) {
	r, g, b, a := clr.RGBA()
	if a == 0 {
		return 0, 0, 0, 0
	}

	rf = float64(r) / float64(a)
	gf = float64(g) / float64(a)
	bf = float64(b) / float64(a)
	af = float64(a) / 0xffff
	return
}

// DrawLine draws a line on the given destination dst.
//
// DrawLine is intended to be used mainly for debugging or prototyping purpose.
func DrawLine(dst *ebiten.Image, x1, y1, x2, y2 float64, clr color.Color) {
	ew, eh := emptyImage.Size()
	length := math.Hypot(x2-x1, y2-y1)

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(length/float64(ew), 1/float64(eh))
	op.GeoM.Rotate(math.Atan2(y2-y1, x2-x1))
	op.GeoM.Translate(x1, y1)
	op.ColorM.Scale(colorScale(clr))
	_ = dst.DrawImage(emptyImage, op)
}

// DrawRect draws a rectangle on the given destination dst.
//
// DrawRect is intended to be used mainly for debugging or prototyping purpose.
func DrawRect(dst *ebiten.Image, x, y, width, height float64, clr color.Color) {
	ew, eh := emptyImage.Size()

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(width/float64(ew), height/float64(eh))
	op.GeoM.Translate(x, y)
	op.ColorM.Scale(colorScale(clr))
	_ = dst.DrawImage(emptyImage, op)
}
